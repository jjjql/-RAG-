package httpnb

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"

	"rag-gateway/internal/rules"
)

func (d *Deps) handleExactCollection(w http.ResponseWriter, r *http.Request) {
	tid := writeTraceID(w, r)
	switch r.Method {
	case http.MethodGet:
		d.listExactRules(w, r, tid)
	case http.MethodPost:
		d.createExactRule(w, r, tid)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, tid, "METHOD_NOT_ALLOWED", http.StatusText(http.StatusMethodNotAllowed))
	}
}

func (d *Deps) listExactRules(w http.ResponseWriter, r *http.Request, tid string) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	if page == 0 {
		page = 1
	}
	ps, _ := strconv.Atoi(q.Get("pageSize"))
	if ps == 0 {
		ps = 20
	}
	var scopePtr *string
	if _, ok := q["scope"]; ok {
		s := q.Get("scope")
		scopePtr = &s
	}
	resp := d.Exact.List(scopePtr, page, ps)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set(headerTraceID, tid)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (d *Deps) createExactRule(w http.ResponseWriter, r *http.Request, tid string) {
	var body rules.ExactRuleCreate
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, tid, "INVALID_JSON", "请求体不是合法 JSON")
		return
	}
	if body.Key == "" {
		writeJSONError(w, http.StatusBadRequest, tid, "INVALID_KEY", "key 不能为空")
		return
	}
	rule, err := d.Exact.Create(r.Context(), body)
	if err != nil {
		if err == rules.ErrConflict {
			writeJSONError(w, http.StatusConflict, tid, "EXACT_KEY_CONFLICT", "同作用域下 KEY 已存在")
			return
		}
		writeJSONError(w, http.StatusInternalServerError, tid, "INTERNAL", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set(headerTraceID, tid)
	w.WriteHeader(http.StatusCreated)
	_ = json.NewEncoder(w).Encode(rule)
}

func (d *Deps) handleExactItem(w http.ResponseWriter, r *http.Request) {
	tid := writeTraceID(w, r)
	idStr := strings.TrimPrefix(r.URL.Path, "/v1/admin/rules/exact/")
	idStr = strings.TrimSpace(idStr)
	if idStr == "" || strings.Contains(idStr, "/") {
		writeJSONError(w, http.StatusNotFound, tid, "NOT_FOUND", "规则不存在")
		return
	}
	if _, err := uuid.Parse(idStr); err != nil {
		writeJSONError(w, http.StatusNotFound, tid, "NOT_FOUND", "规则不存在")
		return
	}
	switch r.Method {
	case http.MethodGet:
		d.getExactRule(w, r, tid, idStr)
	case http.MethodPatch:
		d.patchExactRule(w, r, tid, idStr)
	case http.MethodDelete:
		d.deleteExactRule(w, r, tid, idStr)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, tid, "METHOD_NOT_ALLOWED", http.StatusText(http.StatusMethodNotAllowed))
	}
}

func (d *Deps) getExactRule(w http.ResponseWriter, r *http.Request, tid, id string) {
	rule, err := d.Exact.GetExact(r.Context(), id)
	if errors.Is(err, redis.Nil) {
		writeJSONError(w, http.StatusNotFound, tid, "NOT_FOUND", "规则不存在")
		return
	}
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, tid, "INTERNAL", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set(headerTraceID, tid)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(rule)
}

func (d *Deps) patchExactRule(w http.ResponseWriter, r *http.Request, tid, id string) {
	var body rules.ExactRulePatch
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, tid, "INVALID_JSON", "请求体不是合法 JSON")
		return
	}
	if !body.HasAny() {
		writeJSONError(w, http.StatusBadRequest, tid, "参数无效", "至少提供一个可更新字段（scope/key/dat）")
		return
	}
	rule, err := d.Exact.Patch(r.Context(), id, body)
	if errors.Is(err, redis.Nil) {
		writeJSONError(w, http.StatusNotFound, tid, "NOT_FOUND", "规则不存在")
		return
	}
	if err != nil {
		if errors.Is(err, rules.ErrConflict) {
			writeJSONError(w, http.StatusConflict, tid, "EXACT_KEY_CONFLICT", "同作用域下 KEY 已存在")
			return
		}
		if strings.Contains(err.Error(), "patch 为空") || strings.Contains(err.Error(), "key 不能为空") {
			writeJSONError(w, http.StatusBadRequest, tid, "参数无效", err.Error())
			return
		}
		writeJSONError(w, http.StatusInternalServerError, tid, "INTERNAL", err.Error())
		return
	}
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set(headerTraceID, tid)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(rule)
}

func (d *Deps) deleteExactRule(w http.ResponseWriter, r *http.Request, tid, id string) {
	err := d.Exact.Delete(r.Context(), id)
	if errors.Is(err, redis.Nil) {
		writeJSONError(w, http.StatusNotFound, tid, "NOT_FOUND", "规则不存在")
		return
	}
	if err != nil {
		writeJSONError(w, http.StatusInternalServerError, tid, "INTERNAL", err.Error())
		return
	}
	w.Header().Set(headerTraceID, tid)
	w.WriteHeader(http.StatusNoContent)
}
