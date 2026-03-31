package vector

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

// QdrantConfig HTTP 检索参数。
type QdrantConfig struct {
	BaseURL        string // 如 http://127.0.0.1:6333
	Collection     string
	APIKey         string
	ScoreThreshold float64
	Limit          int
	HTTPClient     *http.Client
}

// QdrantStore 最小 REST 检索（points/search）。
type QdrantStore struct {
	cfg QdrantConfig
}

// NewQdrant 构造；collection 非空；threshold 默认 0.85；limit 默认 1。
func NewQdrant(cfg QdrantConfig) *QdrantStore {
	if cfg.Limit <= 0 {
		cfg.Limit = 1
	}
	if cfg.ScoreThreshold <= 0 {
		cfg.ScoreThreshold = 0.85
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = &http.Client{Timeout: 5 * time.Second}
	}
	return &QdrantStore{cfg: cfg}
}

type qdrantSearchReq struct {
	Vector []float64 `json:"vector"`
	Limit  int       `json:"limit"`
	WithPayload bool `json:"with_payload"`
}

type qdrantSearchResp struct {
	Result []struct {
		Score   float64           `json:"score"`
		Payload map[string]any    `json:"payload"`
	} `json:"result"`
	Status string `json:"status"`
}

func (q *QdrantStore) Search(ctx context.Context, in SearchInput) (SearchResult, bool, error) {
	if q == nil || len(in.Vector) == 0 {
		return SearchResult{}, false, nil
	}
	base := strings.TrimRight(strings.TrimSpace(q.cfg.BaseURL), "/")
	if base == "" || strings.TrimSpace(q.cfg.Collection) == "" {
		return SearchResult{}, false, fmt.Errorf("vector: qdrant url/collection 为空")
	}
	url := fmt.Sprintf("%s/collections/%s/points/search", base, strings.TrimSpace(q.cfg.Collection))

	body := qdrantSearchReq{
		Vector:      in.Vector,
		Limit:       q.cfg.Limit,
		WithPayload: true,
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return SearchResult{}, false, err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return SearchResult{}, false, err
	}
	req.Header.Set("Content-Type", "application/json")
	if k := strings.TrimSpace(q.cfg.APIKey); k != "" {
		req.Header.Set("api-key", k)
	}

	res, err := q.cfg.HTTPClient.Do(req)
	if err != nil {
		return SearchResult{}, false, err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return SearchResult{}, false, err
	}
	if res.StatusCode != http.StatusOK {
		return SearchResult{}, false, fmt.Errorf("vector: qdrant HTTP %d: %s", res.StatusCode, string(bytes.TrimSpace(b)))
	}
	var out qdrantSearchResp
	if err := json.Unmarshal(b, &out); err != nil {
		return SearchResult{}, false, err
	}
	if len(out.Result) == 0 {
		return SearchResult{}, false, nil
	}
	hit := out.Result[0]
	if hit.Score < q.cfg.ScoreThreshold {
		return SearchResult{}, false, nil
	}
	text := ""
	if hit.Payload != nil {
		if v, ok := hit.Payload["text"]; ok {
			switch t := v.(type) {
			case string:
				text = strings.TrimSpace(t)
			}
		}
	}
	if text == "" {
		return SearchResult{}, false, nil
	}
	kind := "cache"
	if hit.Payload != nil {
		if v, ok := hit.Payload["source"]; ok {
			if s, _ := v.(string); strings.TrimSpace(s) == "rag_writeback" {
				kind = "dedup"
			}
		}
	}
	return SearchResult{Text: text, Score: hit.Score, HitKind: kind}, true, nil
}

// WriteAnswerInput 持久化语义去重写回（Qdrant upsert）。
type WriteAnswerInput struct {
	Vector  []float64
	Text    string
	Query   string
	TraceID string
}

type qdrantUpsertReq struct {
	Points []qdrantPointUpsert `json:"points"`
}

type qdrantPointUpsert struct {
	ID      any            `json:"id"`
	Vector  []float64      `json:"vector"`
	Payload map[string]any `json:"payload"`
}

// UpsertWriteAnswer 将 RAG 成功结果写入当前 collection（payload.source=rag_writeback）。
func (q *QdrantStore) UpsertWriteAnswer(ctx context.Context, in WriteAnswerInput) error {
	if q == nil || len(in.Vector) == 0 || strings.TrimSpace(in.Text) == "" {
		return nil
	}
	base := strings.TrimRight(strings.TrimSpace(q.cfg.BaseURL), "/")
	if base == "" || strings.TrimSpace(q.cfg.Collection) == "" {
		return fmt.Errorf("vector: qdrant url/collection 为空")
	}
	id := uuid.NewString()
	payload := map[string]any{
		"text":       strings.TrimSpace(in.Text),
		"source":     "rag_writeback",
		"query":      strings.TrimSpace(in.Query),
		"trace_id":   strings.TrimSpace(in.TraceID),
		"created_at": time.Now().UTC().Format(time.RFC3339Nano),
	}
	body := qdrantUpsertReq{
		Points: []qdrantPointUpsert{
			{ID: id, Vector: in.Vector, Payload: payload},
		},
	}
	raw, err := json.Marshal(body)
	if err != nil {
		return err
	}
	url := fmt.Sprintf("%s/collections/%s/points?wait=true", base, strings.TrimSpace(q.cfg.Collection))
	req, err := http.NewRequestWithContext(ctx, http.MethodPut, url, bytes.NewReader(raw))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	if k := strings.TrimSpace(q.cfg.APIKey); k != "" {
		req.Header.Set("api-key", k)
	}
	res, err := q.cfg.HTTPClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	b, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	if res.StatusCode != http.StatusOK {
		return fmt.Errorf("vector: qdrant upsert HTTP %d: %s", res.StatusCode, string(bytes.TrimSpace(b)))
	}
	return nil
}
