package identitystore

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/subtle"
	"database/sql"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/benisong/bitchat/farp/identity"
	"github.com/benisong/bitchat/internal/db"
)

const (
	deviceKeyFilename = "device.key"
	deviceKeySize     = 32
	keyVersion        = 1
)

var identityAADPrefix = []byte("FARP_LOCAL_IDENTITY_V1")

// LoadOrCreate keeps the account stable across daemon restarts. The wrapping key
// is device-local and should move to the OS key store in a later PUK phase.
func LoadOrCreate(database *db.DB, dataDir string) (*identity.Node, error) {
	wrappingKey, err := loadOrCreateDeviceKey(dataDir)
	if err != nil {
		return nil, err
	}

	var accountPubkey, encryptedBundle, nonce []byte
	err = database.SQL().QueryRow(`
		SELECT account_pubkey, encrypted_bundle, nonce
		FROM local_identity
		WHERE id = 1
	`).Scan(&accountPubkey, &encryptedBundle, &nonce)
	if errors.Is(err, sql.ErrNoRows) {
		return create(database, wrappingKey)
	}
	if err != nil {
		return nil, fmt.Errorf("load identity record: %w", err)
	}
	plaintext, err := openBundle(wrappingKey, accountPubkey, nonce, encryptedBundle)
	if err != nil {
		return nil, fmt.Errorf("decrypt identity: %w", err)
	}
	node, err := identity.ParsePrivate(plaintext)
	if err != nil {
		return nil, fmt.Errorf("parse identity: %w", err)
	}
	if subtle.ConstantTimeCompare(accountPubkey, node.AccountKey.Pub) != 1 {
		return nil, errors.New("identity public key does not match encrypted bundle")
	}
	return node, nil
}

func create(database *db.DB, wrappingKey []byte) (*identity.Node, error) {
	node, err := identity.Generate()
	if err != nil {
		return nil, err
	}
	plaintext, err := node.MarshalPrivate()
	if err != nil {
		return nil, err
	}
	nonce, encryptedBundle, err := sealBundle(wrappingKey, node.AccountKey.Pub, plaintext)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC().Unix()
	_, err = database.SQL().Exec(`
		INSERT INTO local_identity (
			id, account_pubkey, encrypted_bundle, nonce, key_version, created_at, updated_at
		) VALUES (1, ?, ?, ?, ?, ?, ?)
	`, node.AccountKey.Pub, encryptedBundle, nonce, keyVersion, now, now)
	if err != nil {
		return nil, fmt.Errorf("persist identity: %w", err)
	}
	return node, nil
}

func sealBundle(key, accountPubkey, plaintext []byte) ([]byte, []byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, nil, err
	}
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, nil, fmt.Errorf("identity nonce: %w", err)
	}
	ciphertext := gcm.Seal(nil, nonce, plaintext, identityAAD(accountPubkey))
	return nonce, ciphertext, nil
}

func openBundle(key, accountPubkey, nonce, ciphertext []byte) ([]byte, error) {
	gcm, err := newGCM(key)
	if err != nil {
		return nil, err
	}
	if len(nonce) != gcm.NonceSize() {
		return nil, errors.New("invalid identity nonce")
	}
	plaintext, err := gcm.Open(nil, nonce, ciphertext, identityAAD(accountPubkey))
	if err != nil {
		return nil, errors.New("identity authentication failed")
	}
	return plaintext, nil
}

func newGCM(key []byte) (cipher.AEAD, error) {
	if len(key) != deviceKeySize {
		return nil, errors.New("invalid device wrapping key")
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	return cipher.NewGCM(block)
}

func identityAAD(accountPubkey []byte) []byte {
	aad := make([]byte, 0, len(identityAADPrefix)+len(accountPubkey))
	aad = append(aad, identityAADPrefix...)
	aad = append(aad, accountPubkey...)
	return aad
}

func loadOrCreateDeviceKey(dataDir string) ([]byte, error) {
	path := filepath.Join(dataDir, deviceKeyFilename)
	key, err := os.ReadFile(path)
	if err == nil {
		if len(key) != deviceKeySize {
			return nil, errors.New("invalid device key file")
		}
		if err := os.Chmod(path, 0o600); err != nil {
			return nil, fmt.Errorf("protect device key: %w", err)
		}
		return key, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return nil, fmt.Errorf("read device key: %w", err)
	}

	key = make([]byte, deviceKeySize)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("generate device key: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if errors.Is(err, os.ErrExist) {
		return loadOrCreateDeviceKey(dataDir)
	}
	if err != nil {
		return nil, fmt.Errorf("create device key: %w", err)
	}
	written := false
	defer func() {
		file.Close()
		if !written {
			os.Remove(path)
		}
	}()
	if _, err := io.Copy(file, bytes.NewReader(key)); err != nil {
		return nil, fmt.Errorf("write device key: %w", err)
	}
	if err := file.Sync(); err != nil {
		return nil, fmt.Errorf("sync device key: %w", err)
	}
	written = true
	return key, nil
}
