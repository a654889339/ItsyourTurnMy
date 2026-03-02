package service

import (
	"sync"
	"time"
)

// RateLimiter IP请求频率限制器
type RateLimiter struct {
	records map[string]*rateLimitRecord
	mu      sync.RWMutex
	// 配置参数
	windowSize    time.Duration // 时间窗口
	maxRequests   int           // 窗口内最大请求数
	blockDuration time.Duration // 超限后封禁时长
}

type rateLimitRecord struct {
	requests    []time.Time // 请求时间记录
	blockedAt   time.Time   // 封禁开始时间
	isBlocked   bool        // 是否被封禁
}

// 默认配置：1分钟内最多10次请求，超限后封禁5分钟
const (
	DefaultRateLimitWindow    = 1 * time.Minute
	DefaultRateLimitMax       = 10
	DefaultRateLimitBlockTime = 5 * time.Minute
)

// NewRateLimiter 创建速率限制器
func NewRateLimiter(windowSize time.Duration, maxRequests int, blockDuration time.Duration) *RateLimiter {
	if windowSize == 0 {
		windowSize = DefaultRateLimitWindow
	}
	if maxRequests == 0 {
		maxRequests = DefaultRateLimitMax
	}
	if blockDuration == 0 {
		blockDuration = DefaultRateLimitBlockTime
	}

	rl := &RateLimiter{
		records:       make(map[string]*rateLimitRecord),
		windowSize:    windowSize,
		maxRequests:   maxRequests,
		blockDuration: blockDuration,
	}

	// 启动清理协程
	go rl.cleanupLoop()

	return rl
}

// Allow 检查IP是否允许请求
// 返回: 是否允许, 剩余封禁秒数（如果被封禁）
func (rl *RateLimiter) Allow(ip string) (bool, int) {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	record, exists := rl.records[ip]

	if !exists {
		// 新IP，创建记录
		rl.records[ip] = &rateLimitRecord{
			requests: []time.Time{now},
		}
		return true, 0
	}

	// 检查是否在封禁期
	if record.isBlocked {
		elapsed := now.Sub(record.blockedAt)
		if elapsed < rl.blockDuration {
			remaining := int((rl.blockDuration - elapsed).Seconds())
			return false, remaining
		}
		// 封禁期已过，重置状态
		record.isBlocked = false
		record.requests = []time.Time{now}
		return true, 0
	}

	// 清理窗口外的旧请求记录
	windowStart := now.Add(-rl.windowSize)
	validRequests := make([]time.Time, 0, len(record.requests))
	for _, t := range record.requests {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}

	// 检查是否超过限制
	if len(validRequests) >= rl.maxRequests {
		// 超限，开始封禁
		record.isBlocked = true
		record.blockedAt = now
		record.requests = validRequests
		return false, int(rl.blockDuration.Seconds())
	}

	// 记录本次请求
	validRequests = append(validRequests, now)
	record.requests = validRequests
	return true, 0
}

// GetRemainingRequests 获取剩余可用请求数
func (rl *RateLimiter) GetRemainingRequests(ip string) int {
	rl.mu.RLock()
	defer rl.mu.RUnlock()

	record, exists := rl.records[ip]
	if !exists {
		return rl.maxRequests
	}

	if record.isBlocked {
		return 0
	}

	now := time.Now()
	windowStart := now.Add(-rl.windowSize)
	count := 0
	for _, t := range record.requests {
		if t.After(windowStart) {
			count++
		}
	}

	remaining := rl.maxRequests - count
	if remaining < 0 {
		remaining = 0
	}
	return remaining
}

// cleanupLoop 定期清理过期记录
func (rl *RateLimiter) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rl.cleanup()
	}
}

// cleanup 清理过期记录
func (rl *RateLimiter) cleanup() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	// 清理窗口+封禁时间的2倍之前的记录
	expireThreshold := rl.windowSize + rl.blockDuration*2

	for ip, record := range rl.records {
		// 如果被封禁且封禁期已过很久
		if record.isBlocked && now.Sub(record.blockedAt) > expireThreshold {
			delete(rl.records, ip)
			continue
		}

		// 如果没有被封禁且所有请求都很旧
		if !record.isBlocked && len(record.requests) > 0 {
			latestRequest := record.requests[len(record.requests)-1]
			if now.Sub(latestRequest) > expireThreshold {
				delete(rl.records, ip)
			}
		}
	}
}

// 全局速率限制器实例
var (
	// 登录请求限制：1分钟最多5次，超限封禁15分钟
	LoginRateLimiter *RateLimiter
	// 注册请求限制：1分钟最多3次，超限封禁30分钟
	RegisterRateLimiter *RateLimiter
	// 发送验证码限制：1分钟最多2次，超限封禁10分钟
	SendCodeRateLimiter *RateLimiter
	// 公开API限制：1分钟最多30次，超限封禁5分钟
	PublicAPIRateLimiter *RateLimiter
)

// InitRateLimiters 初始化速率限制器
func InitRateLimiters() {
	LoginRateLimiter = NewRateLimiter(1*time.Minute, 5, 15*time.Minute)
	RegisterRateLimiter = NewRateLimiter(1*time.Minute, 3, 30*time.Minute)
	SendCodeRateLimiter = NewRateLimiter(1*time.Minute, 2, 10*time.Minute)
	PublicAPIRateLimiter = NewRateLimiter(1*time.Minute, 30, 5*time.Minute)
}
