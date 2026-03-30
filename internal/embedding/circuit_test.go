package embedding

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"

	"rag-gateway/internal/circuitbreaker"
)

type stubSvc struct {
	err error
}

func (s stubSvc) Embed(ctx context.Context, in EmbedInput) (*EmbedResult, error) {
	if s.err != nil {
		return nil, s.err
	}
	return &EmbedResult{Dimensions: 1, Embedding: []float64{1}}, nil
}

func TestCircuitService_Embed_Open(t *testing.T) {
	br := circuitbreaker.New(2, 0)
	svc := &CircuitService{
		Inner:   stubSvc{err: errors.New("boom")},
		Breaker: br,
	}
	_, _ = svc.Embed(context.Background(), EmbedInput{Text: "a"})
	_, _ = svc.Embed(context.Background(), EmbedInput{Text: "a"})
	_, err := svc.Embed(context.Background(), EmbedInput{Text: "a"})
	assert.Error(t, err)
	assert.ErrorIs(t, err, circuitbreaker.ErrOpen)
}
