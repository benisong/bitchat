package config

import "time"

// 稳健性常量：全局可热调参
const (
	// 铁闸
	QuotaWindowSec    = 60
	MinQuotaPerWindow = 5  // 公共节点每账户每分钟的硬下限
	MaxQuotaPerWindow = 10 // 任何身份都不能超过的硬上限

	// 积分赚取
	CreditWitness            int64 = 1
	CreditRelayQueryAddr     int64 = 50
	CreditRelayDelivery      int64 = 1000
	CreditRelayOpportunistic int64 = 1000

	// Relay 付费
	RelayRealtimeBaseFee   int64 = 1 // 不再 100，让一個信產最小消耗等於赚取一次见证人
	RelayRealtimePerKB     int64 = 1
	RelayRealtimeMaxPerEnv int64 = 10
	RelayOfflineStoreFee   int64 = 1 // 同样降低，最小付费制
	RelayOfflinePerDayExt  int64 = 1

	// DHT PoW
	DHTPoWPrefix             = "FARP_DHT_V1"
	DHTPoWDefaultLeadingBits = 22
	DHTPoWMinLeadingBits     = 18
	DHTPoWMaxLeadingBits     = 26
	DHTPoWAvgTargetMs        = 1200 // 目标均值，难度动态调整
	DHTPoWSamplesPerEpoch    = 1008

	// 路由缓存
	RouteTTLDefault = 2 * time.Hour
	RouteTTLStable  = 24 * time.Hour
	RouteCacheTable = "routes"

	// 离线碎片
	OfflineFragmentCopies  = 3
	OfflineFragmentTTL     = 30 * 24 * time.Hour // 30天
	OfflineFragmentMaxSize = 1024 * 1024         // 1MB 单片上限

	// 消息
	MaxPendingSkippedKeys  = 500
	MaxOutboxBatchInterval = 10 * time.Second

	// 移民
	AuthExpireSec          = 300
	MigrationListenTimeout = 120 * time.Second
)
