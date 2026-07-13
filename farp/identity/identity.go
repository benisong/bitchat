package identity

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"fmt"
)

// Identity 包含当前节点的签名密钥对及会话状态
// 所有私密材料保留在这里，不外泄。

// NodeID 以 base58 或 base64 URL-safe 表示 Ed25519 公钥
// 本地存储只有 (pubkey, privkey) bytes

type KeyPair struct {
	Pub  ed25519.PublicKey
	Priv ed25519.PrivateKey
}

type Node struct {
	ID       string // base64.RawURLEncoding(pubkey)
	SignKey  *KeyPair
	PreKey   []byte // X25519 static key (for Noise + Double Ratchet init)
	Epoch    uint64 // 设备迁移纪元
	DeviceID string // 每台设备唯一雪啴值
}

// Generate 生成新节点身份
func Generate() (*Node, error) {
	_, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("ed25519 gen: %w", err)
	}
	pub := priv.Public().(ed25519.PublicKey)
	id := base64.RawURLEncoding.EncodeToString(pub)

	// PreKey: 从 ed25519 私钥派生 X25519，不是最优做法，先做 MVP，后续用 Curve25519 独立生成
	pre := make([]byte, 32)
	if _, err := rand.Read(pre); err != nil {
		return nil, fmt.Errorf("prekey gen: %w", err)
	}

	return &Node{
		ID:       id,
		SignKey:  &KeyPair{Pub: pub, Priv: priv},
		PreKey:   pre,
		Epoch:    0,
		DeviceID: mustUUID4(),
	}, nil
}

// Sign 用 SignKey 签名隨机消息
func (n *Node) Sign(msg []byte) []byte {
	return ed25519.Sign(n.SignKey.Priv, msg)
}

func (n *Node) Verify(msg, sig []byte) bool {
	return ed25519.Verify(n.SignKey.Pub, msg, sig)
}

// 简易雪啴用内容替代
func mustUUID4() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}
