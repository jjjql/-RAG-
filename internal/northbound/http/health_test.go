package httpnb

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHealthHandler_OK(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	rr := httptest.NewRecorder()
	HealthHandler(rr, req)

	assert.Equal(t, http.StatusOK, rr.Code)
	assert.Contains(t, rr.Header().Get("Content-Type"), "application/json")
	tid := rr.Header().Get(headerTraceID)
	_, err := uuid.Parse(tid)
	require.NoError(t, err)
	assert.JSONEq(t, `{"status":"ok"}`, rr.Body.String())
}

func TestHealthHandler_EchoTraceID(t *testing.T) {
	fixed := uuid.MustParse("550e8400-e29b-41d4-a716-446655440000")
	req := httptest.NewRequest(http.MethodGet, "/v1/health", nil)
	req.Header.Set(headerTraceID, fixed.String())
	rr := httptest.NewRecorder()
	HealthHandler(rr, req)

	assert.Equal(t, fixed.String(), rr.Header().Get(headerTraceID))
}
