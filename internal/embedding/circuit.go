package embedding

import (
	"context"
	"fmt"

	"rag-gateway/internal/circuitbreaker"
)

// CircuitService 为 Service 增加熔断包装（SYS-ENG-01）。
type CircuitService struct {
	Inner  Service
	Breaker *circuitbreaker.Breaker
}

// Embed 在熔断允许时调用 Inner；开路时快速失败。
func (c *CircuitService) Embed(ctx context.Context, in EmbedInput) (*EmbedResult, error) {
	if c == nil || c.Inner == nil {
		return nil, fmt.Errorf("embedding: CircuitService 未初始化")
	}
	if c.Breaker != nil && !c.Breaker.Allow() {
		return nil, fmt.Errorf("embedding: %w", circuitbreaker.ErrOpen)
	}
	out, err := c.Inner.Embed(ctx, in)
	if c.Breaker != nil {
		if err != nil {
			c.Breaker.Fail()
		} else {
			c.Breaker.Success()
		}
	}
	return out, err
}
