package downstream

import (
	"context"
	"fmt"

	"rag-gateway/internal/circuitbreaker"
)

type breakerAnswerer struct {
	inner Answerer
	br    *circuitbreaker.Breaker
}

// WrapAnswerer 为 Answerer 增加熔断；br 为 nil 时等价于 inner。
func WrapAnswerer(inner Answerer, br *circuitbreaker.Breaker) Answerer {
	if inner == nil {
		return nil
	}
	if br == nil {
		return inner
	}
	return &breakerAnswerer{inner: inner, br: br}
}

func (b *breakerAnswerer) Answer(ctx context.Context, in AnswerInput) (string, error) {
	if !b.br.Allow() {
		return "", fmt.Errorf("downstream: %w", circuitbreaker.ErrOpen)
	}
	text, err := b.inner.Answer(ctx, in)
	if err != nil {
		b.br.Fail()
	} else {
		b.br.Success()
	}
	return text, err
}
