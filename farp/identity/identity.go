package identity

import (
	"crypto/ecdh"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/benisong/bitchat/farp/protocol"
)

const (
	DeviceCertificateVersion uint16 = 1
	privateBundleVersion     uint16 = 1
)

var (
	ErrInvalidCertificate = errors.New("invalid device certificate")
	ErrInvalidPrivateData = errors.New("invalid identity private data")
)

type KeyPair struct {
	Pub  ed25519.PublicKey
	Priv ed25519.PrivateKey
}

type AgreementKeyPair struct {
	Pub  []byte
	Priv []byte
}

// DeviceCertificate binds one active device and agreement key to an account.
type DeviceCertificate struct {
	Version              uint16 `json:"version"`
	AccountPubkey        []byte `json:"account_pubkey"`
	DevicePubkey         []byte `json:"device_pubkey"`
	AgreementPubkey      []byte `json:"agreement_pubkey"`
	PreviousDevicePubkey []byte `json:"previous_device_pubkey,omitempty"`
	DeviceID             string `json:"device_id"`
	Epoch                uint64 `json:"epoch"`
	IssuedAtUnix         int64  `json:"issued_at_unix"`
	Signature            []byte `json:"signature"`
}

func (c DeviceCertificate) signingBytes() ([]byte, error) {
	b := protocol.NewBuilder("FARP_DEVICE_CERTIFICATE_V1")
	b.Uint16(c.Version)
	b.Field(c.AccountPubkey)
	b.Field(c.DevicePubkey)
	b.Field(c.AgreementPubkey)
	b.Field(c.PreviousDevicePubkey)
	b.String(c.DeviceID)
	b.Uint64(c.Epoch)
	b.Int64(c.IssuedAtUnix)
	return b.Build()
}

// CanonicalBytes includes the account signature and can be embedded in other signed objects.
func (c DeviceCertificate) CanonicalBytes() ([]byte, error) {
	payload, err := c.signingBytes()
	if err != nil {
		return nil, err
	}
	b := protocol.NewBuilder("FARP_DEVICE_CERTIFICATE_WIRE_V1")
	b.Field(payload)
	b.Field(c.Signature)
	return b.Build()
}

func (c DeviceCertificate) Verify() error {
	if c.Version != DeviceCertificateVersion ||
		len(c.AccountPubkey) != ed25519.PublicKeySize ||
		len(c.DevicePubkey) != ed25519.PublicKeySize ||
		len(c.AgreementPubkey) != 32 ||
		c.DeviceID == "" ||
		len(c.Signature) != ed25519.SignatureSize {
		return ErrInvalidCertificate
	}
	payload, err := c.signingBytes()
	if err != nil {
		return err
	}
	if !ed25519.Verify(ed25519.PublicKey(c.AccountPubkey), payload, c.Signature) {
		return ErrInvalidCertificate
	}
	return nil
}

type Node struct {
	ID           string
	AccountKey   *KeyPair
	DeviceKey    *KeyPair
	AgreementKey *AgreementKeyPair
	DeviceID     string
	Certificate  DeviceCertificate
}

func Generate() (*Node, error) {
	account, err := generateSigningKey()
	if err != nil {
		return nil, fmt.Errorf("account key: %w", err)
	}
	device, err := generateSigningKey()
	if err != nil {
		return nil, fmt.Errorf("device key: %w", err)
	}
	agreementPrivate, err := ecdh.X25519().GenerateKey(rand.Reader)
	if err != nil {
		return nil, fmt.Errorf("agreement key: %w", err)
	}
	deviceID, err := newDeviceID()
	if err != nil {
		return nil, fmt.Errorf("device id: %w", err)
	}

	certificate := DeviceCertificate{
		Version:         DeviceCertificateVersion,
		AccountPubkey:   cloneBytes(account.Pub),
		DevicePubkey:    cloneBytes(device.Pub),
		AgreementPubkey: agreementPrivate.PublicKey().Bytes(),
		DeviceID:        deviceID,
		Epoch:           0,
		IssuedAtUnix:    time.Now().UTC().Unix(),
	}
	payload, err := certificate.signingBytes()
	if err != nil {
		return nil, err
	}
	certificate.Signature = ed25519.Sign(account.Priv, payload)

	node := &Node{
		ID:         base64.RawURLEncoding.EncodeToString(account.Pub),
		AccountKey: account,
		DeviceKey:  device,
		AgreementKey: &AgreementKeyPair{
			Pub:  agreementPrivate.PublicKey().Bytes(),
			Priv: agreementPrivate.Bytes(),
		},
		DeviceID:    deviceID,
		Certificate: certificate,
	}
	if err := node.Validate(); err != nil {
		return nil, err
	}
	return node, nil
}

