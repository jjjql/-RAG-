package coalesce

import (
	"math"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCosineSimilarity_identical(t *testing.T) {
	a := []float64{1, 2, 3}
	s, ok := CosineSimilarity(a, a)
	require.True(t, ok)
	assert.InDelta(t, 1.0, s, 1e-9)
}

func TestCosineSimilarity_orthogonal(t *testing.T) {
	a := []float64{1, 0, 0}
	b := []float64{0, 1, 0}
	s, ok := CosineSimilarity(a, b)
	require.True(t, ok)
	assert.InDelta(t, 0.0, s, 1e-9)
}

func TestCosineSimilarity_mismatchDim(t *testing.T) {
	_, ok := CosineSimilarity([]float64{1}, []float64{1, 2})
	assert.False(t, ok)
}

func TestCosineSimilarity_nearParallel(t *testing.T) {
	a := []float64{1, 0, 0}
	b := []float64{0.99, 0.1, 0}
	s, ok := CosineSimilarity(a, b)
	require.True(t, ok)
	want := 0.99 / (1.0 * math.Sqrt(0.99*0.99+0.1*0.1))
	assert.InDelta(t, want, s, 1e-6)
	assert.Greater(t, s, 0.95)
}
