package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoad_HTTPAddrFromFile(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "cfg.yaml")
	require.NoError(t, os.WriteFile(p, []byte("server:\n  http_addr: :9999\nredis:\n  addr: localhost:6379\n"), 0o600))

	cfg, err := Load(p, ":8080")
	require.NoError(t, err)
	assert.Equal(t, ":9999", cfg.Server.HTTPAddr)
}

func TestLoad_DefaultHTTPAddr(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "cfg.yaml")
	require.NoError(t, os.WriteFile(p, []byte("server: {}\nredis:\n  addr: x\n"), 0o600))

	cfg, err := Load(p, ":7070")
	require.NoError(t, err)
	assert.Equal(t, ":7070", cfg.Server.HTTPAddr)
}

func TestLoad_DownstreamEnvOverridesFile(t *testing.T) {
	t.Setenv("GATEWAY_DOWNSTREAM_MODE", "langchain_http")
	t.Setenv("GATEWAY_DOWNSTREAM_HTTP_BASE_URL", "http://langchain-mock:1989")
	t.Setenv("GATEWAY_DOWNSTREAM_HTTP_PATH", "/v1/rag/invoke")
	t.Setenv("GATEWAY_DOWNSTREAM_TIMEOUT_MS", "9999")

	dir := t.TempDir()
	p := filepath.Join(dir, "cfg.yaml")
	yaml := `
server:
  http_addr: ":8080"
redis:
  addr: "127.0.0.1:6379"
downstream:
  enabled: true
  mode: mock
  mock_text: "file-mock"
  timeout_ms: 100
`
	require.NoError(t, os.WriteFile(p, []byte(yaml), 0o600))

	cfg, err := Load(p, ":8080")
	require.NoError(t, err)
	assert.Equal(t, "langchain_http", cfg.Downstream.Mode)
	assert.Equal(t, "http://langchain-mock:1989", cfg.Downstream.HTTPBaseURL)
	assert.Equal(t, "/v1/rag/invoke", cfg.Downstream.HTTPPath)
	assert.Equal(t, 9999, cfg.Downstream.TimeoutMS)
}
