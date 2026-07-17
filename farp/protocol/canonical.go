package protocol

import (
	"bytes"
	"encoding/binary"
	"errors"
	"math"
)

var ErrFieldTooLarge = errors.New("canonical field exceeds uint32 length")

// Builder creates deterministic, length-prefixed payloads for signatures.
type Builder struct {
	buf bytes.Buffer
	err error
}

func NewBuilder(domain string) *Builder {
	b := &Builder{}
	b.Field([]byte(domain))
	return b
}

func (b *Builder) Uint8(value uint8) {
	if b.err != nil {
		return
	}
	b.err = b.buf.WriteByte(value)
}

func (b *Builder) Uint16(value uint16) {
	if b.err != nil {
		return
	}
	var encoded [2]byte
	binary.BigEndian.PutUint16(encoded[:], value)
	_, b.err = b.buf.Write(encoded[:])
}

func (b *Builder) Uint64(value uint64) {
	if b.err != nil {
		return
	}
	var encoded [8]byte
	binary.BigEndian.PutUint64(encoded[:], value)
	_, b.err = b.buf.Write(encoded[:])
}

func (b *Builder) Int64(value int64) {
	b.Uint64(uint64(value))
}

func (b *Builder) Field(value []byte) {
	if b.err != nil {
		return
	}
	if uint64(len(value)) > math.MaxUint32 {
		b.err = ErrFieldTooLarge
		return
	}
	var size [4]byte
	binary.BigEndian.PutUint32(size[:], uint32(len(value)))
	if _, err := b.buf.Write(size[:]); err != nil {
		b.err = err
		return
	}
	_, b.err = b.buf.Write(value)
}

func (b *Builder) String(value string) {
	b.Field([]byte(value))
}

func (b *Builder) Build() ([]byte, error) {
	if b.err != nil {
		return nil, b.err
	}
	result := make([]byte, b.buf.Len())
	copy(result, b.buf.Bytes())
	return result, nil
}
