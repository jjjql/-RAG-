package rules

import (
	"strings"
	"sync"
)

// memoryEntry 内存命中项（用户路径只读此处，不访问 Redis）。
type memoryEntry struct {
	ID  string
	DAT string
}

// ExactMemoryIndex 精确规则内存索引（热路径）。
type ExactMemoryIndex struct {
	mu sync.RWMutex
	// key: scopeNorm + "\x00" + exactKey
	m map[string]memoryEntry
}

// NewExactMemoryIndex 构造空索引。
func NewExactMemoryIndex() *ExactMemoryIndex {
	return &ExactMemoryIndex{m: make(map[string]memoryEntry)}
}

func compositeKey(scopeNorm, exactKey string) string {
	var b strings.Builder
	b.WriteString(scopeNorm)
	b.WriteByte(0)
	b.WriteString(exactKey)
	return b.String()
}

// ReplaceAll 用全量规则替换索引。
func (x *ExactMemoryIndex) ReplaceAll(rules []ExactRule) {
	next := make(map[string]memoryEntry, len(rules))
	for _, r := range rules {
		sk := ScopeKey(r.Scope)
		next[compositeKey(sk, r.Key)] = memoryEntry{ID: r.ID, DAT: r.DAT}
	}
	x.mu.Lock()
	x.m = next
	x.mu.Unlock()
}

// Lookup 按用户问题文本与作用域做精确相等匹配（与配置 KEY 相等）。
func (x *ExactMemoryIndex) Lookup(scopeNorm, query string) (id, dat string, ok bool) {
	x.mu.RLock()
	defer x.mu.RUnlock()
	e, ok := x.m[compositeKey(scopeNorm, query)]
	if !ok {
		return "", "", false
	}
	return e.ID, e.DAT, true
}
