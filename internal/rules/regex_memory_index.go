package rules

import (
	"regexp"
	"sort"
	"sync"
)

// regexCompiled 预编译条目（用户路径只读，不访问 Redis）。
type regexCompiled struct {
	ID       string
	DAT      string
	Priority int
	Updated  int64 // UnixNano 用于稳定排序
	re       *regexp.Regexp
}

// RegexMemoryIndex 正则规则内存索引：按作用域分组，priority 降序，同 priority 按更新时间降序。
type RegexMemoryIndex struct {
	mu sync.RWMutex
	// key: scopeNorm；每条切片已排序，遍历即匹配顺序
	byScope map[string][]regexCompiled
}

// NewRegexMemoryIndex 构造空索引。
func NewRegexMemoryIndex() *RegexMemoryIndex {
	return &RegexMemoryIndex{byScope: make(map[string][]regexCompiled)}
}

// ReplaceAll 用全量规则替换索引；跳过编译失败的规则（数据面应已在写入时校验）。
func (x *RegexMemoryIndex) ReplaceAll(rules []RegexRule) {
	grouped := make(map[string][]RegexRule)
	for _, r := range rules {
		sk := ScopeKey(r.Scope)
		grouped[sk] = append(grouped[sk], r)
	}
	next := make(map[string][]regexCompiled, len(grouped))
	for sk, list := range grouped {
		compiled := make([]regexCompiled, 0, len(list))
		for _, r := range list {
			re, err := regexp.Compile(r.Pattern)
			if err != nil {
				continue
			}
			compiled = append(compiled, regexCompiled{
				ID:       r.ID,
				DAT:      r.DAT,
				Priority: r.Priority,
				Updated:  r.UpdatedAt.UnixNano(),
				re:       re,
			})
		}
		sort.Slice(compiled, func(i, j int) bool {
			if compiled[i].Priority != compiled[j].Priority {
				return compiled[i].Priority > compiled[j].Priority
			}
			if compiled[i].Updated != compiled[j].Updated {
				return compiled[i].Updated > compiled[j].Updated
			}
			return compiled[i].ID < compiled[j].ID
		})
		next[sk] = compiled
	}
	x.mu.Lock()
	x.byScope = next
	x.mu.Unlock()
}

// Match 在用户作用域下对 query 做首次命中；仅匹配 scopeNorm 相同的规则（与精确规则分区一致）。
func (x *RegexMemoryIndex) Match(scopeNorm, query string) (id, dat string, ok bool) {
	x.mu.RLock()
	list := x.byScope[scopeNorm]
	x.mu.RUnlock()
	for _, c := range list {
		if c.re.MatchString(query) {
			return c.ID, c.DAT, true
		}
	}
	return "", "", false
}
