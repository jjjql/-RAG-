package vector

import (
	"context"
	"errors"
	"testing"
	"time"

	"rag-gateway/internal/circuitbreaker"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type errStore struct{ err error }

func (e errStore) Search(ctx context.Context, in SearchInput) (SearchResult, bool, error) {
	return SearchResult{}, false, e.err
}

func TestCircuitStore_OpenFastFail(t *testing.T) {
	b := circuitbreaker.New(1, time.Hour)
	inner := errStore{err: errors.New("boom")}
	s := NewCircuitStore(inner, b, 50*time.Millisecond)
	_, _, err := s.Search(context.Background(), SearchInput{Vector: []float64{1}})
	require.Error(t, err)
	_, _, err = s.Search(context.Background(), SearchInput{Vector: []float64{1}})
	require.Error(t, err)
	assert.True(t, errors.Is(err, circuitbreaker.ErrOpen))
}

func TestCircuitStore_Timeout(t *testing.T) {
	slow := StoreFunc(func(ctx context.Context, in SearchInput) (SearchResult, bool, error) {
		select {
		case <-time.After(200 * time.Millisecond):
			return SearchResult{}, false, nil
		case <-ctx.Done():
			return SearchResult{}, false, ctx.Err()
		}
	})
	s := NewCircuitStore(slow, nil, 20*time.Millisecond)
	_, _, err := s.Search(context.Background(), SearchInput{Vector: []float64{1}})
	require.Error(t, err)
	assert.ErrorIs(t, err, context.DeadlineExceeded)
}

// StoreFunc 测试用 Store 实现。
type StoreFunc func(context.Context, SearchInput) (SearchResult, bool, error)

func (f StoreFunc) Search(ctx context.Context, in SearchInput) (SearchResult, bool, error) {
	return f(ctx, in)
}
