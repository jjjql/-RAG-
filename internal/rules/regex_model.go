package rules

import "time"

// RegexRule 与 interface/openapi.yaml RegexRule 对齐。
type RegexRule struct {
	ID        string    `json:"id"`
	Scope     *string   `json:"scope,omitempty"`
	Pattern   string    `json:"pattern"`
	DAT       string    `json:"dat"`
	Priority  int       `json:"priority"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

// RegexRuleCreate 管理端创建体。
type RegexRuleCreate struct {
	Scope    *string `json:"scope,omitempty"`
	Pattern  string  `json:"pattern"`
	DAT      string  `json:"dat"`
	Priority int     `json:"priority"`
}

// RegexRulePatch 管理端部分更新。
type RegexRulePatch struct {
	Scope    *string `json:"scope,omitempty"`
	Pattern  *string `json:"pattern,omitempty"`
	DAT      *string `json:"dat,omitempty"`
	Priority *int    `json:"priority,omitempty"`
}

// RegexRuleListResponse 与 OpenAPI 列表响应一致。
type RegexRuleListResponse struct {
	Items    []RegexRule `json:"items"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
	Total    int         `json:"total"`
}
