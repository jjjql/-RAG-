package httpnb

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strings"
	"time"

	"rag-gateway/internal/circuitbreaker"
	"rag-gateway/internal/downstream"
	"rag-gateway/internal/embedding"
	"rag-gateway/internal/observability"
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

	if d.Embedder != nil {
		emb0 := time.Now()
		_, err := d.Embedder.Embed(r.Context(), embedding.EmbedInput{
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
		// 向量检索（L3）未接库时：嵌入成功后仍可走下游 RAG（NFR-P02 子阶段后续拆分）
	}

	if d.Downstream != nil {
		observability.ObserveQAPhase("rag_prep", time.Since(qaStart))
		sid := ""
		if body.SessionID != nil {
			sid = strings.TrimSpace(*body.SessionID)
		}
		text, err := d.Downstream.Complete(r.Context(), downstream.AnswerInput{
			Query:     body.Query,
			TraceID:   tid,
			SessionID: sid,
		})
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
