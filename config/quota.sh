#!/bin/bash
# ============================================================
#  QUOTA MONITOR DAEMON (Consolidated)
#  Akumulasi USAGE_QUOTA + enforce quota + expiry
#  Action dari settings (action_quota)
#  Support: VMess | VLESS | Trojan | Shadowsocks
#  Interval: 60 detik
# ============================================================

source /usr/local/lib/xray-db.sh

INTERVAL=60

# ---------- Bot config dari settings ----------
CHATID=$(xray_db_get_setting bot_chatid "")
KEY=$(xray_db_get_setting bot_token "")
URL="https://api.telegram.org/bot$KEY/sendMessage"

# ============================================================
#  GLOBAL CHECK
# ============================================================
function quota_enforce_enabled() {
    local val
    val=$(xray_db_get_setting quota_enforce "on")
    [[ "$val" == "on" ]]
}

# ============================================================
#  NOTIFIKASI
# ============================================================
function notify_bot() {
    local text="$1"
    [[ -z "$CHATID" || -z "$KEY" ]] && return

    local payload
    payload=$(jq -n \
        --arg chat "$CHATID" \
        --arg text "$text" \
        '{chat_id: $chat, text: $text, parse_mode: "HTML", disable_web_page_preview: true}')

    curl -s -X POST -H "Content-Type: application/json" \
        -d "$payload" "$URL" >/dev/null 2>&1
}

# ============================================================
#  KONVERSI BYTES
# ============================================================
function con() {
    local -i bytes=${1:-0}
    if [[ $bytes -lt 1024 ]]; then
        echo "${bytes}B"
    elif [[ $bytes -lt 1048576 ]]; then
        echo "$(( (bytes + 1023)/1024 ))KB"
    elif [[ $bytes -lt 1073741824 ]]; then
        echo "$(( (bytes + 1048575)/1048576 ))MB"
    else
        echo "$(( (bytes + 1073741823)/1073741824 ))GB"
    fi
}

# ============================================================
#  GET TOTAL DARI XRAY
# ============================================================
function get_current_total() {
    local proto="$1" user="$2"
    xray-cli check "$proto" quota "$user" 2>/dev/null \
        | grep -o '"total_bytes": *[0-9]*' | awk '{print $2}' | head -1
}

# ============================================================
#  UPDATE USAGE (via usage_state table)
# ============================================================
function update_usage() {
    local proto="$1" user="$2"

    local current
    current=$(get_current_total "$proto" "$user")
    [[ -z "$current" ]] && current=0

    local last
    last=$(xray_db_get_state "$proto" "$user")
    [[ -z "$last" ]] && last=0

    local delta
    if [[ "$current" -lt "$last" ]]; then
        delta=$current
    else
        delta=$((current - last))
    fi

    if [[ "$delta" -gt 0 ]]; then
        xray_db_add_usage "$proto" "$user" "$delta"
    fi

    xray_db_set_state "$proto" "$user" "$current"
}

# ============================================================
#  GET ACTION GLOBAL
# ============================================================
function get_action_quota() {
    xray_db_get_setting action_quota "lock"
}

# ============================================================
#  APPLY ACTION
# ============================================================
function apply_action() {
    local proto="$1" user="$2"
    local action
    action=$(get_action_quota)

    case "$action" in
        delete)
            local secret
            secret=$(xray_db_get "$proto" "$user" | cut -d'|' -f2)
            [[ -n "$secret" ]] && xray_db_log_user "$user" "$secret" "$proto"

            xray-cli delete "$user" >/dev/null 2>&1
            xray_db_delete "$proto" "$user"
            return 1
            ;;
        lock|*)
            xray-cli delete "$user" >/dev/null 2>&1
            xray_db_lock "$proto" "$user"
            return 0
            ;;
    esac
}

# ============================================================
#  ENFORCE QUOTA
# ============================================================
function enforce_quota() {
    local proto="$1" user="$2"

    local quota usage status
    read -r quota usage status < <(sqlite3 -separator ' ' "$XRAY_DB" \
        "SELECT quota, COALESCE(usage_quota,0), status FROM $proto WHERE username='$user';" 2>/dev/null)

    [[ "$status" != "active" ]] && return
    [[ -z "$quota" || "$quota" -eq 0 ]] && return

    if [[ "$usage" -ge "$quota" ]]; then
        local action_label
        apply_action "$proto" "$user"
        local rc=$?
        [[ $rc -eq 1 ]] && action_label="Deleted" || action_label="Locked"

        notify_bot "⚠️ <b>QUOTA EXCEEDED</b>

<b>Protocol</b> : ${proto^^}
<b>Username</b> : <code>${user}</code>
<b>Quota</b>    : $(con "$quota")
<b>Used</b>     : $(con "$usage")
<b>Action</b>   : ${action_label}

<b>Time</b>     : $(date '+%Y-%m-%d %H:%M:%S')"
    fi
}

# ============================================================
#  ENFORCE EXPIRY
# ============================================================
function enforce_expiry() {
    local proto="$1" user="$2"

    local exp status
    read -r exp status < <(sqlite3 -separator ' ' "$XRAY_DB" \
        "SELECT exp, status FROM $proto WHERE username='$user';" 2>/dev/null)

    [[ "$status" != "active" ]] && return
    [[ -z "$exp" ]] && return

    local exp_ts now_ts
    exp_ts=$(date -d "$exp" +%s 2>/dev/null)
    now_ts=$(date +%s)
    [[ -z "$exp_ts" ]] && return

    if [[ "$now_ts" -gt "$exp_ts" ]]; then
        local action_label
        apply_action "$proto" "$user"
        local rc=$?
        [[ $rc -eq 1 ]] && action_label="Deleted" || action_label="Locked"

        notify_bot "⏰ <b>ACCOUNT EXPIRED</b>

<b>Protocol</b> : ${proto^^}
<b>Username</b> : <code>${user}</code>
<b>Expired</b>  : ${exp}
<b>Action</b>   : ${action_label}

<b>Time</b>     : $(date '+%Y-%m-%d %H:%M:%S')"
    fi
}

# ============================================================
#  MONITOR
# ============================================================
function monitor_proto() {
    local proto="$1"

    while true; do
        sleep "$INTERVAL"
        quota_enforce_enabled || continue

        mapfile -t users < <(sqlite3 "$XRAY_DB" \
            "SELECT username FROM $proto WHERE status='active';" 2>/dev/null)

        for user in "${users[@]}"; do
            [[ -z "$user" ]] && continue
            sleep 0.3
            update_usage  "$proto" "$user"
            enforce_quota "$proto" "$user"
            enforce_expiry "$proto" "$user"
        done
    done
}

# ============================================================
#  MAIN
# ============================================================
monitor_proto "vmess"       &
monitor_proto "vless"       &
monitor_proto "trojan"      &
monitor_proto "shadowsocks" &

wait