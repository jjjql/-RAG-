// Package vector L3 向量语义检索抽象（noop / Qdrant），见 specs/architecture/VECTOR_L3.md。
package vector

import "context"

// SearchInput 一次检索入参。
type SearchInput struct {
	Vector  []float64
	TraceID string
}

// SearchResult 命中时的缓存可见文本与相似度。
type SearchResult struct {
	Text  string
	Score float64
	// HitKind：cache=管理/预置 L3；dedup=持久化 RAG 写回（见 SEMANTIC_DEDUP_PERSISTENT.md）。
	HitKind string
}

// Store 向量库抽象；未命中时 ok=false。
type Store interface {
	Search(ctx context.Context, in SearchInput) (sr SearchResult, ok bool, err error)
}
