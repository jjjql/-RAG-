package httpnb

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"

	"github.com/google/uuid"

	"rag-gateway/internal/rules"
)

func (d *Deps) handleRegexCollection(w http.ResponseWriter, r *http.Request) {
	tid := writeTraceID(w, r)
	if d.Regex == nil {
		writeJSONError(w, http.StatusServiceUnavailable, tid, "服务不可用", "正则规则模块未初始化")
		return
	}
	switch r.Method {
	case http.MethodGet:
		d.listRegexRules(w, r, tid)
	case http.MethodPost:
		d.createRegexRule(w, r, tid)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, tid, "METHOD_NOT_ALLOWED", http.StatusText(http.StatusMethodNotAllowed))
	}
}

func (d *Deps) listRegexRules(w http.ResponseWriter, r *http.Request, tid string) {
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
	resp := d.Regex.List(scopePtr, page, ps)
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.Header().Set(headerTraceID, tid)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(resp)
}

func (d *Deps) createRegexRule(w http.ResponseWriter, r *http.Request, tid string) {
	var body rules.RegexRuleCreate
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		writeJSONError(w, http.StatusBadRequest, tid, "INVALID_JSON", "请求体不是合法 JSON")
		return
	}
	if strings.TrimSpace(body.Pattern) == "" {
		writeJSONError(w, http.StatusBadRequest, tid, "参数无效", "pattern 不能为空")
		return
	}
	rule, err := d.Regex.Create(r.Context(), body)
	if err != nil {
		if errors.Is(err, rules.ErrInvalidRegex) {
			writeJSONError(w, http.StatusBadRequest, tid, "参数无效", err.Error())
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

func (d *Deps) handleRegexItem(w http.ResponseWriter, r *http.Request) {
	tid := writeTraceID(w, r)
	if d.Regex == nil {
		writeJSONError(w, http.StatusServiceUnavailable, tid, "服务不可用", "正则规则模块未初始化")
		return
	}
	idStr := strings.TrimPrefix(r.URL.Path, "/v1/admin/rules/regex/")
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
		d.getRegexRule(w, r, tid, idStr)
	case http.MethodPatch:
		d.patchRegexRule(w, r, tid, idStr)
	case http.MethodDelete:
		d.deleteRegexRule(w, r, tid, idStr)
	default:
		writeJSONError(w, http.StatusMethodNotAllowed, tid, "METHOD_NOT_ALLOWED", http.StatusText(http.StatusMethodNotAllowed))
	}
}

func (d *Deps) getRegexRule(w http.ResponseWriter, r *http.Request, tid, id string) {
	rule, err := d.Regex.GetRegex(r.Context(), id)
	if errors.Is(err, rules.ErrRegexNotFound) {
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

func (d *Deps) patchRegexRule(w http.ResponseWriter, r *http.Request, tid, id string) {
	var patch rules.RegexRulePatch
	if err := json.NewDecoder(r.Body).Decode(&patch); err != nil {
		writeJSONError(w, http.StatusBadRequest, tid, "INVALID_JSON", "请求体不是合法 JSON")
		return
	}
	rule, err := d.Regex.PatchRegex(r.Context(), id, patch)
	if errors.Is(err, rules.ErrRegexNotFound) {
		writeJSONError(w, http.StatusNotFound, tid, "NOT_FOUND", "规则不存在")
		return
	}
	if err != nil {
		if errors.Is(err, rules.ErrInvalidRegex) {
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

func (d *Deps) deleteRegexRule(w http.ResponseWriter, r *http.Request, tid, id string) {
	err := d.Regex.DeleteRegex(r.Context(), id)
	if errors.Is(err, rules.ErrRegexNotFound) {
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
