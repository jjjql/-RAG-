package vector

import (
	"context"
	"time"
)

// TimeoutUpserter 为 Upsert 增加独立超时（失败不影响用户 SSE；由调用方记录日志）。
type TimeoutUpserter struct {
	Inner   *QdrantStore
	Timeout time.Duration
}

// WriteAnswer 实现 AnswerWriter。
func (t *TimeoutUpserter) WriteAnswer(ctx context.Context, in WriteAnswerInput) error {
	if t == nil || t.Inner == nil {
		return nil
	}
	to := t.Timeout
	if to <= 0 {
		to = 3 * time.Second
	}
	c2, cancel := context.WithTimeout(ctx, to)
	defer cancel()
	return t.Inner.UpsertWriteAnswer(c2, in)
}
