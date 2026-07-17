package messaging

import (
	"crypto/ed25519"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/benisong/bitchat/farp/identity"
	"github.com/benisong/bitchat/farp/protocol"
)

const (
	CurrentEnvelopeVersion      uint16 = 1
	CurrentAuthorizationVersion uint16 = 1
	MessageIDSize                      = 16
	TaskIDSize                         = 16
	MaxEnvelopeCiphertext              = 1024 * 1024
	MaxEnvelopeHops             uint16 = 16
	MaxEnvelopeLifetime                = 30 * 24 * time.Hour
	MaxAuthorizationLifetime           = 24 * time.Hour
	MaxFutureClockSkew                 = 5 * time.Minute
)

var (
	ErrInvalidEnvelope      = errors.New("invalid FARP envelope")
	ErrInvalidSignature     = errors.New("invalid protocol signature")
	ErrEnvelopeExpired      = errors.New("FARP envelope expired")
	ErrInvalidAuthorization = errors.New("invalid service authorization")
)

type Envelope struct {
	Version             uint16
	MessageID           []byte
	TaskID              []byte
	SenderAccountPubkey []byte
	RecipientPubkey     []byte
	DeviceCertificate   identity.DeviceCertificate
	EphemeralPubkey     []byte
	Ciphertext          []byte
	Nonce               []byte
	CreatedAtUnix       int64
	ExpiresAtUnix       int64
	MaxHops             uint16
	Type                EnvelopeType
	Signature           []byte
}

type EnvelopeType uint8

const (
	TypeDirectMessage EnvelopeType = iota
	TypeOfflineFragment
	TypeAddressQueryReq
	TypeAddressQueryResp
	TypeMigrationAnnounce
	TypeWitnessSignature
	TypePing
	TypePong
	TypeRelayRequest
	TypeRelayDelivery
	TypeFragmentACK
	TypeSoloMigration
)

func (t EnvelopeType) valid() bool {
	return t >= TypeDirectMessage && t <= TypeSoloMigration
}

func NewEnvelope(
	node *identity.Node,
	recipientPubkey []byte,
	ephemeralPubkey []byte,
	ciphertext []byte,
	nonce []byte,
	typeValue EnvelopeType,
	ttl time.Duration,
	maxHops uint16,
) (*Envelope, error) {
	messageID := make([]byte, MessageIDSize)
	if _, err := rand.Read(messageID); err != nil {
		return nil, fmt.Errorf("message id: %w", err)
	}
	now := time.Now().UTC()
	envelope := &Envelope{
		Version:         CurrentEnvelopeVersion,
		MessageID:       messageID,
		RecipientPubkey: cloneBytes(recipientPubkey),
		EphemeralPubkey: cloneBytes(ephemeralPubkey),
		Ciphertext:      cloneBytes(ciphertext),
		Nonce:           cloneBytes(nonce),
		CreatedAtUnix:   now.Unix(),
		ExpiresAtUnix:   now.Add(ttl).Unix(),
		MaxHops:         maxHops,
		Type:            typeValue,
	}
	if err := envelope.Sign(node); err != nil {
		return nil, err
	}
	return envelope, nil
}

func (e *Envelope) SigningBytes() ([]byte, error) {
	certificate, err := e.DeviceCertificate.CanonicalBytes()
	if err != nil {
		return nil, err
	}
	b := protocol.NewBuilder("FARP_ENVELOPE_V1")
	b.Uint16(e.Version)
	b.Field(e.MessageID)
	b.Field(e.TaskID)
	b.Field(e.SenderAccountPubkey)
	b.Field(e.RecipientPubkey)
	b.Field(certificate)
	b.Field(e.EphemeralPubkey)
	b.Field(e.Ciphertext)
	b.Field(e.Nonce)
	b.Int64(e.CreatedAtUnix)
	b.Int64(e.ExpiresAtUnix)
	b.Uint16(e.MaxHops)
	b.Uint8(uint8(e.Type))
	return b.Build()
}

