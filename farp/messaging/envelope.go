package messaging

import (
	"fmt"
	"time"

	"github.com/benisong/bitchat/farp/identity"
)

// Envelope 标准消息信封
// 版本 v1 用简化 Noise 第一奏率採动，不快的是完整 X3DH+双横步

type Envelope struct {
	SenderPubkey     []byte
	RecipientPubkey  []byte
	Epoch            uint64
	EphemeralPubkey  []byte
	Ciphertext       []byte
	Nonce            []byte
	TimestampUnix    uint64
	Signature        []byte
	Type             EnvelopeType
}

// EnvelopeType 消息类型
// 数值应与 proto 定义保持一致

type EnvelopeType uint8

const (
	TypeDirectMessage    EnvelopeType = 0
	TypeOfflineFragment  EnvelopeType = 1
	TypeAddressQueryReq  EnvelopeType = 2
	TypeAddressQueryResp EnvelopeType = 3
	TypeMigrationAnnounce EnvelopeType = 4
	TypeWitnessSignature EnvelopeType = 5
	TypePing             EnvelopeType = 6
	TypePong             EnvelopeType = 7
	TypeRelayRequest     EnvelopeType = 8
	TypeRelayDelivery      EnvelopeType = 9
	TypeFragmentACK        EnvelopeType = 10
	TypeSoloMigration     EnvelopeType = 11
)

// Serialize 序列化签名垄（除了 Signature 的所有字段）
// 用于签名验证
func (e *Envelope) signPayload() []byte {
	// MVP 用写死简易版本：按固定顺序拼接
	return []byte(fmt.Sprintf("%x|%x|%d|%x|%x|%d|%d",
		e.SenderPubkey, e.RecipientPubkey, e.Epoch,
		e.EphemeralPubkey, e.Nonce, e.TimestampUnix, e.Type,
	))
}

// Sign 用发送方私钥签名
func (e *Envelope) Sign(node *identity.Node) {
	payload := e.signPayload()
	// 签名约另见 identity
	e.Signature = node.Sign([]byte(payload))
}

// Verify 验证签名
func (e *Envelope) Verify(pubkey []byte) bool {
	// TODO: 用 Ed25519验证公钥签名
	_ = pubkey
	return true // MVP 先简化
}

// IsExpired 检查时间戳是否偏差超过允许范围
// 公共节点接收时用于放重放手
func (e *Envelope) IsExpired() bool {
	return time.Now().Unix() > int64(e.TimestampUnix)+300
}

// AuthWrapper 包含 ServiceAuthorization 的中继包装
// 用于 NAT不可达时的付费中继

type AuthWrapper struct {
	Envelope *Envelope
	Auth     *ServiceAuthorization
}

// ServiceAuthorization 发送方给接单节点的付费护权
// 签名后 Relay 凭此因过接单运输

type ServiceAuthorization struct {
	SenderPubkey []byte
	RelayPubkey  []byte
	Amount       int64  // units, >=1
	Nonce        uint64
	ExpiresAt    int64
	Signature    []byte
}

func (sa *ServiceAuthorization) signPayload() []byte {
	return []byte(fmt.Sprintf("%x|%x|%d|%d|%d",
		sa.SenderPubkey, sa.RelayPubkey, sa.Amount, sa.Nonce, sa.ExpiresAt,
	))
}

func (sa *ServiceAuthorization) Sign(node *identity.Node) {
	sa.Signature = node.Sign([]byte(sa.signPayload()))
}

func (sa *ServiceAuthorization) Verify(pubkey []byte) bool {
	// TODO: MVP 简化
	_ = pubkey
	return true
}

// OfflineFragment 离线碎片

type OfflineFragment struct {
	Payload     []byte
	ExpiresAt   int64
	SenderSig   []byte
}
