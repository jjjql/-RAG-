package config

import (
	"errors"
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config 网关根配置（随功能扩展字段）。
type Config struct {
	Server     ServerConfig     `mapstructure:"server"`
	Redis      RedisConfig      `mapstructure:"redis"`
	Embedding  EmbeddingConfig  `mapstructure:"embedding"`
	Vector     VectorConfig     `mapstructure:"vector"`
	Coalesce   CoalesceConfig   `mapstructure:"coalesce"`
	Downstream DownstreamConfig `mapstructure:"downstream"`
}

// VectorConfig L3 向量语义缓存（见 specs/architecture/VECTOR_L3.md）。
type VectorConfig struct {
	Enabled        bool    `mapstructure:"enabled"`
	Mode           string  `mapstructure:"mode"` // noop | qdrant
	ScoreThreshold float64 `mapstructure:"score_threshold"`
	// TimeoutMS 单次向量检索超时（SYS-ENG-01：Go→Qdrant），默认 100。
	TimeoutMS int `mapstructure:"timeout_ms"`
	Qdrant     QdrantConfigYAML `mapstructure:"qdrant"`
	// SemanticDedup 跨进程持久化语义去重（见 SEMANTIC_DEDUP_PERSISTENT.md）。
	SemanticDedup SemanticDedupConfig `mapstructure:"semantic_dedup"`
	// CircuitBreaker 向量检索熔断；见 SYS_ENG_01_BREAKER.md。
	CircuitBreaker CircuitBreakerConfig `mapstructure:"circuit_breaker"`
}

// SemanticDedupConfig RAG 成功写回 Qdrant；读可走主集合（payload.source）或独立集合二次检索。
type SemanticDedupConfig struct {
	Enabled bool `mapstructure:"enabled"`
	// Collection 为空：与 qdrant.collection 共用，依赖单次 Search + payload.source=rag_writeback 区分命中来源。
	// 非空且与主集合不同：L3 miss 后再检索本集合；写回仅写入本集合。
	Collection string `mapstructure:"collection"`
	// ScoreThreshold 检索阈值；≤0 时沿用 vector.score_threshold。
	ScoreThreshold float64 `mapstructure:"score_threshold"`
}

// QdrantConfigYAML 映射至 vector.QdrantConfig。
type QdrantConfigYAML struct {
	URL        string `mapstructure:"url"`
	Collection string `mapstructure:"collection"`
	APIKey     string `mapstructure:"api_key"`
}

// CoalesceConfig 相似请求合并（见 specs/architecture/COALESCE_DESIGN.md）。
type CoalesceConfig struct {
	Enabled bool `mapstructure:"enabled"`
	// Mode：local（进程内 singleflight）| redis（跨实例，依赖 redis.addr）。
	Mode string `mapstructure:"mode"`
	// Semantic 为 true 时，RAG 合并键由「同 scope 下 embedding 余弦相似度 ≥ similarity_threshold」决定（须 embedding.enabled）。
	Semantic bool `mapstructure:"semantic"`
	// SimilarityThreshold 余弦相似度下限，默认 0.95；仅在 semantic=true 时生效。
	SimilarityThreshold float64 `mapstructure:"similarity_threshold"`
	// SemanticMaxActive 单 scope 下语义合并活跃组上限（redis 模式）；超出则当次请求不再合并。
	SemanticMaxActive int `mapstructure:"semantic_max_active"`
	// LockTTLSeconds Redis 占锁秒数，应大于 downstream.timeout_ms。
	LockTTLSeconds int `mapstructure:"lock_ttl_seconds"`
	// ResultTTLSeconds 结果键保留秒数，其它网关实例在此期间可读同一次应答。
	ResultTTLSeconds int `mapstructure:"result_ttl_seconds"`
}

// CircuitBreakerConfig 熔断（SYS-ENG-01）；Enabled=false 时不包装下游调用。
type CircuitBreakerConfig struct {
	Enabled           bool `mapstructure:"enabled"`
	FailureThreshold  int  `mapstructure:"failure_threshold"`
	OpenSeconds       int  `mapstructure:"open_seconds"`
}

// EmbeddingConfig 侧车 UDS/TCP（见 specs/architecture/contracts/UDS_EMBEDDING.md）。
type EmbeddingConfig struct {
	Enabled bool `mapstructure:"enabled"`
	// Transport：unix（默认）或 tcp（Windows 等环境可显式选 tcp）。
	Transport string `mapstructure:"transport"`
	// SocketPath UDS 路径，默认与 .cursorrules 一致 /tmp/rag_gateway.sock。
	SocketPath string `mapstructure:"socket_path"`
	// TCPAddr 如 127.0.0.1:18080；仅 transport=tcp 时必填。
	TCPAddr string `mapstructure:"tcp_addr"`
	// TimeoutMS 单次 embed 超时，默认 100（毫秒）。
	TimeoutMS int `mapstructure:"timeout_ms"`
	// PingTimeoutMS ping/pong 超时，默认 5000。
	PingTimeoutMS int `mapstructure:"ping_timeout_ms"`
	// CircuitBreaker 侧车调用熔断；见 specs/architecture/SYS_ENG_01_BREAKER.md。
	CircuitBreaker CircuitBreakerConfig `mapstructure:"circuit_breaker"`
}

// DownstreamConfig 智能问答下游（FR-U01/FR-U03）；mode=mock | langchain_http。
type DownstreamConfig struct {
	Enabled   bool   `mapstructure:"enabled"`
	Mode      string `mapstructure:"mode"` // mock | langchain_http
	MockText  string `mapstructure:"mock_text"`
	// MockDelayMS 仅 mode=mock：Answer 前睡眠毫秒数，用于 SYS-ENG-01 超时/故障注入（生产勿用）。
	MockDelayMS int `mapstructure:"mock_delay_ms"`
	TimeoutMS int    `mapstructure:"timeout_ms"`
	// LangChain 南向 HTTP（见 specs/architecture/contracts/HTTP_LANGCHAIN_DOWNSTREAM.md）
	HTTPBaseURL      string `mapstructure:"http_base_url"`
	HTTPPath         string `mapstructure:"http_path"`
	HTTPAPIKeyHeader string `mapstructure:"http_api_key_header"`
	HTTPAPIKey       string `mapstructure:"http_api_key"` // 建议用环境变量 GATEWAY_DOWNSTREAM_HTTP_API_KEY
	// CircuitBreaker 下游调用熔断；见 specs/architecture/SYS_ENG_01_BREAKER.md。
	CircuitBreaker CircuitBreakerConfig `mapstructure:"circuit_breaker"`
}

// ServerConfig HTTP 等服务监听配置。
type ServerConfig struct {
	HTTPAddr string `mapstructure:"http_addr"`
}

// RedisConfig Redis 连接（精确规则持久化 + Pub/Sub）。
type RedisConfig struct {
	Addr     string `mapstructure:"addr"`
	Password string `mapstructure:"password"`
	DB       int    `mapstructure:"db"`
}

// Load 从 path 指向的 YAML 加载配置；环境变量前缀 GATEWAY_，嵌套键用 _（如 GATEWAY_REDIS_ADDR）。
func Load(path string, defaultHTTPAddr string) (*Config, error) {
	if path == "" {
		return nil, errors.New("config: path 为空")
	}
	if defaultHTTPAddr == "" {
		defaultHTTPAddr = ":8080"
	}

	v := viper.New()
	v.SetEnvPrefix("GATEWAY")
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))
	v.AutomaticEnv()
	// 嵌套键在部分环境下需显式绑定（Docker Compose 常用）
	_ = v.BindEnv("redis.addr", "GATEWAY_REDIS_ADDR")
	_ = v.BindEnv("downstream.http_api_key", "GATEWAY_DOWNSTREAM_HTTP_API_KEY")
	_ = v.BindEnv("downstream.mode", "GATEWAY_DOWNSTREAM_MODE")
	_ = v.BindEnv("downstream.http_base_url", "GATEWAY_DOWNSTREAM_HTTP_BASE_URL")
	_ = v.BindEnv("downstream.http_path", "GATEWAY_DOWNSTREAM_HTTP_PATH")
	_ = v.BindEnv("downstream.timeout_ms", "GATEWAY_DOWNSTREAM_TIMEOUT_MS")
	_ = v.BindEnv("downstream.mock_delay_ms", "GATEWAY_DOWNSTREAM_MOCK_DELAY_MS")
	_ = v.BindEnv("downstream.http_api_key_header", "GATEWAY_DOWNSTREAM_HTTP_API_KEY_HEADER")
	_ = v.BindEnv("coalesce.enabled", "GATEWAY_COALESCE_ENABLED")
	_ = v.BindEnv("coalesce.mode", "GATEWAY_COALESCE_MODE")
	_ = v.BindEnv("coalesce.semantic", "GATEWAY_COALESCE_SEMANTIC")
	_ = v.BindEnv("coalesce.similarity_threshold", "GATEWAY_COALESCE_SIMILARITY_THRESHOLD")
	_ = v.BindEnv("coalesce.semantic_max_active", "GATEWAY_COALESCE_SEMANTIC_MAX_ACTIVE")
	_ = v.BindEnv("vector.enabled", "GATEWAY_VECTOR_ENABLED")
	_ = v.BindEnv("vector.mode", "GATEWAY_VECTOR_MODE")
	_ = v.BindEnv("vector.timeout_ms", "GATEWAY_VECTOR_TIMEOUT_MS")
	_ = v.BindEnv("vector.score_threshold", "GATEWAY_VECTOR_SCORE_THRESHOLD")
	_ = v.BindEnv("vector.qdrant.url", "GATEWAY_VECTOR_QDRANT_URL")
	_ = v.BindEnv("vector.qdrant.collection", "GATEWAY_VECTOR_QDRANT_COLLECTION")
	_ = v.BindEnv("vector.qdrant.api_key", "GATEWAY_VECTOR_QDRANT_API_KEY")
	_ = v.BindEnv("vector.semantic_dedup.enabled", "GATEWAY_VECTOR_SEMANTIC_DEDUP_ENABLED")
	_ = v.BindEnv("vector.semantic_dedup.collection", "GATEWAY_VECTOR_SEMANTIC_DEDUP_COLLECTION")
	_ = v.BindEnv("vector.semantic_dedup.score_threshold", "GATEWAY_VECTOR_SEMANTIC_DEDUP_SCORE_THRESHOLD")

	v.SetConfigFile(path)
	v.SetConfigType("yaml")
	if err := v.ReadInConfig(); err != nil {
		return nil, fmt.Errorf("读取配置: %w", err)
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("解析配置: %w", err)
	}
	if cfg.Server.HTTPAddr == "" {
		cfg.Server.HTTPAddr = defaultHTTPAddr
	}
	return &cfg, nil
}
