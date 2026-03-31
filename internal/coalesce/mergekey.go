package coalesce

import "strings"

// ScopePrefixFromMergeKey 从 Key(scope, query) 结果中取出 scope 段（\\x00 之前）。
func ScopePrefixFromMergeKey(mergeKey string) string {
	i := strings.IndexByte(mergeKey, 0)
	if i < 0 {
		return ""
	}
	return mergeKey[:i]
}
