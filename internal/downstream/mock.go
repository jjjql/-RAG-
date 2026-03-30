package downstream

import (
	"context"
	"time"
)

// Mock 固定文本应答，用于 SYS-FUNC-03 / 集成测试。
type Mock struct {
	Text  string
	Delay time.Duration // 可选，用于超时单测
}

// NewMock 构造 Mock；text 为空时使用默认占位串。
func NewMock(text string) *Mock {
	if text == "" {
		text = "mock-rag"
	}
	return &Mock{Text: text}
}

// Answer 实现 Answerer。
func (m *Mock) Answer(ctx context.Context, in AnswerInput) (string, error) {
	if m.Delay > 0 {
		t := time.NewTimer(m.Delay)
		defer t.Stop()
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-t.C:
		}
	}
	return m.Text, nil
}
