package rules

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"time"

	"github.com/google/uuid"
	"github.com/redis/go-redis/v9"
)

const (
	redisKeyRegexRulePrefix = "rag:regex:rule:"
	redisSetRegexIDs        = "rag:regex:ids"
	redisChannelRegex       = "rag:regex:changed"
)

// ErrInvalidRegex 正则无法编译（FR-A02）。
var ErrInvalidRegex = errors.New("rules: invalid regex pattern")

// ErrRegexNotFound 规则不存在。
var ErrRegexNotFound = errors.New("rules: regex rule not found")

// RedisRegexStore 正则规则 Redis 持久化与通知。
type RedisRegexStore struct {
	rdb *redis.Client
}

// NewRedisRegexStore 构造存储。
func NewRedisRegexStore(rdb *redis.Client) *RedisRegexStore {
	return &RedisRegexStore{rdb: rdb}
}

// ChannelRegex 返回 Pub/Sub 频道名。
func ChannelRegex() string { return redisChannelRegex }

func compilePattern(pattern string) error {
	if pattern == "" {
		return ErrInvalidRegex
	}
	if _, err := regexp.Compile(pattern); err != nil {
		return fmt.Errorf("%w: %v", ErrInvalidRegex, err)
	}
	return nil
}

// Create 创建规则；非法正则应在上层映射为 400。
func (s *RedisRegexStore) Create(ctx context.Context, in RegexRuleCreate) (RegexRule, error) {
	if err := compilePattern(in.Pattern); err != nil {
		return RegexRule{}, err
	}

	id := uuid.NewString()
	now := time.Now().UTC()
	rule := RegexRule{
		ID:        id,
		Scope:     in.Scope,
		Pattern:   in.Pattern,
		DAT:       in.DAT,
		Priority:  in.Priority,
		CreatedAt: now,
		UpdatedAt: now,
	}
	b, err := json.Marshal(rule)
	if err != nil {
		return RegexRule{}, err
	}

	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, redisKeyRegexRulePrefix+id, b, 0)
	pipe.SAdd(ctx, redisSetRegexIDs, id)
	pipe.Publish(ctx, redisChannelRegex, "reload")
	_, err = pipe.Exec(ctx)
	if err != nil {
		return RegexRule{}, err
	}
	return rule, nil
}

// GetByID 按 ID 读取规则。
func (s *RedisRegexStore) GetByID(ctx context.Context, id string) (RegexRule, error) {
	b, err := s.rdb.Get(ctx, redisKeyRegexRulePrefix+id).Bytes()
	if err == redis.Nil {
		return RegexRule{}, ErrRegexNotFound
	}
	if err != nil {
		return RegexRule{}, err
	}
	var r RegexRule
	if err := json.Unmarshal(b, &r); err != nil {
		return RegexRule{}, err
	}
	return r, nil
}

// ListAll 返回全部规则（用于内存重建）。
func (s *RedisRegexStore) ListAll(ctx context.Context) ([]RegexRule, error) {
	ids, err := s.rdb.SMembers(ctx, redisSetRegexIDs).Result()
	if err != nil {
		return nil, err
	}
	out := make([]RegexRule, 0, len(ids))
	for _, id := range ids {
		b, err := s.rdb.Get(ctx, redisKeyRegexRulePrefix+id).Bytes()
		if err == redis.Nil {
			continue
		}
		if err != nil {
			return nil, err
		}
		var r RegexRule
		if err := json.Unmarshal(b, &r); err != nil {
			return nil, err
		}
		out = append(out, r)
	}
	return out, nil
}

// Update 全量替换规则 JSON（协调器已合并 patch）。
func (s *RedisRegexStore) Update(ctx context.Context, rule RegexRule) error {
	if err := compilePattern(rule.Pattern); err != nil {
		return err
	}
	b, err := json.Marshal(rule)
	if err != nil {
		return err
	}
	pipe := s.rdb.Pipeline()
	pipe.Set(ctx, redisKeyRegexRulePrefix+rule.ID, b, 0)
	pipe.Publish(ctx, redisChannelRegex, "reload")
	_, err = pipe.Exec(ctx)
	return err
}

// Delete 删除规则。
func (s *RedisRegexStore) Delete(ctx context.Context, id string) error {
	pipe := s.rdb.Pipeline()
	pipe.Del(ctx, redisKeyRegexRulePrefix+id)
	pipe.SRem(ctx, redisSetRegexIDs, id)
	pipe.Publish(ctx, redisChannelRegex, "reload")
	_, err := pipe.Exec(ctx)
	return err
}
