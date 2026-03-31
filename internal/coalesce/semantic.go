package coalesce

import "context"

// Semantic 基于 query 向量余弦相似度的 RAG 合并（须 coalesce.enabled + embedding）。
type Semantic interface {
	Merge(ctx context.Context, mergeKey string, embedding []float64, fn func(context.Context) (string, error)) (string, error)
}
