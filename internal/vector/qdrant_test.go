package vector

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQdrantStore_Search_Hit(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"result":[{"score":0.99,"payload":{"text":"cached-answer"}}],"status":"ok"}`))
	}))
	defer srv.Close()

	st := NewQdrant(QdrantConfig{
		BaseURL:        srv.URL,
		Collection:     "c1",
		ScoreThreshold: 0.9,
		HTTPClient:     srv.Client(),
	})
	sr, ok, err := st.Search(context.Background(), SearchInput{Vector: []float64{0.1, 0.2}})
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "cached-answer", sr.Text)
	assert.Equal(t, "cache", sr.HitKind)
}

func TestQdrantStore_Search_DedupKind(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte(`{"result":[{"score":0.99,"payload":{"text":"from-rag","source":"rag_writeback"}}],"status":"ok"}`))
	}))
	defer srv.Close()

	st := NewQdrant(QdrantConfig{
		BaseURL:        srv.URL,
		Collection:     "c1",
		ScoreThreshold: 0.9,
		HTTPClient:     srv.Client(),
	})
	sr, ok, err := st.Search(context.Background(), SearchInput{Vector: []float64{0.1, 0.2}})
	require.NoError(t, err)
	assert.True(t, ok)
	assert.Equal(t, "dedup", sr.HitKind)
}

func TestQdrantStore_UpsertWriteAnswer_OK(t *testing.T) {
	var method, path string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		method = r.Method
		path = r.URL.Path
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"status":"ok"}`))
	}))
	defer srv.Close()

	st := NewQdrant(QdrantConfig{
		BaseURL:        srv.URL,
		Collection:     "c1",
		ScoreThreshold: 0.9,
		HTTPClient:     srv.Client(),
	})
	err := st.UpsertWriteAnswer(context.Background(), WriteAnswerInput{
		Vector:  []float64{0.1, 0.2},
		Text:    "ans",
		Query:   "q",
		TraceID: "t1",
	})
	require.NoError(t, err)
	assert.Equal(t, http.MethodPut, method)
	assert.Contains(t, path, "/collections/c1/points")
}

func TestNoop_Search(t *testing.T) {
	var n Noop
	_, ok, err := n.Search(context.Background(), SearchInput{Vector: []float64{1}})
	require.NoError(t, err)
	assert.False(t, ok)
}
