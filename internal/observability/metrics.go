// Package observability 可观测性埋点（Prometheus 等）。
package observability

import (
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

var (
	qaCompleted   *prometheus.CounterVec
	qaPhaseSeconds *prometheus.HistogramVec
)

func init() {
	qaCompleted = prometheus.NewCounterVec(prometheus.CounterOpts{
		Namespace: "gateway",
		Name:      "qa_completed_total",
		Help:      "POST /v1/qa 业务路径完成次数（按 source 分组）",
	}, []string{"source"})
	buckets := []float64{.001, .002, .005, .01, .025, .05, .1, .25, .5, 1}
	qaPhaseSeconds = prometheus.NewHistogramVec(prometheus.HistogramOpts{
		Namespace: "gateway",
		Name:      "qa_phase_duration_seconds",
		Help:      "POST /v1/qa 分阶段耗时（秒）；embed/vector/rag_prep/coalesce 见 SYS_OBSERVABILITY_METRICS.md",
		Buckets:   buckets,
	}, []string{"phase"})
	prometheus.MustRegister(qaCompleted)
	prometheus.MustRegister(qaPhaseSeconds)
}

// RecordQA 记录一次问答路径收口（rule_exact / rule_regex / rag / error_*）。
func RecordQA(source string) {
	if source == "" {
		source = "unknown"
	}
	qaCompleted.WithLabelValues(source).Inc()
}

// ObserveQAPhase 记录分阶段耗时（phase：embed、vector、rag_prep、coalesce 等）。
func ObserveQAPhase(phase string, d time.Duration) {
	if phase == "" {
		phase = "unknown"
	}
	qaPhaseSeconds.WithLabelValues(phase).Observe(d.Seconds())
}
