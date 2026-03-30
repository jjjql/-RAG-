package httpnb

import (
	"encoding/json"
	"net/http"
)

// HealthResponse 与 interface/openapi.yaml HealthResponse 一致。
type HealthResponse struct {
	Status string `json:"status"`
}

// HealthHandler 实现 GET /v1/health。
func HealthHandler(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}
	writeTraceID(w, r)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(HealthResponse{Status: "ok"})
}
