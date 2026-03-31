package vector

import (
	"context"
	"time"

	"rag-gateway/internal/circuitbreaker"
)

// CircuitStore 对 Store 包装：单次 Search 使用 OpTimeout（默认 100ms）；可选熔断（SYS-ENG-01：Go→Qdrant）。
type CircuitStore struct {
	Inner     Store
	Breaker   *circuitbreaker.Breaker
	OpTimeout time.Duration
}

// NewCircuitStore 构造；inner 非空。breaker 可为 nil（仅超时）。opTimeout<=0 时默认 100ms。
func NewCircuitStore(inner Store, breaker *circuitbreaker.Breaker, opTimeout time.Duration) *CircuitStore {
	if opTimeout <= 0 {
		opTimeout = 100 * time.Millisecond
	}
	return &CircuitStore{Inner: inner, Breaker: breaker, OpTimeout: opTimeout}
}

// Search 在 Allow 与超时内调用 Inner；无命中（ok=false, err=nil）计为成功。
func (c *CircuitStore) Search(ctx context.Context, in SearchInput) (SearchResult, bool, error) {
	if c == nil || c.Inner == nil {
		return SearchResult{}, false, nil
	}
	if c.Breaker != nil && !c.Breaker.Allow() {
		return SearchResult{}, false, circuitbreaker.ErrOpen
	}
	to := c.OpTimeout
	if to <= 0 {
		to = 100 * time.Millisecond
	}
	c2, cancel := context.WithTimeout(ctx, to)
	defer cancel()
	sr, ok, err := c.Inner.Search(c2, in)
	if c.Breaker != nil {
		if err != nil {
			c.Breaker.Fail()
		} else {
			c.Breaker.Success()
		}
	}
	return sr, ok, err
}
