package httpnb

import (
	"net/http"

	"github.com/google/uuid"
)

const headerTraceID = "X-Trace-Id"

// writeTraceID 将 trace 写入响应头：若请求已带合法 UUID 则回传，否则生成新的。
func writeTraceID(w http.ResponseWriter, r *http.Request) string {
	raw := r.Header.Get(headerTraceID)
	if id, err := uuid.Parse(raw); err == nil {
		w.Header().Set(headerTraceID, id.String())
		return id.String()
	}
	id := uuid.New()
	w.Header().Set(headerTraceID, id.String())
	return id.String()
}
