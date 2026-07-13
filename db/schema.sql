-- FARP Core Daemon v0.1 Schema
-- 纯整数，不含浮点

-- 自身身份（单行）
CREATE TABLE IF NOT EXISTS identity (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    pubkey BLOB NOT NULL UNIQUE,
    created_at INTEGER NOT NULL
);

-- PUK 信息（助记词指针与本地存储指标，不存明文助记词）
CREATE TABLE IF NOT EXISTS puk_meta (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    exists_flag INTEGER NOT NULL CHECK (exists_flag IN (0,1)), -- 是否已生成 PUK
    salt BLOB, -- PBKDF2 用的盐
    cipher_params TEXT, -- JSON: {"n":204800,"algo":"pbkdf2","cipher":"aes-256-gcm"}
    updated_at INTEGER NOT NULL
);

-- 联系人
CREATE TABLE IF NOT EXISTS contacts (
    pubkey TEXT PRIMARY KEY,
    display_name TEXT,
    ratchet_root_key BLOB, -- 本地存储，依赖全盘加密
    epoch INTEGER DEFAULT 0,
    last_active INTEGER,
    status INTEGER DEFAULT 0 -- 0=SLEEP(我的视角未激活聊天),更多状态后续扩展
);

-- 消息
CREATE TABLE IF NOT EXISTS messages (
    id INTEGER PRIMARY KEY,
    contact_pubkey TEXT NOT NULL REFERENCES contacts(pubkey),
    direction INTEGER NOT NULL, -- 0=in, 1=out
    plaintext BLOB, -- 已解密后的明文，依赖全盘加密
    envelope_hash TEXT UNIQUE, -- 用于去重
    sent_at INTEGER,
    received_at INTEGER DEFAULT 0,
    read_at INTEGER DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_messages_contact ON messages(contact_pubkey, received_at DESC);

-- 出队缓冲（异步打包）
CREATE TABLE IF NOT EXISTS outbox (
    id INTEGER PRIMARY KEY,
    recipient_pubkey TEXT NOT NULL,
    payload BLOB NOT NULL, -- AEAD 加密后的消息体
    priority INTEGER DEFAULT 0,
    created_at INTEGER NOT NULL,
    batched_at INTEGER DEFAULT 0,
    attempts INTEGER DEFAULT 0,
    last_error TEXT DEFAULT ''
);
CREATE INDEX IF NOT EXISTS idx_outbox_recipient ON outbox(recipient_pubkey, batched_at);

-- 路由缓存
CREATE TABLE IF NOT EXISTS routes (
    pubkey TEXT PRIMARY KEY,
    multiaddrs TEXT NOT NULL, -- JSON 数组
    observed_at INTEGER NOT NULL,
    expires_at INTEGER NOT NULL
);

-- 积分（本地视角的他人/自己）
CREATE TABLE IF NOT EXISTS credits (
    pubkey TEXT PRIMARY KEY,
    balance INTEGER DEFAULT 0, -- int64，单位 1 Credit
    frozen INTEGER DEFAULT 0,
    contribution_ratio INTEGER DEFAULT 0, -- 存储为千分比如 1230=12.30%，显示时除以100
    last_updated INTEGER NOT NULL,
    last7_days_up_gb INTEGER DEFAULT 0 -- GB，箭化存储
);

-- 离线碎片（作为 Relay 时托管的）
CREATE TABLE IF NOT EXISTS fragments (
    id INTEGER PRIMARY KEY,
    target_pubkey TEXT NOT NULL,
    sender_pubkey TEXT NOT NULL,
    fragment_blob BLOB NOT NULL,
    expires_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL
);
CREATE INDEX IF NOT EXISTS idx_fragments_target ON fragments(target_pubkey, expires_at);

-- 荣誉殿堂
CREATE TABLE IF NOT EXISTS hall_of_fame (
    pubkey TEXT PRIMARY KEY,
    total_burned INTEGER NOT NULL, -- units
    first_burn_at INTEGER NOT NULL,
    last_burn_at INTEGER NOT NULL
);

-- 本地授权已签发未结算
CREATE TABLE IF NOT EXISTS outstanding_authorizations (
    id INTEGER PRIMARY KEY,
    relay_pubkey TEXT NOT NULL,
    amount INTEGER NOT NULL, -- 总额
    nonce INTEGER NOT NULL,
    expires_at INTEGER NOT NULL,
    created_at INTEGER NOT NULL
);

-- 积分消耗存根（自证式记账）
CREATE TABLE IF NOT EXISTS spent_receipts (
    receipt_hash TEXT PRIMARY KEY,
    amount INTEGER NOT NULL,
    relay_pubkey TEXT NOT NULL,
    consumed_at INTEGER NOT NULL
);

-- 见证人记录（本地 pending credits）
CREATE TABLE IF NOT EXISTS credit_records_pending (
    id INTEGER PRIMARY KEY,
    owner_pubkey TEXT NOT NULL,
    amount INTEGER NOT NULL,
    kind TEXT NOT NULL, -- witness | relay_query | relay_delivery | relay_opportunistic
    proof_blob BLOB, -- 签名或 ACK
    created_at INTEGER NOT NULL,
    gossiped INTEGER DEFAULT 0
);

-- DHT 难度状态
CREATE TABLE IF NOT EXISTS dht_difficulty (
    epoch_key TEXT PRIMARY KEY, -- 如 "2026-07-13"
    leading_bits_target INTEGER NOT NULL, -- [18,26]
    samples_count INTEGER DEFAULT 0,
    avg_time_ms INTEGER DEFAULT 0,
    updated_at INTEGER NOT NULL
);

-- 设备状态机（服务端视角的 SLEEP/ACTIVE）
CREATE TABLE IF NOT EXISTS device_state (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    epoch INTEGER DEFAULT 0,
    state INTEGER DEFAULT 0, -- 0=SLEEP, 1=ACTIVE
    activated_at INTEGER DEFAULT 0
);

-- 铁闸配额本地快照
CREATE TABLE IF NOT EXISTS rate_quota (
    id INTEGER PRIMARY KEY CHECK (id = 1),
    capacity INTEGER NOT NULL DEFAULT 5,
    tokens INTEGER NOT NULL DEFAULT 5,
    last_refill INTEGER NOT NULL
);
