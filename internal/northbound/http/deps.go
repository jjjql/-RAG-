package httpnb

import (
	"rag-gateway/internal/coalesce"
	"rag-gateway/internal/downstream"
	"rag-gateway/internal/embedding"
	"rag-gateway/internal/rules"
	"rag-gateway/internal/vector"
)

// Deps 注入北向 HTTP 依赖。
type Deps struct {
	Exact      *rules.ExactCoordinator
	Regex      *rules.RegexCoordinator
	Embedder   embedding.Service   // 非 nil 且配置启用时，未命中规则后尝试侧车 embed
	Downstream *downstream.Client // 非 nil 且配置启用时，未命中规则（及可选 embed 成功后）走智能问答
	// Coalesce 同 scope+规范化 query 的下游合并策略（进程内或 Redis 跨机）；未启用时为 Passthrough。
	Coalesce coalesce.Merger
	// Semantic 非 nil 且 RAG 前已有 embedding 时，按余弦相似度合并（见 coalesce.Semantic）；否则仅用 Coalesce 文本键。
	Semantic coalesce.Semantic
	// VectorEnabled 为 true 且 Embed 成功时执行 L3 检索（SYS-ENG-01：超时/熔断见 vector 包装）。
	VectorEnabled bool
	Vector        vector.Store // 非 nil；未启用向量能力时为 Noop
	// DedupVector 非 nil 时：主 L3 未命中后再检索持久化去重集合（独立 collection 模式）。
	DedupVector vector.Store
	// DedupWriter 非 nil 时：RAG 成功后异步写回 Qdrant（见 SEMANTIC_DEDUP_PERSISTENT.md）。
	DedupWriter vector.AnswerWriter
}
