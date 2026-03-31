package coalesce

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestSemanticLocal_similarVectors_singleFlight(t *testing.T) {
	var calls int32
	m := NewSemanticLocal(2*time.Second, 0.95)
	mk := Key(nil, "question one")
	embA := []float64{1, 0, 0}
	embB := []float64{0.99, 0.1, 0}

	const n = 16
	var wg sync.WaitGroup
	wg.Add(n)
	for i := 0; i < n; i++ {
		go func(useB bool) {
			defer wg.Done()
			emb := embA
			if useB {
				emb = embB
			}
			s, err := m.Merge(context.Background(), mk, emb, func(ctx context.Context) (string, error) {
				if atomic.AddInt32(&calls, 1) == 1 {
					time.Sleep(35 * time.Millisecond)
				}
				return "merged", nil
			})
			require.NoError(t, err)
			assert.Equal(t, "merged", s)
		}(i%2 == 1)
	}
	wg.Wait()
	assert.EqualValues(t, 1, calls)
}

func TestSemanticLocal_differentScope_noMerge(t *testing.T) {
	var calls int32
	m := NewSemanticLocal(2*time.Second, 0.95)
	emb := []float64{1, 0, 0}
	fn := func(ctx context.Context) (string, error) {
		atomic.AddInt32(&calls, 1)
		return "x", nil
	}
	s1 := "s1"
	s2 := "s2"
	_, err := m.Merge(context.Background(), Key(&s1, "q"), emb, fn)
	require.NoError(t, err)
	_, err = m.Merge(context.Background(), Key(&s2, "q"), emb, fn)
	require.NoError(t, err)
	assert.EqualValues(t, 2, calls)
}
