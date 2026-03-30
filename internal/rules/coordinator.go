package rules

import (
	"context"
	"sort"
	"time"
)

// ExactCoordinator 组合 Redis 与内存索引；负责加载与热更新。
type ExactCoordinator struct {
	store *RedisExactStore
	mem   *ExactMemoryIndex
}

// NewExactCoordinator 构造协调器。
func NewExactCoordinator(store *RedisExactStore, mem *ExactMemoryIndex) *ExactCoordinator {
	return &ExactCoordinator{store: store, mem: mem}
}

// Reload 从 Redis 全量重建内存索引（Pub/Sub 或启动时调用）。
func (c *ExactCoordinator) Reload(ctx context.Context) error {
	rules, err := c.store.ListAll(ctx)
	if err != nil {
		return err
	}
	c.mem.ReplaceAll(rules)
	return nil
}

// Create 持久化并刷新内存（单实例即时生效；多实例依赖 Pub/Sub）。
func (c *ExactCoordinator) Create(ctx context.Context, in ExactRuleCreate) (ExactRule, error) {
	r, err := c.store.Create(ctx, in)
	if err != nil {
		return ExactRule{}, err
	}
	_ = c.Reload(ctx)
	return r, nil
}

// ExactRuleListResponse 与 OpenAPI 列表响应一致。
type ExactRuleListResponse struct {
	Items    []ExactRule `json:"items"`
	Page     int         `json:"page"`
	PageSize int         `json:"pageSize"`
	Total    int         `json:"total"`
}

// List 分页列表；scopeQuery 非空时过滤相同 scopeNorm。
func (c *ExactCoordinator) List(scopeQuery *string, page, pageSize int) ExactRuleListResponse {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	rules, err := c.snapshotRules()
	if err != nil {
		return ExactRuleListResponse{Items: []ExactRule{}, Page: page, PageSize: pageSize, Total: 0}
	}
	filtered := make([]ExactRule, 0, len(rules))
	want := ""
	if scopeQuery != nil {
		want = *scopeQuery
	}
	for _, r := range rules {
		if scopeQuery != nil && ScopeKey(r.Scope) != want {
			continue
		}
		filtered = append(filtered, r)
	}
	sort.Slice(filtered, func(i, j int) bool {
		ti, tj := filtered[i].UpdatedAt, filtered[j].UpdatedAt
		if ti.Equal(tj) {
			return filtered[i].ID < filtered[j].ID
		}
		return ti.After(tj)
	})

	total := len(filtered)
	start := (page - 1) * pageSize
	if start > total {
		start = total
	}
	end := start + pageSize
	if end > total {
		end = total
	}
	pageItems := filtered[start:end]
	if pageItems == nil {
		pageItems = []ExactRule{}
	}
	return ExactRuleListResponse{
		Items:    pageItems,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}
}

func (c *ExactCoordinator) snapshotRules() ([]ExactRule, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return c.store.ListAll(ctx)
}

// MatchExact 用户问答：精确命中（读内存）。
func (c *ExactCoordinator) MatchExact(scope *string, query string) (id, dat string, ok bool) {
	return c.mem.Lookup(ScopeKey(scope), query)
}

// GetExact 按 ID 返回规则（读 Redis）。
func (c *ExactCoordinator) GetExact(ctx context.Context, id string) (ExactRule, error) {
	return c.store.GetByID(ctx, id)
}

// Patch 部分更新并 Reload 内存索引。
func (c *ExactCoordinator) Patch(ctx context.Context, id string, p ExactRulePatch) (ExactRule, error) {
	r, err := c.store.Update(ctx, id, p)
	if err != nil {
		return ExactRule{}, err
	}
	_ = c.Reload(ctx)
	return r, nil
}

// Delete 删除规则并 Reload 内存索引。
func (c *ExactCoordinator) Delete(ctx context.Context, id string) error {
	err := c.store.Delete(ctx, id)
	if err != nil {
		return err
	}
	_ = c.Reload(ctx)
	return nil
}
