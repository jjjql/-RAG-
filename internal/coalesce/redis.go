package coalesce

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

// RedisConfig 跨网关实例合并（与 rules 共用同一 Redis）。
type RedisConfig struct {
	MergeTimeout time.Duration // 单次下游 RAG 上限，通常对齐 downstream.timeout_ms
	LockTTL      time.Duration // SET NX 占锁时间，须明显大于 MergeTimeout
	ResultTTL    time.Duration // 结果键 TTL，供其它实例读到同一次应答（含短期缓存）
}

// Redis 使用 Redis 记录「进行中」与「已完成结果」，使不同机器上的网关对同键只跑一处下游。
type Redis struct {
	rdb *redis.Client
	cfg RedisConfig
}

// NewRedis 构造；rdb 非空。
func NewRedis(rdb *redis.Client, cfg RedisConfig) *Redis {
	if cfg.LockTTL <= 0 {
		cfg.LockTTL = 120 * time.Second
	}
	if cfg.ResultTTL <= 0 {
		cfg.ResultTTL = 90 * time.Second
	}
	if cfg.MergeTimeout <= 0 {
		cfg.MergeTimeout = 100 * time.Millisecond
	}
	return &Redis{rdb: rdb, cfg: cfg}
}

func keyHex(mergeKey string) string {
	h := sha256.Sum256([]byte(mergeKey))
	return hex.EncodeToString(h[:])
}

type storedResult struct {
	OK   bool   `json:"ok"`
	Text string `json:"text,omitempty"`
	Err  string `json:"err,omitempty"`
}

// Do 实现 Merger：先读结果缓存；再抢锁执行；否则轮询等待结果键。
func (r *Redis) Do(ctx context.Context, mergeKey string, fn func(context.Context) (string, error)) (string, error) {
	if r == nil || r.rdb == nil {
		return fn(ctx)
	}
	kh := keyHex(mergeKey)
	lockK := "rag:coalesce:lock:" + kh
	resK := "rag:coalesce:res:" + kh
	t, e, _ := r.doKeysWithRole(ctx, lockK, resK, fn)
	return t, e
}

// doKeysWithRole 使用显式 lock/res 键执行合并。leaderExecuted=true 表示本 goroutine 执行了 runLeader（需由调用方做语义组清理等）。
func (r *Redis) doKeysWithRole(ctx context.Context, lockK, resK string, fn func(context.Context) (string, error)) (text string, err error, leaderExecuted bool) {
	if r == nil || r.rdb == nil {
		t, e := fn(ctx)
		return t, e, true
	}
	if s, gerr := r.rdb.Get(ctx, resK).Result(); gerr == nil {
		t, e := decodeStored(s)
		return t, e, false
	} else if !errors.Is(gerr, redis.Nil) {
		return "", gerr, false
	}

	ok, err := r.rdb.SetNX(ctx, lockK, "1", r.cfg.LockTTL).Result()
	if err != nil {
		return "", err, false
	}
	if ok {
		t, e := r.runLeader(lockK, resK, fn)
		return t, e, true
	}
	t, e := r.waitFollower(ctx, resK)
	return t, e, false
}

func (r *Redis) runLeader(lockK, resK string, fn func(context.Context) (string, error)) (string, error) {
	c2, cancel := context.WithTimeout(context.Background(), r.cfg.MergeTimeout)
	defer cancel()
	text, err := fn(c2)
	var payload []byte
	if err != nil {
		payload, _ = json.Marshal(storedResult{OK: false, Err: err.Error()})
	} else {
		payload, _ = json.Marshal(storedResult{OK: true, Text: text})
	}
	bg := context.Background()
	pipe := r.rdb.Pipeline()
	pipe.Set(bg, resK, payload, r.cfg.ResultTTL)
	pipe.Del(bg, lockK)
	if _, perr := pipe.Exec(bg); perr != nil {
		return "", perr
	}
	if err != nil {
		return "", err
	}
	return text, nil
}

func (r *Redis) waitFollower(ctx context.Context, resK string) (string, error) {
	deadline := time.Now().Add(r.cfg.LockTTL + 10*time.Second)
	ticker := time.NewTicker(15 * time.Millisecond)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return "", ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return "", fmt.Errorf("coalesce redis: 等待同键结果超时")
			}
			s, err := r.rdb.Get(ctx, resK).Result()
			if err == nil {
				return decodeStored(s)
			}
			if !errors.Is(err, redis.Nil) {
				return "", err
			}
		}
	}
}

func decodeStored(s string) (string, error) {
	var out storedResult
	if err := json.Unmarshal([]byte(s), &out); err != nil {
		return "", err
	}
	if !out.OK {
		if out.Err == "" {
			return "", errors.New("coalesce redis: 下游失败")
		}
		return "", errors.New(out.Err)
	}
	return out.Text, nil
}
