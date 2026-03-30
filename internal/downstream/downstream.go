// Package downstream 智能问答下游适配（可 Mock，单测不连外网）。
package downstream

import (
	"context"
	"errors"
	"time"
)

// ErrDisabled 未启用下游或未配置 Answerer。
var ErrDisabled = errors.New("downstream: disabled")

// AnswerInput 一次问答下游调用入参。
type AnswerInput struct {
	Query     string
	TraceID   string
	SessionID string // 北向 sessionId 透传（FR-U01 可选上下文）
}

// Answerer 下游抽象（RAG / Mock）。
type Answerer interface {
	Answer(ctx context.Context, in AnswerInput) (text string, err error)
}

// Client 带超时的编排封装（SYS-ENG-01：调用下游须设上限）。
type Client struct {
	A       Answerer
	Timeout time.Duration
}

// Complete 在 Timeout 内调用 Answerer；c 或 A 为 nil 返回 ErrDisabled。
func (c *Client) Complete(ctx context.Context, in AnswerInput) (string, error) {
	if c == nil || c.A == nil {
		return "", ErrDisabled
	}
	to := c.Timeout
	if to <= 0 {
		to = 100 * time.Millisecond
	}
	ctx2, cancel := context.WithTimeout(ctx, to)
	defer cancel()
	return c.A.Answer(ctx2, in)
}
