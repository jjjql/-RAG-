package rules

import (
	"context"
	"sort"
	"time"
)

// RegexCoordinator 组合 Redis 与内存索引；负责加载与热更新。
type RegexCoordinator struct {
	store *RedisRegexStore
	mem   *RegexMemoryIndex
}

// NewRegexCoordinator 构造协调器。
func NewRegexCoordinator(store *RedisRegexStore, mem *RegexMemoryIndex) *RegexCoordinator {
	return &RegexCoordinator{store: store, mem: mem}
}

// Reload 从 Redis 全量重建内存索引。
func (c *RegexCoordinator) Reload(ctx context.Context) error {
	rules, err := c.store.ListAll(ctx)
	if err != nil {
		return err
	}
	c.mem.ReplaceAll(rules)
	return nil
}

// Create 持久化并刷新内存。
func (c *RegexCoordinator) Create(ctx context.Context, in RegexRuleCreate) (RegexRule, error) {
	r, err := c.store.Create(ctx, in)
	if err != nil {
		return RegexRule{}, err
	}
	_ = c.Reload(ctx)
	return r, nil
}

// List 分页列表（读 Redis，与精确规则列表策略一致）。
func (c *RegexCoordinator) List(scopeQuery *string, page, pageSize int) RegexRuleListResponse {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 20
	}
	if pageSize > 100 {
		pageSize = 100
	}

	rules, err := c.snapshotRegexRules()
	if err != nil {
		return RegexRuleListResponse{Items: []RegexRule{}, Page: page, PageSize: pageSize, Total: 0}
	}
	filtered := make([]RegexRule, 0, len(rules))
	for _, r := range rules {
		if scopeQuery != nil && ScopeKey(r.Scope) != *scopeQuery {
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
		pageItems = []RegexRule{}
	}
	return RegexRuleListResponse{
		Items:    pageItems,
		Page:     page,
		PageSize: pageSize,
		Total:    total,
	}
}

func (c *RegexCoordinator) snapshotRegexRules() ([]RegexRule, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return c.store.ListAll(ctx)
}

// GetRegex 按 ID 返回规则。
func (c *RegexCoordinator) GetRegex(ctx context.Context, id string) (RegexRule, error) {
	return c.store.GetByID(ctx, id)
}

// PatchRegex 部分更新。
func (c *RegexCoordinator) PatchRegex(ctx context.Context, id string, patch RegexRulePatch) (RegexRule, error) {
	cur, err := c.store.GetByID(ctx, id)
	if err != nil {
		return RegexRule{}, err
	}
	if patch.Scope != nil {
		cur.Scope = patch.Scope
	}
	if patch.Pattern != nil {
		cur.Pattern = *patch.Pattern
	}
	if patch.DAT != nil {
		cur.DAT = *patch.DAT
	}
	if patch.Priority != nil {
		cur.Priority = *patch.Priority
	}
	cur.UpdatedAt = time.Now().UTC()
	if err := c.store.Update(ctx, cur); err != nil {
		return RegexRule{}, err
	}
	_ = c.Reload(ctx)
	return cur, nil
}

// DeleteRegex 删除规则。
func (c *RegexCoordinator) DeleteRegex(ctx context.Context, id string) error {
	if _, err := c.store.GetByID(ctx, id); err != nil {
		return err
	}
	if err := c.store.Delete(ctx, id); err != nil {
		return err
	}
	_ = c.Reload(ctx)
	return nil
}

// MatchRegex 用户问答：正则命中（读内存）；userScope 与规则 scope 须一致（含全局空作用域）。
func (c *RegexCoordinator) MatchRegex(userScope *string, query string) (id, dat string, ok bool) {
	sk := ScopeKey(userScope)
	return c.mem.Match(sk, query)
}
