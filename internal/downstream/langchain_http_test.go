package downstream

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
