package coalesce

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRAG_Do_Disabled(t *testing.T) {
	r := &RAG{Enabled: false}
	var n int32
	out, err := r.Do(context.Background(), "k", func(ctx context.Context) (string, error) {
		atomic.AddInt32(&n, 1)
		return "ok", nil
	})
	require.NoError(t, err)
	assert.Equal(t, "ok", out)
	assert.EqualValues(t, 1, n)
}

func TestRAG_Do_SingleFlight(t *testing.T) {
	r := &RAG{Enabled: true, MergeTimeout: 2 * time.Second}
	var n int32
	key := Key(nil, "Hello")
	const workers = 20
	ch := make(chan string, workers)
	for i := 0; i < workers; i++ {
		go func() {
			s, err := r.Do(context.Background(), key, func(ctx context.Context) (string, error) {
				atomic.AddInt32(&n, 1)
				time.Sleep(20 * time.Millisecond)
				return "once", nil
			})
			if err != nil {
				ch <- ""
				return
			}
			ch <- s
		}()
	}
	for i := 0; i < workers; i++ {
		s := <-ch
		assert.Equal(t, "once", s)
	}
	assert.EqualValues(t, 1, n)
}
