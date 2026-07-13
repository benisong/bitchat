package pow

import (
	"bytes"
	"crypto/sha256"
	"encoding/binary"
	"strconv"
	"time"

	"github.com/benisong/bitchat/internal/config"
)

// DHTPoWProof DHT 地址发布的微工作量证明
// 防投毒用：每次 Put 需要计算 hash，模糊说是 2^22 次

type DHTPoWProof struct {
	Nonce     uint64
	Timestamp uint32 // yyyyMMdd
	HashResult [32]byte
}

// Challenge 生成当前挑战内容
func Challenge(pubkey []byte, multiaddrs []byte) []byte {
	today := uint32(time.Now().UTC().Year()*10000 + int(time.Now().UTC().Month())*100 + time.Now().UTC().Day())
	buf := make([]byte, 4)
	binary.BigEndian.PutUint32(buf, today)
	out := make([]byte, 0, len(config.DHTPoWPrefix)+4+len(pubkey)+len(multiaddrs))
	out = append(out, []byte(config.DHTPoWPrefix)...)
	out = append(out, buf...)
	out = append(out, pubkey...)
	out = append(out, multiaddrs...)
	return out
}

// Verify 验证提供的 proof
func Verify(proof *DHTPoWProof, pubkey, multiaddrs []byte, difficulty int) bool {
	expected := todayUint32()
	if proof.Timestamp != expected {
		return false
	}
	ch := Challenge(pubkey, multiaddrs)
	// 重新计算
	data := make([]byte, 0, len(ch)+8)
	data = append(data, ch...)
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, proof.Nonce)
	data = append(data, b...)
	result := sha256.Sum256(data)
	if !bytes.Equal(result[:], proof.HashResult[:]) {
		return false
	}
	return countLeadingZeroBits(result[:]) >= difficulty
}

// Mine 按指定难度挖掘合适 nonce
func Mine(pubkey, multiaddrs []byte, difficulty int) *DHTPoWProof {
	ch := Challenge(pubkey, multiaddrs)
	today := todayUint32()
	var nonce uint64
	for {
		data := make([]byte, 0, len(ch)+8)
		data = append(data, ch...)
		b := make([]byte, 8)
		binary.BigEndian.PutUint64(b, nonce)
		data = append(data, b...)
		h := sha256.Sum256(data)
		if countLeadingZeroBits(h[:]) >= difficulty {
			return &DHTPoWProof{
				Nonce:      nonce,
				Timestamp:  today,
				HashResult: h,
			}
		}
		nonce++
	}
}

// todayUint32 UTC yyyyMMdd
func todayUint32() uint32 {
	t := time.Now().UTC()
	v, _ := strconv.Atoi(strconv.Itoa(t.Year()) + sprintfTwo(int(t.Month())) + sprintfTwo(t.Day()))
	return uint32(v)
}

func sprintfTwo(n int) string {
	if n < 10 {
		return "0" + strconv.Itoa(n)
	}
	return strconv.Itoa(n)
}

// countLeadingZeroBits 计算前导零位数
func countLeadingZeroBits(hash []byte) int {
	bits := 0
	for _, b := range hash {
		for i := 7; i >= 0; i-- {
			if (b>>i)&1 == 0 {
				bits++
			} else {
				return bits
			}
		}
	}
	return bits
}
