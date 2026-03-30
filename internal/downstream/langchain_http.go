package downstream

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// LangChainHTTPConfig 南向 LangChain HTTP 客户端配置（见 contracts/HTTP_LANGCHAIN_DOWNSTREAM.md）。
type LangChainHTTPConfig struct {
	BaseURL        string
	Path           string // 须以 / 开头，默认 /v1/rag/invoke
	APIKeyHeader   string
	APIKey         string
	HTTPClient     *http.Client // 可选；外层 Complete 仍以 context 超时为准
}

// LangChainHTTP 调用 LangChain 暴露的 invoke JSON API。
type LangChainHTTP struct {
	cfg LangChainHTTPConfig
	cli *http.Client
}

// NewLangChainHTTP 构造客户端。
func NewLangChainHTTP(c LangChainHTTPConfig) *LangChainHTTP {
	cli := c.HTTPClient
	if cli == nil {
		cli = &http.Client{Timeout: 120 * time.Second}
	}
	return &LangChainHTTP{cfg: c, cli: cli}
}

type invokeRequest struct {
	Query     string `json:"query"`
	SessionID string `json:"sessionId,omitempty"`
	TraceID   string `json:"traceId,omitempty"`
}

type invokeResponse struct {
	Answer        string `json:"answer"`
	Explanation   string `json:"explanation,omitempty"`
}

// Answer 实现 Answerer。
func (l *LangChainHTTP) Answer(ctx context.Context, in AnswerInput) (string, error) {
	if l == nil {
		return "", fmt.Errorf("downstream: LangChainHTTP 未初始化")
	}
	path := l.cfg.Path
	if path == "" {
		path = "/v1/rag/invoke"
	}
	if !strings.HasPrefix(path, "/") {
		return "", fmt.Errorf("downstream: http_path 须以 / 开头")
	}
	base := strings.TrimRight(strings.TrimSpace(l.cfg.BaseURL), "/")
	if base == "" {
		return "", fmt.Errorf("downstream: http_base_url 不能为空")
	}
	url := base + path

	payload := invokeRequest{
		Query:     in.Query,
		SessionID: strings.TrimSpace(in.SessionID),
		TraceID:   strings.TrimSpace(in.TraceID),
	}
	raw, err := json.Marshal(payload)
	if err != nil {
		return "", err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(raw))
	if err != nil {
		return "", err
	}
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	h := strings.TrimSpace(l.cfg.APIKeyHeader)
	k := strings.TrimSpace(l.cfg.APIKey)
	if h != "" && k != "" {
		req.Header.Set(h, k)
	}

	res, err := l.cli.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("langchain: HTTP %d: %s", res.StatusCode, string(bytes.TrimSpace(body)))
	}
	var ir invokeResponse
	if err := json.Unmarshal(body, &ir); err != nil {
		return "", fmt.Errorf("langchain: 解析 JSON: %w", err)
	}
	ans := strings.TrimSpace(ir.Answer)
	if ans == "" {
		return "", fmt.Errorf("langchain: 响应缺少非空 answer")
	}
	if exp := strings.TrimSpace(ir.Explanation); exp != "" {
		ans = ans + "\n\n" + exp
	}
	return ans, nil
}
