#!/usr/bin/env bash
#
# Xray Users DB Initialization
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
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    username  TEXT UNIQUE NOT NULL,
    uuid      TEXT NOT NULL,
    exp       TEXT NOT NULL,
    quota     INTEGER DEFAULT 0,
    iplimit   INTEGER DEFAULT 0,
    status    TEXT DEFAULT 'active',
    created   TEXT DEFAULT CURRENT_TIMESTAMP,
    expired   TEXT
);
CREATE INDEX IF NOT EXISTS idx_vmess_user   ON vmess(username);
CREATE INDEX IF NOT EXISTS idx_vmess_status ON vmess(status);
CREATE INDEX IF NOT EXISTS idx_vmess_exp    ON vmess(exp);

-- ============================================================
--  VLESS
-- ============================================================
CREATE TABLE IF NOT EXISTS vless (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    username  TEXT UNIQUE NOT NULL,
    uuid      TEXT NOT NULL,
    exp       TEXT NOT NULL,
    quota     INTEGER DEFAULT 0,
    iplimit   INTEGER DEFAULT 0,
    status    TEXT DEFAULT 'active',
    created   TEXT DEFAULT CURRENT_TIMESTAMP,
    expired   TEXT
);
CREATE INDEX IF NOT EXISTS idx_vless_user   ON vless(username);
CREATE INDEX IF NOT EXISTS idx_vless_status ON vless(status);
CREATE INDEX IF NOT EXISTS idx_vless_exp    ON vless(exp);

-- ============================================================
--  Trojan
-- ============================================================
CREATE TABLE IF NOT EXISTS trojan (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    username  TEXT UNIQUE NOT NULL,
    password  TEXT NOT NULL,
    exp       TEXT NOT NULL,
    quota     INTEGER DEFAULT 0,
    iplimit   INTEGER DEFAULT 0,
    status    TEXT DEFAULT 'active',
    created   TEXT DEFAULT CURRENT_TIMESTAMP,
    expired   TEXT
);
CREATE INDEX IF NOT EXISTS idx_trojan_user   ON trojan(username);
CREATE INDEX IF NOT EXISTS idx_trojan_status ON trojan(status);
CREATE INDEX IF NOT EXISTS idx_trojan_exp    ON trojan(exp);

-- ============================================================
--  Shadowsocks
-- ============================================================
CREATE TABLE IF NOT EXISTS shadowsocks (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    username  TEXT UNIQUE NOT NULL,
    password  TEXT NOT NULL,
    exp       TEXT NOT NULL,
    quota     INTEGER DEFAULT 0,
    iplimit   INTEGER DEFAULT 0,
    status    TEXT DEFAULT 'active',
    created   TEXT DEFAULT CURRENT_TIMESTAMP,
    expired   TEXT
);
CREATE INDEX IF NOT EXISTS idx_ss_user   ON shadowsocks(username);
CREATE INDEX IF NOT EXISTS idx_ss_status ON shadowsocks(status);
CREATE INDEX IF NOT EXISTS idx_ss_exp    ON shadowsocks(exp);

-- ============================================================
--  SSH / OpenVPN
-- ============================================================
CREATE TABLE IF NOT EXISTS ssh (
    id        INTEGER PRIMARY KEY AUTOINCREMENT,
    username  TEXT UNIQUE NOT NULL,
    password  TEXT NOT NULL,
    exp       TEXT NOT NULL,
    quota     INTEGER DEFAULT 0,
    iplimit   INTEGER DEFAULT 0,
    status    TEXT DEFAULT 'active',
    created   TEXT DEFAULT CURRENT_TIMESTAMP,
    expired   TEXT
);
CREATE INDEX IF NOT EXISTS idx_ssh_user   ON ssh(username);
CREATE INDEX IF NOT EXISTS idx_ssh_status ON ssh(status);
CREATE INDEX IF NOT EXISTS idx_ssh_exp    ON ssh(exp);
EOF

chmod 600 "$DB"
chown root:root "$DB"