func (n *Node) Validate() error {
	if n == nil || n.AccountKey == nil || n.DeviceKey == nil || n.AgreementKey == nil {
		return ErrInvalidPrivateData
	}
	if len(n.AccountKey.Priv) != ed25519.PrivateKeySize ||
		len(n.DeviceKey.Priv) != ed25519.PrivateKeySize ||
		len(n.AgreementKey.Priv) != 32 {
		return ErrInvalidPrivateData
	}
	accountPub := n.AccountKey.Priv.Public().(ed25519.PublicKey)
	devicePub := n.DeviceKey.Priv.Public().(ed25519.PublicKey)
	agreementPrivate, err := ecdh.X25519().NewPrivateKey(n.AgreementKey.Priv)
	if err != nil {
		return ErrInvalidPrivateData
	}
	if !equalBytes(accountPub, n.AccountKey.Pub) ||
		!equalBytes(devicePub, n.DeviceKey.Pub) ||
		!equalBytes(agreementPrivate.PublicKey().Bytes(), n.AgreementKey.Pub) ||
		!equalBytes(n.Certificate.AccountPubkey, accountPub) ||
		!equalBytes(n.Certificate.DevicePubkey, devicePub) ||
		!equalBytes(n.Certificate.AgreementPubkey, n.AgreementKey.Pub) ||
		n.Certificate.DeviceID != n.DeviceID ||
		n.ID != base64.RawURLEncoding.EncodeToString(accountPub) {
		return ErrInvalidPrivateData
	}
	return n.Certificate.Verify()
}

func (n *Node) Epoch() uint64 {
	return n.Certificate.Epoch
}

// Sign signs protocol traffic with the current device key.
func (n *Node) Sign(message []byte) []byte {
	return ed25519.Sign(n.DeviceKey.Priv, message)
}

func (n *Node) Verify(message, signature []byte) bool {
	return ed25519.Verify(n.DeviceKey.Pub, message, signature)
}

func (n *Node) SignAccount(message []byte) []byte {
	return ed25519.Sign(n.AccountKey.Priv, message)
}

type privateBundle struct {
	Version          uint16            `json:"version"`
	AccountPrivate   []byte            `json:"account_private"`
	DevicePrivate    []byte            `json:"device_private"`
	AgreementPrivate []byte            `json:"agreement_private"`
	DeviceID         string            `json:"device_id"`
	Certificate      DeviceCertificate `json:"certificate"`
}

func (n *Node) MarshalPrivate() ([]byte, error) {
	if err := n.Validate(); err != nil {
		return nil, err
	}
	bundle := privateBundle{
		Version:          privateBundleVersion,
		AccountPrivate:   cloneBytes(n.AccountKey.Priv),
		DevicePrivate:    cloneBytes(n.DeviceKey.Priv),
		AgreementPrivate: cloneBytes(n.AgreementKey.Priv),
		DeviceID:         n.DeviceID,
		Certificate:      n.Certificate,
	}
	return json.Marshal(bundle)
}

func ParsePrivate(data []byte) (*Node, error) {
	var bundle privateBundle
	if err := json.Unmarshal(data, &bundle); err != nil {
		return nil, fmt.Errorf("decode identity bundle: %w", err)
	}
	if bundle.Version != privateBundleVersion ||
		len(bundle.AccountPrivate) != ed25519.PrivateKeySize ||
		len(bundle.DevicePrivate) != ed25519.PrivateKeySize ||
		len(bundle.AgreementPrivate) != 32 {
		return nil, ErrInvalidPrivateData
	}
	accountPrivate := ed25519.PrivateKey(cloneBytes(bundle.AccountPrivate))
	devicePrivate := ed25519.PrivateKey(cloneBytes(bundle.DevicePrivate))
	agreementPrivate, err := ecdh.X25519().NewPrivateKey(bundle.AgreementPrivate)
	if err != nil {
		return nil, ErrInvalidPrivateData
	}
	accountPublic := accountPrivate.Public().(ed25519.PublicKey)
	devicePublic := devicePrivate.Public().(ed25519.PublicKey)
	node := &Node{
		ID: base64.RawURLEncoding.EncodeToString(accountPublic),
		AccountKey: &KeyPair{
			Pub:  cloneBytes(accountPublic),
			Priv: accountPrivate,
		},
		DeviceKey: &KeyPair{
			Pub:  cloneBytes(devicePublic),
			Priv: devicePrivate,
		},
		AgreementKey: &AgreementKeyPair{
			Pub:  agreementPrivate.PublicKey().Bytes(),
			Priv: agreementPrivate.Bytes(),
		},
		DeviceID:    bundle.DeviceID,
		Certificate: bundle.Certificate,
	}
	if err := node.Validate(); err != nil {
		return nil, err
	}
	return node, nil
}

func generateSigningKey() (*KeyPair, error) {
	public, private, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, err
	}
	return &KeyPair{Pub: public, Priv: private}, nil
}

func newDeviceID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16]), nil
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
