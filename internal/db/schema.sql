PRAGMA foreign_keys = ON;
PRAGMA secure_delete = FAST;

CREATE TABLE IF NOT EXISTS schema_meta (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    version INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);
INSERT OR IGNORE INTO schema_meta (id, version, updated_at)
VALUES (1, 2, CAST(strftime('%s', 'now') AS INTEGER));
UPDATE schema_meta
SET version = 2, updated_at = CAST(strftime('%s', 'now') AS INTEGER)
WHERE id = 1;

-- Private key material is encrypted with a device-local wrapping key.
CREATE TABLE IF NOT EXISTS local_identity (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    account_pubkey BLOB NOT NULL UNIQUE CHECK (length(account_pubkey) = 32),
    encrypted_bundle BLOB NOT NULL,
    nonce BLOB NOT NULL,
    key_version INTEGER NOT NULL DEFAULT 1,
    created_at INTEGER NOT NULL,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS local_contacts (
    account_pubkey BLOB PRIMARY KEY CHECK (length(account_pubkey) = 32),
    display_name TEXT NOT NULL DEFAULT '',
    encrypted_ratchet_state BLOB,
    highest_device_epoch INTEGER NOT NULL DEFAULT 0,
    status INTEGER NOT NULL DEFAULT 0,
    last_active INTEGER NOT NULL DEFAULT 0
);

CREATE TABLE IF NOT EXISTS local_messages (
    id INTEGER PRIMARY KEY,
    message_id BLOB NOT NULL UNIQUE CHECK (length(message_id) = 16),
    task_id BLOB,
    contact_pubkey BLOB NOT NULL REFERENCES local_contacts(account_pubkey),
    direction INTEGER NOT NULL CHECK (direction IN (0, 1)),
    body_ciphertext BLOB NOT NULL,
    envelope_hash BLOB NOT NULL UNIQUE,
    created_at INTEGER NOT NULL,
    sent_at INTEGER NOT NULL DEFAULT 0,
    received_at INTEGER NOT NULL DEFAULT 0,
    read_at INTEGER NOT NULL DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_local_messages_contact
ON local_messages(contact_pubkey, created_at DESC);

CREATE TABLE IF NOT EXISTS outbox_items (
    id INTEGER PRIMARY KEY,
    message_id BLOB NOT NULL UNIQUE CHECK (length(message_id) = 16),
    recipient_pubkey BLOB NOT NULL CHECK (length(recipient_pubkey) = 32),
    envelope_blob BLOB NOT NULL,
    priority INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    next_attempt_at INTEGER NOT NULL DEFAULT 0,
    attempts INTEGER NOT NULL DEFAULT 0,
    last_error TEXT NOT NULL DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_outbox_ready
ON outbox_items(next_attempt_at, priority DESC);

CREATE TABLE IF NOT EXISTS route_cache (
    account_pubkey BLOB PRIMARY KEY CHECK (length(account_pubkey) = 32),
    encrypted_multiaddrs BLOB NOT NULL,
    observed_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_route_cache_expiry ON route_cache(expires_at);

-- Relay nodes store opaque ciphertext and routing tokens, not sender/recipient identities.
CREATE TABLE IF NOT EXISTS relay_cache (
    id INTEGER PRIMARY KEY,
    task_id BLOB NOT NULL CHECK (length(task_id) = 16),
    fragment_id BLOB NOT NULL,
    ciphertext BLOB NOT NULL,
    next_hop_token BLOB NOT NULL,
    status INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    UNIQUE(task_id, fragment_id)
);
CREATE INDEX IF NOT EXISTS idx_relay_cache_expiry ON relay_cache(expires_at);

CREATE TABLE IF NOT EXISTS contribution_receipts (
    receipt_id BLOB PRIMARY KEY,
    task_id BLOB NOT NULL CHECK (length(task_id) = 16),
    owner_pubkey BLOB NOT NULL CHECK (length(owner_pubkey) = 32),
    service_type INTEGER NOT NULL,
    reward_amount INTEGER NOT NULL CHECK (reward_amount > 0),
    proof_blob BLOB NOT NULL,
    status INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_contribution_owner
ON contribution_receipts(owner_pubkey, created_at DESC);

CREATE TABLE IF NOT EXISTS credit_states (
    owner_pubkey BLOB PRIMARY KEY CHECK (length(owner_pubkey) = 32),
    version INTEGER NOT NULL DEFAULT 0,
    spendable_credit INTEGER NOT NULL DEFAULT 0 CHECK (spendable_credit >= 0),
    lifetime_contribution INTEGER NOT NULL DEFAULT 0 CHECK (lifetime_contribution >= 0),
    burned_credit INTEGER NOT NULL DEFAULT 0 CHECK (burned_credit >= 0),
    state_hash BLOB,
    witness_set_blob BLOB,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS witness_markers (
    marker_id BLOB PRIMARY KEY,
    owner_pubkey BLOB NOT NULL CHECK (length(owner_pubkey) = 32),
    account_state_version INTEGER NOT NULL,
    marker_kind INTEGER NOT NULL,
    state_hash BLOB NOT NULL,
    next_witness_set_hash BLOB,
    status INTEGER NOT NULL DEFAULT 0,
    created_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_witness_marker_expiry ON witness_markers(expires_at);

CREATE TABLE IF NOT EXISTS local_device_state (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    account_pubkey BLOB NOT NULL CHECK (length(account_pubkey) = 32),
    current_device_pubkey BLOB NOT NULL CHECK (length(current_device_pubkey) = 32),
    previous_device_pubkey BLOB,
    epoch INTEGER NOT NULL DEFAULT 0,
    state INTEGER NOT NULL DEFAULT 1,
    updated_at INTEGER NOT NULL
);

CREATE TABLE IF NOT EXISTS local_rate_quota (
    peer_key BLOB PRIMARY KEY,
    capacity INTEGER NOT NULL,
    tokens REAL NOT NULL,
    last_refill INTEGER NOT NULL,
    expires_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_local_rate_quota_expiry ON local_rate_quota(expires_at);

PRAGMA user_version = 2;
