package downstream

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLangChainHTTP_Answer_OK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/v1/rag/invoke", r.URL.Path)
		var req invokeRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&req))
		assert.Equal(t, "hello", req.Query)
		assert.Equal(t, "s1", req.SessionID)
		assert.Equal(t, "tid", req.TraceID)
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(invokeResponse{Answer: "world", Explanation: "because"})
	}))
	defer srv.Close()

	lc := NewLangChainHTTP(LangChainHTTPConfig{
		BaseURL: srv.URL,
		Path:    "/v1/rag/invoke",
	})
	out, err := lc.Answer(context.Background(), AnswerInput{Query: "hello", SessionID: "s1", TraceID: "tid"})
	require.NoError(t, err)
	assert.Contains(t, out, "world")
	assert.Contains(t, out, "because")
}

func TestLangChainHTTP_Answer_HTTPError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusBadGateway)
		_, _ = w.Write([]byte(`{"message":"boom"}`))
	}))
	defer srv.Close()

	lc := NewLangChainHTTP(LangChainHTTPConfig{BaseURL: srv.URL, Path: "/x"})
	_, err := lc.Answer(context.Background(), AnswerInput{Query: "q"})
	require.Error(t, err)
}

// SYS-ENG-01：Go→下游 HTTP 在 context 超时下须返回（不无限阻塞）。
func TestLangChainHTTP_Answer_ContextDeadline(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case <-r.Context().Done():
			return
		case <-time.After(2 * time.Second):
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(invokeResponse{Answer: "late"})
		}
	}))
	defer srv.Close()

	lc := NewLangChainHTTP(LangChainHTTPConfig{
		BaseURL:    srv.URL,
		Path:       "/v1/rag/invoke",
		HTTPClient: &http.Client{Timeout: 5 * time.Second},
	})
	ctx, cancel := context.WithTimeout(context.Background(), 40*time.Millisecond)
	defer cancel()
	_, err := lc.Answer(ctx, AnswerInput{Query: "q"})
	require.Error(t, err)
	assert.True(t, errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled),
		"期望 DeadlineExceeded 或 Canceled，实际: %v", err)
}

