package httpnb

import (
	"net/http"

	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// NewHandler 注册北向 HTTP 路由。
func NewHandler(d *Deps) http.Handler {
	mux := http.NewServeMux()
	mux.Handle("/metrics", promhttp.Handler())
	mux.HandleFunc("/v1/health", HealthHandler)
	if d != nil && d.Exact != nil {
		mux.HandleFunc("/v1/qa", d.handleQA)
		mux.HandleFunc("/v1/admin/rules/exact", d.handleExactCollection)
		mux.HandleFunc("/v1/admin/rules/exact/", d.handleExactItem)
	}
	if d != nil && d.Regex != nil {
		mux.HandleFunc("/v1/admin/rules/regex", d.handleRegexCollection)
		mux.HandleFunc("/v1/admin/rules/regex/", d.handleRegexItem)
	}
	return mux
}
