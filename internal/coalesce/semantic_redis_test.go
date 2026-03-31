package coalesce

import (
	"context"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSemanticRedis_similarVectors_singleDownstream(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()
	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})

	var calls int32
	s := NewSemanticRedis(rdb, RedisConfig{
		MergeTimeout: 2 * time.Second,
		LockTTL:      30 * time.Second,
		ResultTTL:    20 * time.Second,
	}, 0.95, 64)

	mk := Key(nil, "redis q1")
	embA := []float64{1, 0, 0}
	embB := []float64{0.99, 0.1, 0}

	fn := func(ctx context.Context) (string, error) {
		atomic.AddInt32(&calls, 1)
		time.Sleep(120 * time.Millisecond)
		return "one", nil
	}

	done := make(chan struct{})
	go func() {
		out, err := s.Merge(context.Background(), mk, embA, fn)
		require.NoError(t, err)
		assert.Equal(t, "one", out)
		close(done)
	}()
	time.Sleep(30 * time.Millisecond)
	out, err := s.Merge(context.Background(), mk, embB, fn)
	require.NoError(t, err)
	assert.Equal(t, "one", out)
	<-done
	assert.EqualValues(t, 1, calls)
}
