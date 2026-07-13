package ledger

import (
	"sync"
	"time"

	"github.com/benisong/bitchat/internal/config"
)

// CreditRecord 本地视角的积分记录
// 由于无全局共识，所有记录均为"本地可验证的玉断断"

type CreditRecord struct {
	mu sync.RWMutex

	Pubkey            string
	Balance           int64 // units，1 Credit = 1
	Frozen            bool
	ContributionRatio int // 存储为千分比：1000=10.00%
	Last7DaysUpGB     int
	LastUpdated       time.Time
}

// FreezeIfSoulbound 如果积分高但贡献，创建在线率低于阈值，则冻结
func (cr *CreditRecord) FreezeCheck() {
	cr.mu.Lock()
	defer cr.mu.Unlock()

	// 千分比 < 10.00% 且余额 > 100，刚冻结（高能分但号）
	if cr.ContributionRatio < config.ContributionThreshold && cr.Balance > 100 {
		cr.Frozen = true
	} else if cr.ContributionRatio >= config.ContributionThreshold {
		cr.Frozen = false
	}
}

// Add 增加积分（见证人、中继成功等）
func (cr *CreditRecord) Add(amount int64) {
	cr.mu.Lock()
	cr.Balance += amount
	cr.LastUpdated = time.Now()
	cr.mu.Unlock()
}

// CanSpend 检查是否可以消耗指定数量
// 冻结状态下不能用于 Relay 支付提升等
func (cr *CreditRecord) CanSpend(amount int64) bool {
	cr.mu.RLock()
	defer cr.mu.RUnlock()
	return cr.Balance >= amount && !cr.Frozen
}

// Spend 扣减积分
func (cr *CreditRecord) Spend(amount int64) (ok bool) {
	cr.mu.Lock()
	defer cr.mu.Unlock()
	if cr.Balance < amount {
		return false
	}
	cr.Balance -= amount
	cr.LastUpdated = time.Now()
	return true
}
