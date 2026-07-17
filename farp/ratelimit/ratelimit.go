package ratelimit

import (
	"sync"
	"time"

	"github.com/benisong/bitchat/internal/config"
)

// TokenBucket 按 (peerID) 维护令牌桶
// 保护个人节点硬件不被恶意轰炸

type TokenBucket struct {
	mu         sync.Mutex
	capacity   float64
	tokens     float64
	lastRefill time.Time
}

// Manager 管理所有客户端的铁闸配额

type Manager struct {
	mu       sync.RWMutex
	buckets  map[string]*TokenBucket // key: peerID
	capacity float64                 // 当前全局容量
}

func NewManager() *Manager {
	return &Manager{
		buckets:  make(map[string]*TokenBucket),
		capacity: config.MinQuotaPerWindow,
	}
}

// SetCapacity configures a node-wide capacity while preserving the 5-10 hard gate.
func (m *Manager) SetCapacity(c int) {
	if c < config.MinQuotaPerWindow {
		c = config.MinQuotaPerWindow
	}
	if c > config.MaxQuotaPerWindow {
		c = config.MaxQuotaPerWindow
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	m.capacity = float64(c)
	for _, bucket := range m.buckets {
		bucket.mu.Lock()
		bucket.capacity = m.capacity
		bucket.tokens = min(bucket.tokens, bucket.capacity)
		bucket.mu.Unlock()
	}
}

// forKey 拿到某个 peerID 的 bucket
func (m *Manager) forKey(peerID string) *TokenBucket {
	m.mu.RLock()
	if b, ok := m.buckets[peerID]; ok {
		m.mu.RUnlock()
		return b
	}
	m.mu.RUnlock()

	m.mu.Lock()
	// 再检一次
	if b, ok := m.buckets[peerID]; ok {
		m.mu.Unlock()
		return b
	}
	b := &TokenBucket{
		capacity:   m.capacity,
		tokens:     m.capacity,
		lastRefill: time.Now(),
	}
	m.buckets[peerID] = b
	m.mu.Unlock()
	return b
}

// Deduce 尝试消耗 1 个令牌。成功返回 true，否则 false。
// opType 保留扩展
func (m *Manager) Deduce(peerID string, opType string) bool {
	b := m.forKey(peerID)
	b.mu.Lock()
	defer b.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	refill := elapsed / float64(config.QuotaWindowSec) * b.capacity
	if refill > 0 {
		b.tokens = min(b.capacity, b.tokens+refill)
		b.lastRefill = now
	}

	if b.tokens < 1 {
		return false
	}
	b.tokens--
	return true
}

// Peek 查看当前余额（仅用于 status 显示）
func (m *Manager) Peek(peerID string) (tokens float64, capacity float64) {
	b := m.forKey(peerID)
	b.mu.Lock()
	defer b.mu.Unlock()
	now := time.Now()
	elapsed := now.Sub(b.lastRefill).Seconds()
	refill := elapsed / float64(config.QuotaWindowSec) * b.capacity
	if refill > 0 {
		return min(b.capacity, b.tokens+refill), b.capacity
	}
	return b.tokens, b.capacity
}

// ReportError 用于高频帧打包后投递失败时标记
func (m *Manager) ReportError(peerID string, err error) {
	// TODO: 错误统计/劣迹黑名单
	_ = err
}

func min(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
