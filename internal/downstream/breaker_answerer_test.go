package downstream

import (
	"context"
	"errors"
	"testing"
	"time"

	"rag-gateway/internal/circuitbreaker"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestWrapAnswerer_CircuitOpen(t *testing.T) {
	br := circuitbreaker.New(2, time.Hour)
	inner := AnswerFunc(func(ctx context.Context, in AnswerInput) (string, error) {
		return "", errors.New("fail")
	})
	a := WrapAnswerer(inner, br)
	ctx := context.Background()
	_, err1 := a.Answer(ctx, AnswerInput{Query: "q"})
	require.Error(t, err1)
	_, err2 := a.Answer(ctx, AnswerInput{Query: "q"})
	require.Error(t, err2)
	_, err3 := a.Answer(ctx, AnswerInput{Query: "q"})
	require.Error(t, err3)
	assert.ErrorIs(t, err3, circuitbreaker.ErrOpen)
}
