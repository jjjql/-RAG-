package httpnb

import (
	"rag-gateway/internal/downstream"
	"rag-gateway/internal/embedding"
	"rag-gateway/internal/rules"
)

// Deps 注入北向 HTTP 依赖。
type Deps struct {
	Exact      *rules.ExactCoordinator
	Regex      *rules.RegexCoordinator
	Embedder   embedding.Service   // 非 nil 且配置启用时，未命中规则后尝试侧车 embed
	Downstream *downstream.Client // 非 nil 且配置启用时，未命中规则（及可选 embed 成功后）走智能问答
}
