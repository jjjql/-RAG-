package coalesce

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"rag-gateway/internal/downstream"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// 与 qa.go 一致：RAG.Do 内调用 downstream.Client.Complete，并发同键只应触发一次 Answer。
func TestRAG_Do_WithDownstreamClient_SingleFlight(t *testing.T) {
	var calls int32
	ans := downstream.AnswerFunc(func(ctx context.Context, in downstream.AnswerInput) (string, error) {
		atomic.AddInt32(&calls, 1)
		time.Sleep(40 * time.Millisecond)
		return "once", nil
	})
	c := &downstream.Client{A: ans, Timeout: 2 * time.Second}
	r := &RAG{Enabled: true, MergeTimeout: 2 * time.Second}
	key := Key(nil, "Hello")
	const n = 12
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func() {
			defer wg.Done()
			s, err := r.Do(context.Background(), key, func(ctx context.Context) (string, error) {
				return c.Complete(ctx, downstream.AnswerInput{Query: "Hello"})
			})
			require.NoError(t, err)
			assert.Equal(t, "once", s)
		}()
	}
	wg.Wait()
	assert.EqualValues(t, 1, calls)
}
