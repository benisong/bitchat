package identity

import (
	"bytes"
	"testing"
)

func TestPrivateIdentityRoundTrip(t *testing.T) {
	node, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	payload := []byte("signed by the active device")
	signature := node.Sign(payload)
	if !node.Verify(payload, signature) {
		t.Fatal("generated device signature did not verify")
	}

	encoded, err := node.MarshalPrivate()
	if err != nil {
		t.Fatal(err)
	}
	restored, err := ParsePrivate(encoded)
	if err != nil {
		t.Fatal(err)
	}
	if restored.ID != node.ID || restored.DeviceID != node.DeviceID || restored.Epoch() != node.Epoch() {
		t.Fatal("identity changed after private bundle round trip")
	}
	if !bytes.Equal(restored.AgreementKey.Pub, node.AgreementKey.Pub) {
		t.Fatal("agreement key changed after private bundle round trip")
	}
	if !restored.Verify(payload, signature) {
		t.Fatal("restored identity could not verify the original signature")
	}
}

func TestDeviceCertificateRejectsMutation(t *testing.T) {
	node, err := Generate()
	if err != nil {
		t.Fatal(err)
	}
	certificate := node.Certificate
	certificate.Epoch++
	if err := certificate.Verify(); err == nil {
		t.Fatal("mutated device epoch was accepted")
	}
}
