package coalesce

import "context"

// 编译期断言 Merger 实现。
var (
	_ Merger = (*RAG)(nil)
	_ Merger = (*Redis)(nil)
	_ Merger = Passthrough{}
)

// Merger 对下游 RAG 调用的合并抽象：进程内 singleflight（RAG）或 Redis 跨实例合并（Redis）。
type Merger interface {
	Do(ctx context.Context, key string, fn func(context.Context) (string, error)) (string, error)
}

// Passthrough 不合并，直接执行 fn（coalesce.enabled=false 时使用）。
type Passthrough struct{}

func (Passthrough) Do(ctx context.Context, key string, fn func(context.Context) (string, error)) (string, error) {
	return fn(ctx)
}
