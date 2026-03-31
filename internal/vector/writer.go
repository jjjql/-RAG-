package vector

import "context"

// AnswerWriter 将 RAG 应答持久化到向量库（跨进程长期语义去重写回）。
type AnswerWriter interface {
	WriteAnswer(ctx context.Context, in WriteAnswerInput) error
}

type noopAnswerWriter struct{}

func (noopAnswerWriter) WriteAnswer(context.Context, WriteAnswerInput) error { return nil }

// NoopAnswerWriter 不写入。
var NoopAnswerWriter AnswerWriter = noopAnswerWriter{}
