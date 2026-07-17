package identitystore

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/benisong/bitchat/internal/db"
)

func TestIdentityPersistsAcrossDatabaseReopen(t *testing.T) {
	dataDir := t.TempDir()
	firstDB, err := db.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := firstDB.InitSchema(); err != nil {
		t.Fatal(err)
	}
	first, err := LoadOrCreate(firstDB, dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if err := firstDB.Close(); err != nil {
		t.Fatal(err)
	}

	secondDB, err := db.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer secondDB.Close()
	if err := secondDB.InitSchema(); err != nil {
		t.Fatal(err)
	}
	second, err := LoadOrCreate(secondDB, dataDir)
	if err != nil {
		t.Fatal(err)
	}
	if first.ID != second.ID || first.DeviceID != second.DeviceID || first.Epoch() != second.Epoch() {
		t.Fatal("identity changed after database reopen")
	}
	if !bytes.Equal(first.DeviceKey.Pub, second.DeviceKey.Pub) {
		t.Fatal("device signing key changed after database reopen")
	}

	deviceKey, err := os.ReadFile(filepath.Join(dataDir, deviceKeyFilename))
	if err != nil {
		t.Fatal(err)
	}
	if len(deviceKey) != deviceKeySize {
		t.Fatalf("device key length = %d, want %d", len(deviceKey), deviceKeySize)
	}
}

func TestIdentityRejectsTamperedCiphertext(t *testing.T) {
	dataDir := t.TempDir()
	database, err := db.Open(dataDir)
	if err != nil {
		t.Fatal(err)
	}
	defer database.Close()
	if err := database.InitSchema(); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOrCreate(database, dataDir); err != nil {
		t.Fatal(err)
	}
	if _, err := database.SQL().Exec(`
		UPDATE local_identity
		SET encrypted_bundle = zeroblob(length(encrypted_bundle))
		WHERE id = 1
	`); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadOrCreate(database, dataDir); err == nil {
		t.Fatal("tampered encrypted identity was accepted")
	}
}
