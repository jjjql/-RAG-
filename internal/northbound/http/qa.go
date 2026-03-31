package httpnb

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"strings"
	"time"

	"rag-gateway/internal/circuitbreaker"
	"rag-gateway/internal/coalesce"
	"rag-gateway/internal/downstream"
	"rag-gateway/internal/embedding"
	"rag-gateway/internal/observability"
	"rag-gateway/internal/vector"
)

// qaRequest 与 OpenAPI QARequest 对齐。
type qaRequest struct {
	Query     string  `json:"query"`
	Key       *string `json:"key,omitempty"`
	SessionID *string `json:"sessionId,omitempty"`
	Scope     *string `json:"scope,omitempty"`
}

func (d *Deps) handleQA(w http.ResponseWriter, r *http.Request) {
	tid := writeTraceID(w, r)
	if r.Method != http.MethodPost {
		writeJSONError(w, http.StatusMethodNotAllowed, tid, "METHOD_NOT_ALLOWED", http.StatusText(http.StatusMethodNotAllowed))
		return
	}
	accept := r.Header.Get("Accept")
	if !strings.Contains(accept, "text/event-stream") {
		writeJSONError(w, http.StatusNotAcceptable, tid, "NOT_ACCEPTABLE", "Accept 须包含 text/event-stream")
		return
	}
	var body qaRequest
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, tid, "INVALID_JSON", "请求体不是合法 JSON")
		return
	}
	if strings.TrimSpace(body.Query) == "" {
		writeJSONError(w, http.StatusBadRequest, tid, "INVALID_QUERY", "query 不能为空")
		return
	}

	qaStart := time.Now()

	// trace 与 SSE 头须在 WriteHeader 之前写入
	initSSE(w)

	meta, _ := json.Marshal(map[string]string{"traceId": tid})
	_ = writeSSE(w, "meta", string(meta))

	// OpenAPI：仅当客户端提供 key 时走精确 (scope, key)；未传 key 则跳过精确进入正则/向量/RAG。
	if body.Key != nil && strings.TrimSpace(*body.Key) != "" {
		k := strings.TrimSpace(*body.Key)
		if id, dat, ok := d.Exact.MatchExact(body.Scope, k); ok {
			delta, _ := json.Marshal(map[string]string{"text": dat})
			if err := writeSSE(w, "delta", string(delta)); err != nil {
				return
			}
			done, _ := json.Marshal(struct {
				Source  string `json:"source"`
				RuleID  string `json:"ruleId"`
				TraceID string `json:"traceId"`
			}{Source: "rule_exact", RuleID: id, TraceID: tid})
			_ = writeSSE(w, "done", string(done))
			observability.RecordQA("rule_exact")
			return
		}
	}

	if d.Regex != nil {
		if id, dat, ok := d.Regex.MatchRegex(body.Scope, body.Query); ok {
			delta, _ := json.Marshal(map[string]string{"text": dat})
			if err := writeSSE(w, "delta", string(delta)); err != nil {
				return
			}
			done, _ := json.Marshal(struct {
				Source  string `json:"source"`
				RuleID  string `json:"ruleId"`
				TraceID string `json:"traceId"`
			}{Source: "rule_regex", RuleID: id, TraceID: tid})
			_ = writeSSE(w, "done", string(done))
			observability.RecordQA("rule_regex")
			return
		}
	}

	mergeKey := coalesce.Key(body.Scope, body.Query)
	needEmbed := d.Embedder != nil &&
		((d.VectorEnabled && d.Vector != nil) ||
			(d.Downstream != nil && d.Semantic != nil) ||
			d.DedupWriter != nil ||
			d.DedupVector != nil)

	var embVec []float64
	if needEmbed {
		emb0 := time.Now()
		embRes, err := d.Embedder.Embed(r.Context(), embedding.EmbedInput{
			Text:    body.Query,
			TraceID: tid,
		})
		observability.ObserveQAPhase("embed", time.Since(emb0))
		if err != nil {
			observability.RecordQA("error_embedding")
			code := "EMBEDDING_FAILED"
			msg := err.Error()
			var se *embedding.ServerError
			if errors.As(err, &se) {
				msg = se.Message
			}
			if errors.Is(err, circuitbreaker.ErrOpen) {
				code = "EMBEDDING_CIRCUIT_OPEN"
				msg = "侧车熔断开路，请稍后重试"
			}
			if errors.Is(err, context.DeadlineExceeded) {
				code = "EMBEDDING_TIMEOUT"
				msg = "侧车嵌入超时或已取消"
			}
			errBody := ErrorBody{Code: code, Message: msg, TraceID: tid}
			b, _ := json.Marshal(errBody)
			_ = writeSSE(w, "error", string(b))
			return
		}
		if embRes != nil && len(embRes.Embedding) > 0 {
			embVec = embRes.Embedding
		}
	}

	if d.VectorEnabled && d.Vector != nil && len(embVec) > 0 {
		v0 := time.Now()
		sr, hit, verr := d.Vector.Search(r.Context(), vector.SearchInput{
			Vector:  embVec,
			TraceID: tid,
		})
		observability.ObserveQAPhase("vector", time.Since(v0))
		if verr != nil {
			if errors.Is(verr, circuitbreaker.ErrOpen) {
				observability.RecordQA("error_vector_circuit")
				errBody := ErrorBody{
					Code:    "VECTOR_CIRCUIT_OPEN",
					Message: "向量检索熔断开路，请稍后重试",
					TraceID: tid,
				}
				b, _ := json.Marshal(errBody)
				_ = writeSSE(w, "error", string(b))
				return
			}
			log.Printf("向量检索失败，降级走 RAG: traceId=%s err=%v", tid, verr)
		} else if hit {
			delta, _ := json.Marshal(map[string]string{"text": sr.Text})
			if err := writeSSE(w, "delta", string(delta)); err != nil {
				return
			}
			src := "semantic_cache"
			if sr.HitKind == "dedup" {
				src = "semantic_dedup"
			}
			done, _ := json.Marshal(struct {
				Source  string `json:"source"`
				TraceID string `json:"traceId"`
			}{Source: src, TraceID: tid})
			_ = writeSSE(w, "done", string(done))
			observability.RecordQA(src)
			return
		}
	}

	if d.DedupVector != nil && len(embVec) > 0 {
		d0 := time.Now()
		sr2, hit2, derr := d.DedupVector.Search(r.Context(), vector.SearchInput{
			Vector:  embVec,
			TraceID: tid,
		})
		observability.ObserveQAPhase("vector_dedup", time.Since(d0))
		if derr != nil {
			if errors.Is(derr, circuitbreaker.ErrOpen) {
				observability.RecordQA("error_vector_circuit")
				errBody := ErrorBody{
					Code:    "VECTOR_CIRCUIT_OPEN",
					Message: "持久化语义去重检索熔断开路，请稍后重试",
					TraceID: tid,
				}
				b, _ := json.Marshal(errBody)
				_ = writeSSE(w, "error", string(b))
				return
			}
			log.Printf("持久化语义去重检索失败，降级走 RAG: traceId=%s err=%v", tid, derr)
		} else if hit2 {
			delta, _ := json.Marshal(map[string]string{"text": sr2.Text})
			if err := writeSSE(w, "delta", string(delta)); err != nil {
				return
			}
			done, _ := json.Marshal(struct {
				Source  string `json:"source"`
				TraceID string `json:"traceId"`
			}{Source: "semantic_dedup", TraceID: tid})
			_ = writeSSE(w, "done", string(done))
			observability.RecordQA("semantic_dedup")
			return
		}
	}

	if d.Downstream != nil {
		observability.ObserveQAPhase("rag_prep", time.Since(qaStart))
		sid := ""
		if body.SessionID != nil {
			sid = strings.TrimSpace(*body.SessionID)
		}
		fn := func(ctx context.Context) (string, error) {
			return d.Downstream.Complete(ctx, downstream.AnswerInput{
				Query:     body.Query,
				TraceID:   tid,
				SessionID: sid,
			})
		}
		c0 := time.Now()
		var text string
		var err error
		if d.Semantic != nil && len(embVec) > 0 {
			// 语义相似合并：同 scope 下 embedding 余弦 ≥ 阈值则共用一次 RAG（见 coalesce.Semantic）。
			text, err = d.Semantic.Merge(r.Context(), mergeKey, embVec, fn)
		} else {
			// 文本规范化键合并：同 scope + 规范化 query（coalesce.enabled 时）。
			text, err = d.Coalesce.Do(r.Context(), mergeKey, fn)
		}
		observability.ObserveQAPhase("coalesce", time.Since(c0))
		if err != nil {
			observability.RecordQA("error_rag")
			code := "智能问答失败"
			msg := err.Error()
			if errors.Is(err, circuitbreaker.ErrOpen) {
				code = "RAG_CIRCUIT_OPEN"
				msg = "智能问答熔断开路，请稍后重试"
			}
			if errors.Is(err, context.DeadlineExceeded) {
				code = "智能问答超时"
				msg = "调用下游智能问答超时或已取消"
			}
			errBody := ErrorBody{Code: code, Message: msg, TraceID: tid}
			b, _ := json.Marshal(errBody)
			_ = writeSSE(w, "error", string(b))
			return
		}
		delta, _ := json.Marshal(map[string]string{"text": text})
		if err := writeSSE(w, "delta", string(delta)); err != nil {
			return
		}
		done, _ := json.Marshal(struct {
			Source  string `json:"source"`
			TraceID string `json:"traceId"`
		}{Source: "rag", TraceID: tid})
		_ = writeSSE(w, "done", string(done))
		observability.RecordQA("rag")
		if d.DedupWriter != nil && len(embVec) > 0 && strings.TrimSpace(text) != "" {
			q := body.Query
			go func(query, answer, trace string, vec []float64) {
				ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
				defer cancel()
				if err := d.DedupWriter.WriteAnswer(ctx, vector.WriteAnswerInput{
					Vector:  vec,
					Text:    answer,
					Query:   query,
					TraceID: trace,
				}); err != nil {
					log.Printf("持久化语义去重写回失败 traceId=%s: %v", trace, err)
				}
			}(q, text, tid, embVec)
		}
		return
	}

	observability.RecordQA("error_no_match")
	errBody := ErrorBody{
		Code:    "NO_MATCH",
		Message: "未命中规则且未启用智能问答下游（请配置 downstream.enabled）",
		TraceID: tid,
	}
	b, _ := json.Marshal(errBody)
	_ = writeSSE(w, "error", string(b))
}
