package rules

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisExactStore_CreateAndList(t *testing.T) {
	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer rdb.Close()

	st := NewRedisExactStore(rdb)
	ctx := context.Background()

	scope := "t1"
	r1, err := st.Create(ctx, ExactRuleCreate{Scope: &scope, Key: "hello", DAT: "world"})
	require.NoError(t, err)
	assert.NotEmpty(t, r1.ID)

	_, err = st.Create(ctx, ExactRuleCreate{Scope: &scope, Key: "hello", DAT: "dup"})
	assert.ErrorIs(t, err, ErrConflict)

	all, err := st.ListAll(ctx)
	require.NoError(t, err)
	require.Len(t, all, 1)
	assert.Equal(t, "hello", all[0].Key)
	assert.Equal(t, "world", all[0].DAT)
}

func TestRedisExactStore_UpdateAndDelete(t *testing.T) {
	srv, err := miniredis.Run()
	require.NoError(t, err)
	defer srv.Close()

	rdb := redis.NewClient(&redis.Options{Addr: srv.Addr()})
	defer rdb.Close()

	st := NewRedisExactStore(rdb)
	ctx := context.Background()

	r1, err := st.Create(ctx, ExactRuleCreate{Key: "k1", DAT: "d1"})
	require.NoError(t, err)

	patchKey := "k1b"
	updated, err := st.Update(ctx, r1.ID, ExactRulePatch{Key: &patchKey, DAT: strPtr("d2")})
	require.NoError(t, err)
	assert.Equal(t, "k1b", updated.Key)
	assert.Equal(t, "d2", updated.DAT)

	all, err := st.ListAll(ctx)
	require.NoError(t, err)
	require.Len(t, all, 1)

	_, err = st.Create(ctx, ExactRuleCreate{Key: "dup", DAT: "x"})
	require.NoError(t, err)
	_, err = st.Update(ctx, r1.ID, ExactRulePatch{Key: strPtr("dup")})
	assert.ErrorIs(t, err, ErrConflict)

	require.NoError(t, st.Delete(ctx, r1.ID))
	_, err = st.GetByID(ctx, r1.ID)
	assert.ErrorIs(t, err, redis.Nil)

	all, err = st.ListAll(ctx)
	require.NoError(t, err)
	require.Len(t, all, 1)
}

func strPtr(s string) *string { return &s }

func TestMemoryIndex_ReplaceAndLookup(t *testing.T) {
	m := NewExactMemoryIndex()
	scope := "s"
	m.ReplaceAll([]ExactRule{
		{ID: "1", Scope: &scope, Key: "q", DAT: "a"},
	})
	id, dat, ok := m.Lookup("s", "q")
	assert.True(t, ok)
	assert.Equal(t, "1", id)
	assert.Equal(t, "a", dat)
	_, _, ok = m.Lookup("", "q")
	assert.False(t, ok)
}
