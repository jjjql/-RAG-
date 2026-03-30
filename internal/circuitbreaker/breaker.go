// Package circuitbreaker 提供最小熔断器（SYS-ENG-01：失败累积开路 + 时间窗恢复）。
package circuitbreaker

import (
	"errors"
	"sync"
	"time"
)

// ErrOpen 开路状态下拒绝调用（快速失败 / 降级）。
var ErrOpen = errors.New("circuitbreaker: open")

// Breaker 连续失败达到阈值后进入开路一段时间；成功后关闭。
type Breaker struct {
	mu sync.Mutex

	maxFailures  int
	openDuration time.Duration

	failures  int
	openUntil time.Time
}

// New 创建熔断器；maxFailures<=0 时默认为 5；openDuration<=0 时默认为 30s。
func New(maxFailures int, openDuration time.Duration) *Breaker {
	if maxFailures <= 0 {
		maxFailures = 5
	}
	if openDuration <= 0 {
		openDuration = 30 * time.Second
	}
	return &Breaker{maxFailures: maxFailures, openDuration: openDuration}
}

// Allow 是否允许发起下游调用；false 时调用方应直接返回 ErrOpen。
func (b *Breaker) Allow() bool {
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	if !b.openUntil.IsZero() {
		if now.Before(b.openUntil) {
			return false
		}
		// 开路窗口结束：进入关闭态并清零失败计数
		b.openUntil = time.Time{}
		b.failures = 0
	}
	return true
}

// Success 记录一次成功，清零失败计数并确保关闭。
func (b *Breaker) Success() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures = 0
	b.openUntil = time.Time{}
}

// Fail 记录一次失败；达到阈值则开路。
func (b *Breaker) Fail() {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.failures++
	if b.failures >= b.maxFailures {
		b.openUntil = time.Now().Add(b.openDuration)
		b.failures = 0
	}
}