func (e *Envelope) Sign(node *identity.Node) error {
	if node == nil {
		return ErrInvalidEnvelope
	}
	if err := node.Validate(); err != nil {
		return err
	}
	e.Version = CurrentEnvelopeVersion
	e.SenderAccountPubkey = cloneBytes(node.AccountKey.Pub)
	e.DeviceCertificate = cloneCertificate(node.Certificate)
	if err := e.validate(time.Now().UTC(), false); err != nil {
		return err
	}
	payload, err := e.SigningBytes()
	if err != nil {
		return err
	}
	e.Signature = node.Sign(payload)
	return nil
}

func (e *Envelope) Verify(now time.Time) error {
	if err := e.validate(now.UTC(), true); err != nil {
		return err
	}
	if err := e.DeviceCertificate.Verify(); err != nil {
		return err
	}
	if !equalBytes(e.SenderAccountPubkey, e.DeviceCertificate.AccountPubkey) {
		return ErrInvalidEnvelope
	}
	payload, err := e.SigningBytes()
	if err != nil {
		return err
	}
	if !ed25519.Verify(ed25519.PublicKey(e.DeviceCertificate.DevicePubkey), payload, e.Signature) {
		return ErrInvalidSignature
	}
	return nil
}

func (e *Envelope) IsExpired(now time.Time) bool {
	return now.UTC().Unix() > e.ExpiresAtUnix
}

func (e *Envelope) validate(now time.Time, requireSignature bool) error {
	if e == nil ||
		e.Version != CurrentEnvelopeVersion ||
		len(e.MessageID) != MessageIDSize ||
		(len(e.TaskID) != 0 && len(e.TaskID) != TaskIDSize) ||
		len(e.SenderAccountPubkey) != ed25519.PublicKeySize ||
		len(e.RecipientPubkey) != ed25519.PublicKeySize ||
		len(e.EphemeralPubkey) != 32 ||
		len(e.Ciphertext) == 0 || len(e.Ciphertext) > MaxEnvelopeCiphertext ||
		(len(e.Nonce) != 12 && len(e.Nonce) != 24) ||
		e.CreatedAtUnix <= 0 ||
		e.ExpiresAtUnix <= e.CreatedAtUnix ||
		e.MaxHops == 0 || e.MaxHops > MaxEnvelopeHops ||
		!e.Type.valid() {
		return ErrInvalidEnvelope
	}
	createdAt := time.Unix(e.CreatedAtUnix, 0).UTC()
	expiresAt := time.Unix(e.ExpiresAtUnix, 0).UTC()
	if createdAt.After(now.Add(MaxFutureClockSkew)) || expiresAt.Sub(createdAt) > MaxEnvelopeLifetime {
		return ErrInvalidEnvelope
	}
	if now.After(expiresAt) {
		return ErrEnvelopeExpired
	}
	if requireSignature && len(e.Signature) != ed25519.SignatureSize {
		return ErrInvalidSignature
	}
	return nil
}

type AuthWrapper struct {
	Envelope *Envelope
	Auth     *ServiceAuthorization
}

// ServiceAuthorization allows one provider to consume a positive amount of local credit.
type ServiceAuthorization struct {
	Version             uint16
	TaskID              []byte
	SenderAccountPubkey []byte
	ProviderPubkey      []byte
	Amount              int64
	Nonce               []byte
	IssuedAtUnix        int64
	ExpiresAtUnix       int64
	DeviceCertificate   identity.DeviceCertificate
	Signature           []byte
}

func (a *ServiceAuthorization) SigningBytes() ([]byte, error) {
	certificate, err := a.DeviceCertificate.CanonicalBytes()
	if err != nil {
		return nil, err
	}
	b := protocol.NewBuilder("FARP_SERVICE_AUTHORIZATION_V1")
	b.Uint16(a.Version)
	b.Field(a.TaskID)
	b.Field(a.SenderAccountPubkey)
	b.Field(a.ProviderPubkey)
	b.Int64(a.Amount)
	b.Field(a.Nonce)
	b.Int64(a.IssuedAtUnix)
	b.Int64(a.ExpiresAtUnix)
	b.Field(certificate)
	return b.Build()
}

