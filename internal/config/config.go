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
	Downstream DownstreamConfig `mapstructure:"downstream"`
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
	_ = v.BindEnv("downstream.http_api_key_header", "GATEWAY_DOWNSTREAM_HTTP_API_KEY_HEADER")

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
