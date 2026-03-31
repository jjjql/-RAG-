package downstream

import "context"

// AnswerFunc 将函数适配为 Answerer（单测与替身）。
type AnswerFunc func(ctx context.Context, in AnswerInput) (string, error)

// Answer 实现 Answerer。
func (f AnswerFunc) Answer(ctx context.Context, in AnswerInput) (string, error) {
	return f(ctx, in)
}
