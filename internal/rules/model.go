package rules

import "time"

// ExactRule 与 interface/openapi.yaml ExactRule 对齐（JSON 驼峰）。
type ExactRule struct {
	ID        string     `json:"id"`
	Scope     *string    `json:"scope,omitempty"`
	Key       string     `json:"key"`
	DAT       string     `json:"dat"`
	CreatedAt time.Time  `json:"createdAt"`
	UpdatedAt time.Time  `json:"updatedAt"`
}

// ExactRuleCreate 管理端创建体。
type ExactRuleCreate struct {
	Scope *string `json:"scope,omitempty"`
	Key   string  `json:"key"`
	DAT   string  `json:"dat"`
}

// ExactRulePatch 管理端部分更新（与 interface/openapi.yaml ExactRulePatch 对齐）。
type ExactRulePatch struct {
	Scope *string `json:"scope,omitempty"`
	Key   *string `json:"key,omitempty"`
	DAT   *string `json:"dat,omitempty"`
}

// HasAny 是否包含至少一个可更新字段。
func (p ExactRulePatch) HasAny() bool {
	return p.Scope != nil || p.Key != nil || p.DAT != nil
}

// ScopeKey 返回用于索引与匹配的作用域规范化字符串（空串表示全局）。
func ScopeKey(scope *string) string {
	if scope == nil || *scope == "" {
		return ""
	}
	return *scope
}
