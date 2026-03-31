// Package coalesce 相似请求合并（RAG 下游 singleflight），见 specs/architecture/COALESCE_DESIGN.md。
package coalesce

import (
	"context"
	"strings"
	"time"

	"golang.org/x/sync/singleflight"
)

// RAG 对下游 Complete 同源请求合并；Enabled=false 时透传。
type RAG struct {
	Enabled bool
	// MergeTimeout 合并后单次下游调用的上限（通常与 downstream.timeout_ms 一致）。
	MergeTimeout time.Duration

	g singleflight.Group
}

// Key 规范化合并键（scope + query，见 COALESCE_DESIGN.md）。
func Key(scope *string, query string) string {
	sq := ""
	if scope != nil {
		sq = strings.TrimSpace(*scope)
	}
	q := strings.TrimSpace(strings.ToLower(query))
	return sq + "\x00" + q
}

// Do 若启用则同 key 并发只执行一次 fn；fn 使用独立超时上下文，避免首连接取消拖死其余等待方。
func (r *RAG) Do(ctx context.Context, key string, fn func(context.Context) (string, error)) (string, error) {
	if r == nil || !r.Enabled {
		return fn(ctx)
	}
	to := r.MergeTimeout
	if to <= 0 {
		to = 100 * time.Millisecond
	}
	v, err, _ := r.g.Do(key, func() (interface{}, error) {
		c2, cancel := context.WithTimeout(context.Background(), to)
		defer cancel()
		return fn(c2)
	})
	if err != nil {
		return "", err
	}
	return v.(string), nil
}
