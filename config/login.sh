#!/bin/bash
# ============================================================
#  IP LIMIT MONITOR DAEMON (Consolidated)
#  Enforce limit IP per user dengan action_ip (dari settings)
#  Support: VMess | VLESS | Trojan | Shadowsocks | SSH
#  Interval: 30 detik
# ============================================================

source /usr/local/lib/xray-db.sh

INTERVAL=30

# ---------- Bot config dari settings ----------
CHATID=$(xray_db_get_setting bot_chatid "")
KEY=$(xray_db_get_setting bot_token "")
URL="https://api.telegram.org/bot$KEY/sendMessage"

# ============================================================
#  GLOBAL CHECK
# ============================================================
function ip_enforce_enabled() {
    local val
    val=$(xray_db_get_setting ip_enforce "on")
    [[ "$val" == "on" ]]
}

# ============================================================
#  NOTIFIKASI (jq, aman dari newline)
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
#  XRAY IP COUNT/LIST
# ============================================================
function get_online_ip_count_xray() {
    local proto="$1" user="$2"
    local count
    count=$(xray-cli check "$proto" login "$user" 2>/dev/null \
        | grep -o '"ip_count": *[0-9]*' | awk '{print $2}' | head -1)
    [[ -z "$count" ]] && count=0
    echo "$count"
}

function get_online_ips_xray() {
    local proto="$1" user="$2"
    xray-cli check "$proto" login "$user" 2>/dev/null \
        | grep -o '"ips": *\[[^]]*\]' \
        | sed 's/"ips": *\[//;s/\]//' \
        | tr -d '"' | tr ',' '\n' \
        | sed 's/^ *//;s/ *$//' | grep -v '^$' \
        | paste -sd ',' -
}

# ============================================================
#  SSH IP COUNT/LIST
# ============================================================
function get_online_ip_count_ssh() {
    local user="$1"
    local count
    count=$(who 2>/dev/null | awk -v u="$user" '$1==u {print $NF}' | sort -u | wc -l)

    local ps_count
    ps_count=$(ps -u "$user" -o comm= 2>/dev/null | grep -cE "sshd|dropbear|openvpn")
    [[ -z "$ps_count" ]] && ps_count=0

    [[ "$ps_count" -gt "$count" ]] && count=$ps_count
    echo "$count"
}

function get_online_ips_ssh() {
    local user="$1"
    local ips=""
    ips=$(who 2>/dev/null | awk -v u="$user" '$1==u {print $NF}' | sort -u)

    if [[ -z "$ips" ]]; then
        ips=$(ss -tnp 2>/dev/null \
            | grep -E "sshd|dropbear" | grep -w "$user" \
            | awk '{print $5}' | cut -d: -f1 | sort -u)
    fi
    echo "$ips" | tr '\n' ',' | sed 's/,$//'
}

# ============================================================
#  ACTION GLOBAL (dari settings)
# ============================================================
function get_action_ip() {
    xray_db_get_setting action_ip "lock"
}

# ============================================================
#  APPLY ACTION
# ============================================================
function apply_action_xray() {
    local proto="$1" user="$2"
    local action
    action=$(get_action_ip)

    case "$action" in
        delete)
            local secret
            secret=$(xray_db_get "$proto" "$user" | cut -d'|' -f2)
            xray-cli delete "$user" >/dev/null 2>&1
            [[ -n "$secret" ]] && xray_db_log_user "$user" "$secret" "$proto"
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

function apply_action_ssh() {
    local user="$1"
    local action
    action=$(get_action_ip)

    case "$action" in
        delete)
            local pass
            pass=$(xray_db_get "ssh" "$user" | cut -d'|' -f2)
            [[ -n "$pass" ]] && xray_db_log_user "$user" "$pass" "ssh"

            userdel "$user" >/dev/null 2>&1
            sed -i "/^${user}:/d" /etc/group 2>/dev/null
            xray_db_delete "ssh" "$user"
            return 1
            ;;
        lock|*)
            passwd -l "$user" &>/dev/null
            chage -E 0 "$user" &>/dev/null
            xray_db_lock "ssh" "$user"
            return 0
            ;;
    esac
}

# ============================================================
#  ENFORCE
# ============================================================
function enforce_ip_limit_xray() {
    local proto="$1" user="$2"

    local iplimit status
    read -r iplimit status < <(sqlite3 -separator ' ' "$XRAY_DB" \
        "SELECT iplimit, status FROM $proto WHERE username='$user';" 2>/dev/null)

    [[ "$status" != "active" ]] && return
    [[ -z "$iplimit" || "$iplimit" -eq 0 ]] && return

    local online
    online=$(get_online_ip_count_xray "$proto" "$user")
    [[ "$online" -eq 0 ]] && return

    if [[ "$online" -gt "$iplimit" ]]; then
        local ips
        ips=$(get_online_ips_xray "$proto" "$user")

        apply_action_xray "$proto" "$user"
        local rc=$?
        local action_label
        [[ $rc -eq 1 ]] && action_label="Deleted" || action_label="Locked"

        notify_bot "⚠️ <b>IP LIMIT EXCEEDED</b>

<b>Protocol</b> : ${proto^^}
<b>Username</b> : <code>${user}</code>
<b>Limit</b>    : ${iplimit} IP
<b>Online</b>   : ${online} IP
<b>IPs</b>      : <code>${ips}</code>
<b>Action</b>   : ${action_label}

<b>Time</b>     : $(date '+%Y-%m-%d %H:%M:%S')"
    fi
}

function enforce_ip_limit_ssh() {
    local user="$1"

    local iplimit status
    read -r iplimit status < <(sqlite3 -separator ' ' "$XRAY_DB" \
        "SELECT iplimit, status FROM ssh WHERE username='$user';" 2>/dev/null)

    [[ "$status" != "active" ]] && return
    [[ -z "$iplimit" || "$iplimit" -eq 0 ]] && return

    local online
    online=$(get_online_ip_count_ssh "$user")
    [[ "$online" -eq 0 ]] && return

    if [[ "$online" -gt "$iplimit" ]]; then
        local ips
        ips=$(get_online_ips_ssh "$user")

        apply_action_ssh "$user"
        local rc=$?
        local action_label
        [[ $rc -eq 1 ]] && action_label="Deleted" || action_label="Locked"

        notify_bot "⚠️ <b>IP LIMIT EXCEEDED (SSH)</b>

<b>Protocol</b> : SSH
<b>Username</b> : <code>${user}</code>
<b>Limit</b>    : ${iplimit} IP
<b>Online</b>   : ${online} IP
<b>IPs</b>      : <code>${ips}</code>
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
        ip_enforce_enabled || continue

        mapfile -t users < <(sqlite3 "$XRAY_DB" \
            "SELECT username FROM $proto WHERE status='active' AND iplimit > 0;" 2>/dev/null)

        for user in "${users[@]}"; do
            [[ -z "$user" ]] && continue
            sleep 0.2
            enforce_ip_limit_xray "$proto" "$user"
        done
    done
}

function monitor_ssh() {
    while true; do
        sleep "$INTERVAL"
        ip_enforce_enabled || continue

        mapfile -t users < <(sqlite3 "$XRAY_DB" \
            "SELECT username FROM ssh WHERE status='active' AND iplimit > 0;" 2>/dev/null)

        for user in "${users[@]}"; do
            [[ -z "$user" ]] && continue
            sleep 0.2
            enforce_ip_limit_ssh "$user"
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
monitor_ssh                 &

wait