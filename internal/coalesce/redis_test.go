package coalesce

import (
	"context"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedis_Do_TwoInstancesOneDownstream(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = rdb.Close() }()

	cfg := RedisConfig{
		MergeTimeout: 2 * time.Second,
		LockTTL:      30 * time.Second,
		ResultTTL:    60 * time.Second,
	}
	a := NewRedis(rdb, cfg)
	b := NewRedis(rdb, cfg)

	var n int32
	key := Key(nil, "same question")

	var wg sync.WaitGroup
	wg.Add(2)
	go func() {
		defer wg.Done()
		s, err := a.Do(context.Background(), key, func(ctx context.Context) (string, error) {
			atomic.AddInt32(&n, 1)
			time.Sleep(40 * time.Millisecond)
			return "one", nil
		})
		require.NoError(t, err)
		assert.Equal(t, "one", s)
	}()
	go func() {
		defer wg.Done()
		// 确保实例 A 已抢到 SET NX，避免竞态下 B 误当 Leader（miniredis/调度敏感）。
		time.Sleep(30 * time.Millisecond)
		s, err := b.Do(context.Background(), key, func(ctx context.Context) (string, error) {
			atomic.AddInt32(&n, 1)
			return "should-not-run", nil
		})
		require.NoError(t, err)
		assert.Equal(t, "one", s)
	}()
	wg.Wait()
	assert.EqualValues(t, 1, n)
}

// TestRedis_Do_ErrorCached 失败结果写入 res 后，同键再次请求应走缓存且不再调用下游。
func TestRedis_Do_ErrorCached(t *testing.T) {
	mr, err := miniredis.Run()
	require.NoError(t, err)
	defer mr.Close()

	rdb := redis.NewClient(&redis.Options{Addr: mr.Addr()})
	defer func() { _ = rdb.Close() }()

	x := NewRedis(rdb, RedisConfig{
		MergeTimeout: time.Second,
		LockTTL:      30 * time.Second,
		ResultTTL:    60 * time.Second,
	})
	key := Key(nil, "err-cache")
	var calls int32
	_, err1 := x.Do(context.Background(), key, func(ctx context.Context) (string, error) {
		atomic.AddInt32(&calls, 1)
		return "", errors.New("boom")
	})
	require.Error(t, err1)
	_, err2 := x.Do(context.Background(), key, func(ctx context.Context) (string, error) {
		atomic.AddInt32(&calls, 1)
		return "no", nil
	})
	require.Error(t, err2)
	assert.EqualValues(t, 1, calls)
}