func (a *ServiceAuthorization) Sign(node *identity.Node) error {
	if node == nil {
		return ErrInvalidAuthorization
	}
	if err := node.Validate(); err != nil {
		return err
	}
	a.Version = CurrentAuthorizationVersion
	a.SenderAccountPubkey = cloneBytes(node.AccountKey.Pub)
	a.DeviceCertificate = cloneCertificate(node.Certificate)
	if err := a.validate(time.Now().UTC(), false); err != nil {
		return err
	}
	payload, err := a.SigningBytes()
	if err != nil {
		return err
	}
	a.Signature = node.Sign(payload)
	return nil
}

func (a *ServiceAuthorization) Verify(now time.Time) error {
	if err := a.validate(now.UTC(), true); err != nil {
		return err
	}
	if err := a.DeviceCertificate.Verify(); err != nil {
		return err
	}
	if !equalBytes(a.SenderAccountPubkey, a.DeviceCertificate.AccountPubkey) {
		return ErrInvalidAuthorization
	}
	payload, err := a.SigningBytes()
	if err != nil {
		return err
	}
	if !ed25519.Verify(ed25519.PublicKey(a.DeviceCertificate.DevicePubkey), payload, a.Signature) {
		return ErrInvalidSignature
	}
	return nil
}

func (a *ServiceAuthorization) validate(now time.Time, requireSignature bool) error {
	if a == nil ||
		a.Version != CurrentAuthorizationVersion ||
		len(a.TaskID) != TaskIDSize ||
		len(a.SenderAccountPubkey) != ed25519.PublicKeySize ||
		len(a.ProviderPubkey) != ed25519.PublicKeySize ||
		a.Amount <= 0 ||
		len(a.Nonce) != 16 ||
		a.IssuedAtUnix <= 0 ||
		a.ExpiresAtUnix <= a.IssuedAtUnix {
		return ErrInvalidAuthorization
	}
	issuedAt := time.Unix(a.IssuedAtUnix, 0).UTC()
	expiresAt := time.Unix(a.ExpiresAtUnix, 0).UTC()
	if issuedAt.After(now.Add(MaxFutureClockSkew)) || expiresAt.Sub(issuedAt) > MaxAuthorizationLifetime || now.After(expiresAt) {
		return ErrInvalidAuthorization
	}
	if requireSignature && len(a.Signature) != ed25519.SignatureSize {
		return ErrInvalidSignature
	}
	return nil
}

// OfflineFragment is deliberately opaque to custodians.
type OfflineFragment struct {
	Version       uint16
	TaskID        []byte
	Ciphertext    []byte
	NextHopToken  []byte
	ExpiresAtUnix int64
}

func cloneCertificate(value identity.DeviceCertificate) identity.DeviceCertificate {
	return identity.DeviceCertificate{
		Version:              value.Version,
		AccountPubkey:        cloneBytes(value.AccountPubkey),
		DevicePubkey:         cloneBytes(value.DevicePubkey),
		AgreementPubkey:      cloneBytes(value.AgreementPubkey),
		PreviousDevicePubkey: cloneBytes(value.PreviousDevicePubkey),
		DeviceID:             value.DeviceID,
		Epoch:                value.Epoch,
		IssuedAtUnix:         value.IssuedAtUnix,
		Signature:            cloneBytes(value.Signature),
	}
}

func cloneBytes(value []byte) []byte {
	cloned := make([]byte, len(value))
	copy(cloned, value)
	return cloned
}

func equalBytes(left, right []byte) bool {
	if len(left) != len(right) {
		return false
	}
	var different byte
	for i := range left {
		different |= left[i] ^ right[i]
	}
	return different == 0
}
