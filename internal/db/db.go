package db

import (
	"database/sql"
	"fmt"
	"os"
	"path/filepath"

	_ "github.com/mattn/go-sqlite3"
)

// DB 封装 SQLite 操作

type DB struct {
	sql *sql.DB
}

// Open 打开或初始化数据库
func Open(dir string) (*DB, error) {
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, fmt.Errorf("mkdir db: %w", err)
	}
	path := filepath.Join(dir, "bitchat.db")
	sqlDB, err := sql.Open("sqlite3", path+"?_fk=1&_journal_mode=WAL")
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := sqlDB.Ping(); err != nil {
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return &DB{sql: sqlDB}, nil
}

// InitSchema 执行 db/schema.sql
func (d *DB) InitSchema() error {
	// 直接内嵌 schema，避免文件路径问题
	_, err := d.sql.Exec(schemaSQL)
	return err
}

// Close 关闭
func (d *DB) Close() error {
	return d.sql.Close()
}

// SQL 将 schema.sql 原文贴入本文件
// 运行时用 go:embed 替代
var schemaSQL = `
CREATE TABLE IF NOT EXISTS identity (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    pubkey BLOB NOT NULL UNIQUE,
    created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS puk_meta (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    exists_flag INTEGER NOT NULL CHECK (exists_flag IN (0,1)),
    salt BLOB,
    cipher_params TEXT,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS contacts (
    pubkey TEXT PRIMARY KEY,
    display_name TEXT,
    ratchet_root_key BLOB,
    epoch INTEGER DEFAULT 0,
    last_active INTEGER,
    status INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY,
    contact_pubkey TEXT NOT NULL REFERENCES contacts(pubkey),
    direction INTEGER NOT NULL,
    plaintext BLOB,
    envelope_hash TEXT UNIQUE,
    sent_at INTEGER,
    received_at INTEGER DEFAULT 0,
    read_at INTEGER DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_messages_contact ON messages(contact_pubkey, received_at DESC);

CREATE TABLE IF NOT EXISTS outbox (
    id INTEGER PRIMARY KEY,
    recipient_pubkey TEXT NOT NULL,
    payload BLOB NOT NULL,
    priority INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL,
    batched_at INTEGER DEFAULT 0,
    attempts INTEGER DEFAULT 0,
    last_error TEXT DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_outbox_recipient ON outbox(recipient_pubkey, batched_at);

CREATE TABLE IF NOT EXISTS routes (
    pubkey TEXT PRIMARY KEY,
    multiaddrs TEXT NOT NULL,
    observed_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS credits (
    pubkey TEXT PRIMARY KEY,
    balance INTEGER DEFAULT 0,
    frozen INTEGER DEFAULT 0,
    contribution_ratio INTEGER DEFAULT 0,
    last_updated INTEGER NOT NULL,
    last7_days_up_gb INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS fragments (
    id INTEGER PRIMARY KEY,
    target_pubkey TEXT NOT NULL,
    sender_pubkey TEXT NOT NULL,
    fragment_blob BLOB NOT NULL,
    expires_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_fragments_target ON fragments(target_pubkey, expires_at);

CREATE TABLE IF NOT EXISTS hall_of_fame (
    pubkey TEXT PRIMARY KEY,
    total_burned INTEGER NOT NULL,
    first_burn_at INTEGER NOT NULL,
    last_burn_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS outstanding_authorizations (
    id INTEGER PRIMARY KEY,
    relay_pubkey TEXT NOT NULL,
    amount TEXT NOT NULL,
    nonce INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS spent_receipts (
    receipt_hash TEXT PRIMARY KEY,
    amount INTEGER NOT NULL,
    relay_pubkey TEXT NOT NULL,
    consumed_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS credit_records_pending (
    id INTEGER PRIMARY KEY,
    owner_pubkey TEXT NOT NULL,
    amount INTEGER NOT NULL,
    kind TEXT NOT NULL DEFAULT 'relay_delivery',
    proof_blob BLOB,
    created_at INTEGER NOT NULL,
    gossiped INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS device_state (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    epoch INTEGER DEFAULT 0,
    state INTEGER DEFAULT 0,
    activated_at INTEGER DEFAULT 0
);

CREATE TABLE IF NOT EXISTS rate_quota (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    capacity INTEGER NOT NULL DEFAULT 5,
    tokens INTEGER NOT NULL DEFAULT 5,
    last_refill INTEGER NOT NULL
);
`

// Query 配置元方法
func (d *DB) SQL() *sql.DB { return d.sql }
