package httpnb

import (
	"encoding/json"
	"net/http"
)

// ErrorBody 与 interface/openapi.yaml 一致。
type ErrorBody struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	TraceID string `json:"traceId,omitempty"`
}

func writeJSONError(w http.ResponseWriter, status int, traceID string, code, msg string) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	if traceID != "" {
		w.Header().Set(headerTraceID, traceID)
	}
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(ErrorBody{
		Code:    code,
		Message: msg,
		TraceID: traceID,
	})
}
