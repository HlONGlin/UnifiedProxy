#!/bin/sh

set -eu

UNIFIEDPROXY_BIN=/opt/unifiedproxy/bin/unifiedproxy
DATA_DIR=${UNIFIEDPROXY_DATA_DIR:-/opt/unifiedproxy/data}
CORE_PATH=${UNIFIEDPROXY_CORE_PATH:-/opt/unifiedproxy/core/mihomo}
LISTEN_IP=${UNIFIEDPROXY_LISTEN_IP:-0.0.0.0}
PANEL_PORT=${UNIFIEDPROXY_PANEL_PORT:-2053}

random_hex() {
    bytes=$1
    od -An -N "$bytes" -tx1 /dev/urandom | tr -d ' \n'
}

mkdir -p "$DATA_DIR" "$(dirname "$CORE_PATH")"

if [ ! -x "$CORE_PATH" ]; then
    cp /opt/unifiedproxy/bootstrap/mihomo "$CORE_PATH"
    chmod 0755 "$CORE_PATH"
    echo "Installed bundled Mihomo core at $CORE_PATH"
fi

# Explicit arguments run the UnifiedProxy CLI without changing panel settings. This
# keeps commands such as `docker compose run --rm unifiedproxy version` predictable.
if [ "$#" -gt 0 ]; then
    exec "$UNIFIEDPROXY_BIN" "$@"
fi

if [ ! -f "$DATA_DIR/state.json" ]; then
    username=${UNIFIEDPROXY_USERNAME:-admin}
    password=${UNIFIEDPROXY_PASSWORD:-}
    panel_path=${UNIFIEDPROXY_PATH:-}

    if [ -z "$password" ]; then
        password=$(random_hex 16)
    fi
    if [ -z "$panel_path" ]; then
        panel_path="/$(random_hex 8)/"
    fi

    set -- configure \
        --data-dir "$DATA_DIR" \
        --core-path "$CORE_PATH" \
        --listen-ip "$LISTEN_IP" \
        --port "$PANEL_PORT" \
        --path "$panel_path" \
        --username "$username"

    printf '%s\n' "$password" | "$UNIFIEDPROXY_BIN" "$@" --password-stdin

    echo
    echo "UnifiedProxy was initialized. Save these values now:"
    echo "  URL:      http://<server-ip>:${PANEL_PORT}${panel_path}"
    echo "  Username: $username"
    echo "  Password: $password"
    echo
else
    # Container networking requires an externally reachable listen address.
    # CorePath is also pinned to the persistent core volume so in-panel Mihomo
    # updates survive image replacement. All other saved settings are retained.
    "$UNIFIEDPROXY_BIN" configure \
        --data-dir "$DATA_DIR" \
        --core-path "$CORE_PATH" \
        --listen-ip "$LISTEN_IP" >/dev/null
fi

cd /opt/unifiedproxy
exec "$UNIFIEDPROXY_BIN" --control
