package rules

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	redisKeyRulePrefix = "rag:exact:rule:"
	redisKeyIdxPrefix  = "rag:exact:idx:"
	redisSetIDs        = "rag:exact:ids"
	redisChannelExact  = "rag:exact:changed"
)

// ErrConflict 同作用域下 KEY 已存在。
var ErrConflict = errors.New("rules: exact key conflict")

// RedisExactStore 精确规则 Redis 持久化与通知。
type RedisExactStore struct {
	rdb *redis.Client
}

// NewRedisExactStore 构造存储。
func NewRedisExactStore(rdb *redis.Client) *RedisExactStore {
	return &RedisExactStore{rdb: rdb}
}

func idxHash(scopeNorm, key string) string {
	h := sha256.Sum256([]byte(scopeNorm + "\n" + key))
	return hex.EncodeToString(h[:])
}

// Create 创建规则；冲突返回 ErrConflict。
func (s *RedisExactStore) Create(ctx context.Context, in ExactRuleCreate) (ExactRule, error) {
	scopeNorm := ScopeKey(in.Scope)
	if in.Key == "" {
		return ExactRule{}, fmt.Errorf("key 为空")
	}

	id := uuid.NewString()
	now := time.Now().UTC()
	rule := ExactRule{
		ID:        id,
		Scope:     in.Scope,
		Key:       in.Key,
		DAT:       in.DAT,
		CreatedAt: now,
		UpdatedAt: now,
	}
	b, err := json.Marshal(rule)
	if err != nil {
		return ExactRule{}, err
	}

	idxKey := redisKeyIdxPrefix + idxHash(scopeNorm, in.Key)
	ok, err := s.rdb.SetNX(ctx, idxKey, id, 0).Result()
	if err != nil {
		return ExactRule{}, err
	}
	if !ok {
		return ExactRule{}, ErrConflict
	}

	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, redisKeyRulePrefix+id, b, 0)
	pipe.SAdd(ctx, redisSetIDs, id)
	pipe.Publish(ctx, redisChannelExact, "reload")
	_, err = pipe.Exec(ctx)
	if err != nil {
		_ = s.rdb.Del(ctx, idxKey).Err()
		return ExactRule{}, err
	}
	return rule, nil
}

// GetByID 按 ID 读取规则。
func (s *RedisExactStore) GetByID(ctx context.Context, id string) (ExactRule, error) {
	b, err := s.rdb.Get(ctx, redisKeyRulePrefix+id).Bytes()
	if err == redis.Nil {
		return ExactRule{}, redis.Nil
	}
	if err != nil {
		return ExactRule{}, err
	}
	var r ExactRule
	if err := json.Unmarshal(b, &r); err != nil {
		return ExactRule{}, err
	}
	return r, nil
}

// ListAll 返回全部规则（用于内存重建）。
func (s *RedisExactStore) ListAll(ctx context.Context) ([]ExactRule, error) {
	ids, err := s.rdb.SMembers(ctx, redisSetIDs).Result()
	if err != nil {
		return nil, err
	}
	out := make([]ExactRule, 0, len(ids))
	for _, id := range ids {
		b, err := s.rdb.Get(ctx, redisKeyRulePrefix+id).Bytes()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			return nil, err
		}
		var r ExactRule
		if err := json.Unmarshal(b, &r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

// ChannelExact 返回 Pub/Sub 频道名。
func ChannelExact() string { return redisChannelExact }

// Update 按 ID 部分更新；变更 (scope,key) 时重建索引；冲突返回 ErrConflict。
func (s *RedisExactStore) Update(ctx context.Context, id string, p ExactRulePatch) (ExactRule, error) {
	if !p.HasAny() {
		return ExactRule{}, fmt.Errorf("rules: patch 为空")
	}
	old, err := s.GetByID(ctx, id)
	if err == redis.Nil {
		return ExactRule{}, redis.Nil
	}
	if err != nil {
		return ExactRule{}, err
	}
	merged := mergeExactPatch(old, p)
	if strings.TrimSpace(merged.Key) == "" {
		return ExactRule{}, fmt.Errorf("rules: key 不能为空")
	}

	oldIdx := idxHash(ScopeKey(old.Scope), old.Key)
	newIdx := idxHash(ScopeKey(merged.Scope), merged.Key)
	merged.UpdatedAt = time.Now().UTC()

	if newIdx != oldIdx {
		other, err := s.rdb.Get(ctx, redisKeyIdxPrefix+newIdx).Result()
		if err != nil && err != redis.Nil {
			return ExactRule{}, err
		}
		if other != "" && other != id {
			return ExactRule{}, ErrConflict
		}
	}

	b, err := json.Marshal(merged)
	if err != nil {
		return ExactRule{}, err
	}

	pipe := s.rdb.Pipeline()
	if newIdx != oldIdx {
		pipe.Del(ctx, redisKeyIdxPrefix+oldIdx)
		pipe.Set(ctx, redisKeyIdxPrefix+newIdx, id, 0)
	}
	pipe.Set(ctx, redisKeyRulePrefix+id, b, 0)
	pipe.Publish(ctx, redisChannelExact, "reload")
	if _, err := pipe.Exec(ctx); err != nil {
		return ExactRule{}, err
	}
	return merged, nil
}

func mergeExactPatch(old ExactRule, p ExactRulePatch) ExactRule {
	out := old
	if p.Scope != nil {
		if strings.TrimSpace(*p.Scope) == "" {
			out.Scope = nil
		} else {
			s := strings.TrimSpace(*p.Scope)
			out.Scope = &s
		}
	}
	if p.Key != nil {
		out.Key = strings.TrimSpace(*p.Key)
	}
	if p.DAT != nil {
		out.DAT = *p.DAT
	}
	return out
}

// Delete 按 ID 删除规则及索引。
func (s *RedisExactStore) Delete(ctx context.Context, id string) error {
	rule, err := s.GetByID(ctx, id)
	if err == redis.Nil {
		return redis.Nil
	}
	if err != nil {
		return err
	}
	idx := idxHash(ScopeKey(rule.Scope), rule.Key)
	pipe := s.rdb.Pipeline()
	pipe.Del(ctx, redisKeyRulePrefix+id)
	pipe.SRem(ctx, redisSetIDs, id)
	pipe.Del(ctx, redisKeyIdxPrefix+idx)
	pipe.Publish(ctx, redisChannelExact, "reload")
	_, err = pipe.Exec(ctx)
	return err
}
