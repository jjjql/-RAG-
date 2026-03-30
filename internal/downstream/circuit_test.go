package downstream

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"rag-gateway/internal/circuitbreaker"
)

type boomAnswerer struct{}

func (boomAnswerer) Answer(ctx context.Context, in AnswerInput) (string, error) {
	return "", errors.New("boom")
}

func TestWrapAnswerer_Open(t *testing.T) {
	br := circuitbreaker.New(2, 10*time.Millisecond)
	a := WrapAnswerer(boomAnswerer{}, br)
	ctx := context.Background()
	in := AnswerInput{Query: "q"}
	_, _ = a.Answer(ctx, in)
	_, _ = a.Answer(ctx, in)
	_, err := a.Answer(ctx, in)
	assert.ErrorIs(t, err, circuitbreaker.ErrOpen)
}
