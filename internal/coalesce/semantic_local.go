package coalesce

import (
	"context"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"
)

// SemanticLocal 进程内按「同 scope 下 embedding 余弦相似度 ≥ 阈值」合并 RAG。
type SemanticLocal struct {
	mu          sync.Mutex
	mergeTO     time.Duration
	threshold   float64
	byScope     map[string][]*semanticLocalLeader
	sfFactory   func() *singleflight.Group // 测试可注入；默认每 leader 一个 Group
}

type semanticLocalLeader struct {
	emb []float64
	g   *singleflight.Group
}

// NewSemanticLocal 构造；mergeTO 为单次下游超时；threshold 建议 0.95。
func NewSemanticLocal(mergeTO time.Duration, threshold float64) *SemanticLocal {
	if mergeTO <= 0 {
		mergeTO = 100 * time.Millisecond
	}
	if threshold <= 0 {
		threshold = 0.95
	}
	return &SemanticLocal{
		mergeTO:   mergeTO,
		threshold: threshold,
		byScope:   make(map[string][]*semanticLocalLeader),
		sfFactory: func() *singleflight.Group { return &singleflight.Group{} },
	}
}

// Merge 在 mergeKey 的 scope 分区内寻找相似 leader；否则新建组。
func (s *SemanticLocal) Merge(ctx context.Context, mergeKey string, emb []float64, fn func(context.Context) (string, error)) (string, error) {
	if s == nil {
		return fn(ctx)
	}
	if len(emb) == 0 {
		return fn(ctx)
	}
	scope := ScopePrefixFromMergeKey(mergeKey)

	s.mu.Lock()
	leaders := s.byScope[scope]
	var hit *semanticLocalLeader
	for _, L := range leaders {
		sim, ok := CosineSimilarity(L.emb, emb)
		if ok && sim >= s.threshold {
			hit = L
			break
		}
	}
	if hit == nil {
		g := s.sfFactory()
		hit = &semanticLocalLeader{emb: cloneFloat64s(emb), g: g}
		s.byScope[scope] = append(leaders, hit)
	}
	s.mu.Unlock()

	v, err, _ := hit.g.Do("rag", func() (interface{}, error) {
		c2, cancel := context.WithTimeout(context.Background(), s.mergeTO)
		defer cancel()
		text, e := fn(c2)
		s.mu.Lock()
		s.removeLeader(scope, hit)
		s.mu.Unlock()
		return text, e
	})
	if err != nil {
		return "", err
	}
	return v.(string), nil
}

func (s *SemanticLocal) removeLeader(scope string, L *semanticLocalLeader) {
	arr := s.byScope[scope]
	out := arr[:0]
	for _, x := range arr {
		if x != L {
			out = append(out, x)
		}
	}
	if len(out) == 0 {
		delete(s.byScope, scope)
	} else {
		s.byScope[scope] = out
	}
}

func cloneFloat64s(v []float64) []float64 {
	out := make([]float64, len(v))
	copy(out, v)
	return out
}
