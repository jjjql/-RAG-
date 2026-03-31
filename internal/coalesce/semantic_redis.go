package coalesce

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

// SemanticRedis 跨实例：同 scope 下 embedding 余弦 ≥ 阈值则共用同一 groupId 的 lock/res。
type SemanticRedis struct {
	rdb       *redis.Client
	inner     *Redis
	threshold float64
	maxActive int
	vecTTL    time.Duration
}

// NewSemanticRedis 构造；maxActive 为单 scope 下活跃组上限，超出则退化为不合并（直接 fn）。
func NewSemanticRedis(rdb *redis.Client, cfg RedisConfig, threshold float64, maxActive int) *SemanticRedis {
	if threshold <= 0 {
		threshold = 0.95
	}
	if maxActive <= 0 {
		maxActive = 256
	}
	inner := NewRedis(rdb, cfg)
	vecTTL := cfg.LockTTL + cfg.ResultTTL + 60*time.Second
	if vecTTL <= 0 {
		vecTTL = 300 * time.Second
	}
	return &SemanticRedis{
		rdb:       rdb,
		inner:     inner,
		threshold: threshold,
		maxActive: maxActive,
		vecTTL:    vecTTL,
	}
}

func scopeActiveKey(scope string) string {
	return "rag:coalesce:sem:active:" + keyHex(scope)
}

func semVecKey(groupID string) string {
	return "rag:coalesce:sem:vec:" + groupID
}

func semLockKey(groupID string) string {
	return "rag:coalesce:sem:lock:" + groupID
}

func semResKey(groupID string) string {
	return "rag:coalesce:sem:res:" + groupID
}

// Merge 查找或创建语义组并走 Redis lock/res。
func (s *SemanticRedis) Merge(ctx context.Context, mergeKey string, emb []float64, fn func(context.Context) (string, error)) (string, error) {
	if s == nil || s.rdb == nil || s.inner == nil || len(emb) == 0 {
		return fn(ctx)
	}
	scope := ScopePrefixFromMergeKey(mergeKey)
	setKey := scopeActiveKey(scope)

	ids, err := s.rdb.SMembers(ctx, setKey).Result()
	if err != nil {
		return "", err
	}
	if len(ids) >= s.maxActive {
		return fn(ctx)
	}

	var chosen string
	for _, id := range ids {
		raw, gerr := s.rdb.Get(ctx, semVecKey(id)).Bytes()
		if gerr != nil {
			if errors.Is(gerr, redis.Nil) {
				_, _ = s.rdb.SRem(ctx, setKey, id).Result()
			}
			continue
		}
		var other []float64
		if json.Unmarshal(raw, &other) != nil || len(other) != len(emb) {
			continue
		}
		sim, ok := CosineSimilarity(emb, other)
		if ok && sim >= s.threshold {
			chosen = id
			break
		}
	}

	if chosen == "" {
		chosen = uuid.NewString()
		pipe := s.rdb.Pipeline()
		pipe.SAdd(ctx, setKey, chosen)
		b, _ := json.Marshal(emb)
		pipe.Set(ctx, semVecKey(chosen), b, s.vecTTL)
		if _, err := pipe.Exec(ctx); err != nil {
			return "", err
		}
	}

	lockK := semLockKey(chosen)
	resK := semResKey(chosen)
	text, err, leader := s.inner.doKeysWithRole(ctx, lockK, resK, fn)
	if leader {
		bg := context.Background()
		_ = s.rdb.Del(bg, semVecKey(chosen)).Err()
		_, _ = s.rdb.SRem(bg, setKey, chosen).Result()
	}
	if err != nil {
		return "", err
	}
	return text, nil
}
