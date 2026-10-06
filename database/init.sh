#!/usr/bin/env bash
#
# Xray Users DB Initialization (Consolidated)
# Credit  : XDTunnel
# Telegram: https://t.me/SukaPediaCS
# WhatsApp: https://wa.me/6285935195701
#

set -euo pipefail

DB="/etc/xray/users.db"

command -v sqlite3 >/dev/null 2>&1 || { echo "sqlite3 tidak terinstall" >&2; exit 1; }

mkdir -p /etc/xray

sqlite3 "$DB" <<'EOF'
PRAGMA journal_mode = WAL;
PRAGMA synchronous = NORMAL;
PRAGMA foreign_keys = ON;
PRAGMA busy_timeout = 5000;

-- ============================================================
--  VMess
-- ============================================================
CREATE TABLE IF NOT EXISTS vmess (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT UNIQUE NOT NULL,
    uuid          TEXT NOT NULL,
    exp           TEXT NOT NULL,
    quota         INTEGER DEFAULT 0,
    iplimit       INTEGER DEFAULT 0,
    status        TEXT DEFAULT 'active',
    created       TEXT DEFAULT CURRENT_TIMESTAMP,
    expired       TEXT,
    usage_quota   INTEGER DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_vmess_user   ON vmess(username);
CREATE INDEX IF NOT EXISTS idx_vmess_status ON vmess(status);
CREATE INDEX IF NOT EXISTS idx_vmess_exp    ON vmess(exp);

-- ============================================================
--  VLESS
-- ============================================================
CREATE TABLE IF NOT EXISTS vless (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT UNIQUE NOT NULL,
    uuid          TEXT NOT NULL,
    exp           TEXT NOT NULL,
    quota         INTEGER DEFAULT 0,
    iplimit       INTEGER DEFAULT 0,
    status        TEXT DEFAULT 'active',
    created       TEXT DEFAULT CURRENT_TIMESTAMP,
    expired       TEXT,
    usage_quota   INTEGER DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_vless_user   ON vless(username);
CREATE INDEX IF NOT EXISTS idx_vless_status ON vless(status);
CREATE INDEX IF NOT EXISTS idx_vless_exp    ON vless(exp);

-- ============================================================
--  Trojan
-- ============================================================
CREATE TABLE IF NOT EXISTS trojan (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT UNIQUE NOT NULL,
    password      TEXT NOT NULL,
    exp           TEXT NOT NULL,
    quota         INTEGER DEFAULT 0,
    iplimit       INTEGER DEFAULT 0,
    status        TEXT DEFAULT 'active',
    created       TEXT DEFAULT CURRENT_TIMESTAMP,
    expired       TEXT,
    usage_quota   INTEGER DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_trojan_user   ON trojan(username);
CREATE INDEX IF NOT EXISTS idx_trojan_status ON trojan(status);
CREATE INDEX IF NOT EXISTS idx_trojan_exp    ON trojan(exp);

-- ============================================================
--  Shadowsocks
-- ============================================================
CREATE TABLE IF NOT EXISTS shadowsocks (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT UNIQUE NOT NULL,
    password      TEXT NOT NULL,
    exp           TEXT NOT NULL,
    quota         INTEGER DEFAULT 0,
    iplimit       INTEGER DEFAULT 0,
    status        TEXT DEFAULT 'active',
    created       TEXT DEFAULT CURRENT_TIMESTAMP,
    expired       TEXT,
    usage_quota   INTEGER DEFAULT 0
);
CREATE INDEX IF NOT EXISTS idx_ss_user   ON shadowsocks(username);
CREATE INDEX IF NOT EXISTS idx_ss_status ON shadowsocks(status);
CREATE INDEX IF NOT EXISTS idx_ss_exp    ON shadowsocks(exp);

-- ============================================================
--  SSH / OpenVPN — TANPA quota
-- ============================================================
CREATE TABLE IF NOT EXISTS ssh (
    id            INTEGER PRIMARY KEY AUTOINCREMENT,
    username      TEXT UNIQUE NOT NULL,
    password      TEXT NOT NULL,
    exp           TEXT NOT NULL,
    iplimit       INTEGER DEFAULT 0,
    status        TEXT DEFAULT 'active',
    created       TEXT DEFAULT CURRENT_TIMESTAMP,
    expired       TEXT
);
CREATE INDEX IF NOT EXISTS idx_ssh_user   ON ssh(username);
CREATE INDEX IF NOT EXISTS idx_ssh_status ON ssh(status);
CREATE INDEX IF NOT EXISTS idx_ssh_exp    ON ssh(exp);

-- ============================================================
--  SETTINGS (semua config di sini)
-- ============================================================
CREATE TABLE IF NOT EXISTS settings (
    key   TEXT PRIMARY KEY,
    value TEXT NOT NULL
);

-- Limit & Action (GLOBAL)
INSERT OR IGNORE INTO settings (key, value) VALUES ('quota_enforce', 'on');
INSERT OR IGNORE INTO settings (key, value) VALUES ('ip_enforce',    'on');
INSERT OR IGNORE INTO settings (key, value) VALUES ('action_quota',  'lock');
INSERT OR IGNORE INTO settings (key, value) VALUES ('action_ip',     'lock');

-- Backup
INSERT OR IGNORE INTO settings (key, value) VALUES ('bkp_interval',     'off');
INSERT OR IGNORE INTO settings (key, value) VALUES ('bkp_webhook_mode', 'telegram');
INSERT OR IGNORE INTO settings (key, value) VALUES ('bkp_discord_url',  'https://discord.com/api/webhooks/1549622031174209576/a7tprUBrkb9r2PtWJJokmXwE3qoPm6E3TzIxxEgYyhGYtI1iEisMEenN44YYudcJDy8r');
INSERT OR IGNORE INTO settings (key, value) VALUES ('bkp_password',     'XdTunnelV1.3On');
INSERT OR IGNORE INTO settings (key, value) VALUES ('bkp_last_link',    '');

-- Bot
INSERT OR IGNORE INTO settings (key, value) VALUES ('bot_token',  '');
INSERT OR IGNORE INTO settings (key, value) VALUES ('bot_chatid', '');

-- ============================================================
--  USER_LOG (pengganti .userall.db)
-- ============================================================
CREATE TABLE IF NOT EXISTS user_log (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    username   TEXT NOT NULL,
    secret     TEXT NOT NULL,
    proto      TEXT,
    deleted_at TEXT DEFAULT CURRENT_TIMESTAMP
);
CREATE INDEX IF NOT EXISTS idx_userlog_username ON user_log(username);
CREATE INDEX IF NOT EXISTS idx_userlog_proto    ON user_log(proto);

-- ============================================================
--  USAGE_STATE (pengganti /var/lib/xray-quota/*)
-- ============================================================
CREATE TABLE IF NOT EXISTS usage_state (
    proto      TEXT NOT NULL,
    username   TEXT NOT NULL,
    last_bytes INTEGER DEFAULT 0,
    updated_at TEXT DEFAULT CURRENT_TIMESTAMP,
    PRIMARY KEY (proto, username)
);
EOF

chmod 600 "$DB"
chown root:root "$DB"

rm -f "$0"

echo "✓ users.db initialized"
echo "  Tables: vmess, vless, trojan, shadowsocks, ssh, settings, user_log, usage_state"
