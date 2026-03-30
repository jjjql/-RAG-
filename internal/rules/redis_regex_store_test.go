package rules

import (
	"context"
	"testing"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRedisRegexStore_CreateListInvalid(t *testing.T) {
	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer rdb.Close()

	st := NewRedisRegexStore(rdb)
	ctx := context.Background()

	_, err = st.Create(ctx, RegexRuleCreate{Pattern: "(unclosed", DAT: "x", Priority: 1})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrInvalidRegex)

	r1, err := st.Create(ctx, RegexRuleCreate{Pattern: `^hello$`, DAT: "world", Priority: 10})
	require.NoError(t, err)
	assert.NotEmpty(t, r1.ID)

	all, err := st.ListAll(ctx)
	require.NoError(t, err)
	require.Len(t, all, 1)
	assert.Equal(t, `^hello$`, all[0].Pattern)
}

func TestRedisRegexStore_PatchDelete(t *testing.T) {
	s, err := miniredis.Run()
	require.NoError(t, err)
	defer s.Close()

	rdb := redis.NewClient(&redis.Options{Addr: s.Addr()})
	defer rdb.Close()

	st := NewRedisRegexStore(rdb)
	ctx := context.Background()

	r1, err := st.Create(ctx, RegexRuleCreate{Pattern: `a`, DAT: "1", Priority: 1})
	require.NoError(t, err)

	p := `^b$`
	err = st.Update(ctx, RegexRule{ID: r1.ID, Pattern: p, DAT: "2", Priority: 2, CreatedAt: r1.CreatedAt, UpdatedAt: r1.UpdatedAt})
	require.NoError(t, err)

	got, err := st.GetByID(ctx, r1.ID)
	require.NoError(t, err)
	assert.Equal(t, p, got.Pattern)

	require.NoError(t, st.Delete(ctx, r1.ID))
	_, err = st.GetByID(ctx, r1.ID)
	assert.ErrorIs(t, err, ErrRegexNotFound)
}
