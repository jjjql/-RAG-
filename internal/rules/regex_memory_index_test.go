package rules

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRegexMemoryIndex_PriorityAndRecency(t *testing.T) {
	m := NewRegexMemoryIndex()
	scope := "s"
	older := time.Date(2020, 1, 1, 0, 0, 0, 0, time.UTC)
	newer := time.Date(2021, 1, 1, 0, 0, 0, 0, time.UTC)

	m.ReplaceAll([]RegexRule{
		{ID: "low", Scope: &scope, Pattern: `foo`, DAT: "L", Priority: 1, UpdatedAt: newer},
		{ID: "high-old", Scope: &scope, Pattern: `f.o`, DAT: "H1", Priority: 10, UpdatedAt: older},
		{ID: "high-new", Scope: &scope, Pattern: `f.o`, DAT: "H2", Priority: 10, UpdatedAt: newer},
	})

	id, dat, ok := m.Match("s", "foo")
	require.True(t, ok)
	assert.Equal(t, "high-new", id)
	assert.Equal(t, "H2", dat)
}

func TestRegexMemoryIndex_ScopeIsolation(t *testing.T) {
	m := NewRegexMemoryIndex()
	s1 := "a"
	m.ReplaceAll([]RegexRule{
		{ID: "1", Scope: &s1, Pattern: `^x$`, DAT: "A", Priority: 1, UpdatedAt: time.Now().UTC()},
	})
	_, _, ok := m.Match("b", "x")
	assert.False(t, ok)
	id, dat, ok := m.Match("a", "x")
	require.True(t, ok)
	assert.Equal(t, "1", id)
	assert.Equal(t, "A", dat)
}
