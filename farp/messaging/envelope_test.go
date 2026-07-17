package messaging

import (
	"bytes"
	"crypto/rand"
	"errors"
	"testing"
	"time"

	"github.com/benisong/bitchat/farp/identity"
)

func TestEnvelopeSignatureCoversCiphertext(t *testing.T) {
	sender := mustNode(t)
	recipient := mustNode(t)
	envelope, err := NewEnvelope(
		sender,
		recipient.AccountKey.Pub,
		randomBytes(t, 32),
		[]byte("opaque encrypted payload"),
		randomBytes(t, 12),
		TypeDirectMessage,
		time.Hour,
		3,
	)
	if err != nil {
		t.Fatal(err)
	}
	now := time.Now().UTC()
	if err := envelope.Verify(now); err != nil {
		t.Fatalf("valid envelope rejected: %v", err)
	}
	first, err := envelope.SigningBytes()
	if err != nil {
		t.Fatal(err)
	}
	second, err := envelope.SigningBytes()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("envelope signing bytes are not deterministic")
	}

	envelope.Ciphertext[0] ^= 0xff
	if err := envelope.Verify(now); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("ciphertext mutation returned %v, want invalid signature", err)
	}
}

func TestEnvelopeRejectsExpiredMessage(t *testing.T) {
	sender := mustNode(t)
	recipient := mustNode(t)
	envelope, err := NewEnvelope(
		sender,
		recipient.AccountKey.Pub,
		randomBytes(t, 32),
		[]byte("ciphertext"),
		randomBytes(t, 12),
		TypeDirectMessage,
		time.Minute,
		1,
	)
	if err != nil {
		t.Fatal(err)
	}
	if err := envelope.Verify(time.Now().UTC().Add(2 * time.Minute)); !errors.Is(err, ErrEnvelopeExpired) {
		t.Fatalf("expired envelope returned %v", err)
	}
}

func TestServiceAuthorizationRejectsInvalidAmountsAndMutation(t *testing.T) {
	requester := mustNode(t)
	provider := mustNode(t)
	now := time.Now().UTC()
	authorization := &ServiceAuthorization{
		TaskID:         randomBytes(t, TaskIDSize),
		ProviderPubkey: provider.AccountKey.Pub,
		Amount:         -1,
		Nonce:          randomBytes(t, 16),
		IssuedAtUnix:   now.Unix(),
		ExpiresAtUnix:  now.Add(time.Hour).Unix(),
	}
	if err := authorization.Sign(requester); !errors.Is(err, ErrInvalidAuthorization) {
		t.Fatalf("negative authorization returned %v", err)
	}

	authorization.Amount = 5
	if err := authorization.Sign(requester); err != nil {
		t.Fatal(err)
	}
	if err := authorization.Verify(now); err != nil {
		t.Fatalf("valid authorization rejected: %v", err)
	}
	authorization.Amount = 6
	if err := authorization.Verify(now); !errors.Is(err, ErrInvalidSignature) {
		t.Fatalf("mutated amount returned %v", err)
	}
}

func mustNode(t *testing.T) *identity.Node {
	t.Helper()
	node, err := identity.Generate()
	if err != nil {
		t.Fatal(err)
	}
	return node
}

func randomBytes(t *testing.T, size int) []byte {
	t.Helper()
	value := make([]byte, size)
	if _, err := rand.Read(value); err != nil {
		t.Fatal(err)
	}
	return value
}
