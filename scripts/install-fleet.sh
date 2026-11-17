#!/usr/bin/env bash
set -euo pipefail
umask 077
role="${1:-}"; binary="${2:-}"; checksum="${3:-}"
[[ "$role" == control || "$role" == node ]] || { echo 'Usage: install-fleet.sh control|node BINARY SHA256' >&2; exit 1; }
[[ $EUID -eq 0 ]] || { echo 'Run as root' >&2; exit 1; }
[[ -f "$binary" && "$checksum" =~ ^[a-fA-F0-9]{64}$ ]] || { echo 'Binary and SHA256 are required' >&2; exit 1; }

service_manager="${UNIFIEDPROXY_SERVICE_MANAGER:-}"
if [[ -n "$service_manager" ]]; then
  [[ "$service_manager" == systemd || "$service_manager" == openrc ]] || { echo 'UNIFIEDPROXY_SERVICE_MANAGER must be systemd or openrc' >&2; exit 1; }
elif command -v systemctl >/dev/null 2>&1 && systemctl show-environment >/dev/null 2>&1; then
  service_manager=systemd
elif command -v rc-service >/dev/null 2>&1 && command -v rc-update >/dev/null 2>&1 && [[ -x /sbin/openrc-run ]]; then
  service_manager=openrc
else
  echo 'A running Linux systemd or Alpine OpenRC installation is required' >&2; exit 1
fi
export UNIFIEDPROXY_SERVICE_MANAGER="$service_manager"

required=(sha256sum install tar xz socat)
if [[ "$service_manager" == systemd ]]; then required+=(systemctl); else required+=(rc-service rc-update setsid); fi
missing=0
for dependency in "${required[@]}"; do command -v "$dependency" >/dev/null 2>&1 || missing=1; done
if [[ "$service_manager" == openrc && -f /etc/alpine-release && ! -x /etc/init.d/dcron ]]; then missing=1; fi
if [[ $missing -eq 1 ]]; then
  if command -v apk >/dev/null 2>&1; then
    apk_packages=(ca-certificates coreutils tar xz socat dcron)
    if [[ "$service_manager" == openrc ]]; then apk_packages+=(util-linux-misc); fi
    apk add --no-cache "${apk_packages[@]}" </dev/null
  else
    echo "Missing required tools. Install ${required[*]} first." >&2; exit 1
  fi
fi
for dependency in "${required[@]}"; do
  command -v "$dependency" >/dev/null 2>&1 || { echo "Missing $dependency" >&2; exit 1; }
done

printf '%s  %s\n' "$checksum" "$binary" | sha256sum --check --strict
install_dir="${UNIFIEDPROXY_FLEET_INSTALL_DIR:-/usr/local/unifiedproxy-${role}}"
systemd_dir="${UNIFIEDPROXY_SYSTEMD_UNIT_DIR:-/etc/systemd/system}"
openrc_dir="${UNIFIEDPROXY_OPENRC_INIT_DIR:-/etc/init.d}"
periodic_dir="${UNIFIEDPROXY_OPENRC_PERIODIC_DIR:-/etc/periodic/1min}"
for path in "$install_dir" "$systemd_dir" "$openrc_dir" "$periodic_dir"; do
  [[ "$path" =~ ^/[A-Za-z0-9._/-]+$ && "$path" != / && "$path" != *'/../'* && "$path" != */.. ]] || { echo 'Invalid absolute installation path' >&2; exit 1; }
done
mkdir -p "$install_dir/data"
chmod 700 "$install_dir/data"
if [[ -f "$install_dir/$role" ]]; then cp -p "$install_dir/$role" "$install_dir/$role.previous"; fi
install -m 755 "$binary" "$install_dir/$role.next"
"$install_dir/$role.next" version >/dev/null
mv "$install_dir/$role.next" "$install_dir/$role"
install -m 755 "$install_dir/$role" "$install_dir/$role.recovery"

if [[ "$service_manager" == systemd ]]; then
  mkdir -p "$systemd_dir"
  cat > "$systemd_dir/unifiedproxy-${role}.service" <<EOF
[Unit]
Description=UnifiedProxy ${role}
After=network-online.target
Wants=network-online.target
[Service]
Type=simple
Environment=UNIFIEDPROXY_SERVICE_NAME=unifiedproxy-$role.service
Environment=UNIFIEDPROXY_SERVICE_MANAGER=systemd
WorkingDirectory=${install_dir}
ExecStart=${install_dir}/${role} serve --data-dir ${install_dir}/data
Restart=on-failure
RestartSec=5
UMask=0077
KillMode=control-group
[Install]
WantedBy=multi-user.target
EOF
  cat > "$systemd_dir/unifiedproxy-$role-recovery.service" <<EOF
[Unit]
Description=Recover interrupted UnifiedProxy $role upgrades
[Service]
Type=oneshot
ExecStart=$install_dir/$role.recovery upgrade-recover --data-dir $install_dir/data
UMask=0077
EOF
  cat > "$systemd_dir/unifiedproxy-$role-recovery.timer" <<EOF
[Unit]
Description=Check interrupted UnifiedProxy $role upgrades
[Timer]
OnBootSec=1min
OnUnitActiveSec=1min
Unit=unifiedproxy-$role-recovery.service
[Install]
WantedBy=timers.target
EOF
  systemctl daemon-reload
  systemctl enable "unifiedproxy-${role}"
  systemctl enable --now "unifiedproxy-$role-recovery.timer"
else
  mkdir -p "$openrc_dir" "$periodic_dir"
  cat > "$openrc_dir/unifiedproxy-${role}" <<EOF
#!/sbin/openrc-run

name="UnifiedProxy ${role}"
description="UnifiedProxy ${role} agent"
command="${install_dir}/${role}"
command_args="serve --data-dir ${install_dir}/data"
directory="${install_dir}"
command_background="yes"
pidfile="/run/unifiedproxy-${role}.pid"
output_log="/var/log/unifiedproxy-${role}.log"
error_log="/var/log/unifiedproxy-${role}.log"
export UNIFIEDPROXY_SERVICE_NAME="unifiedproxy-${role}"
export UNIFIEDPROXY_SERVICE_MANAGER="openrc"

depend() {
    need net
    after firewall
}
EOF
  cat > "$periodic_dir/unifiedproxy-${role}-upgrade-recover" <<EOF
#!/bin/sh
exec ${install_dir}/${role}.recovery upgrade-recover --data-dir ${install_dir}/data >/dev/null 2>&1
EOF
  chmod 755 "$openrc_dir/unifiedproxy-${role}" "$periodic_dir/unifiedproxy-${role}-upgrade-recover"
  rc-update add "unifiedproxy-${role}" default
  if [[ -f /etc/alpine-release ]]; then
    rc-update add dcron default >/dev/null 2>&1 || true
    rc-service dcron start >/dev/null 2>&1 || echo 'Warning: start dcron to enable automatic upgrade recovery checks.' >&2
  else
    echo 'Warning: OpenRC is supported on Alpine; install the recovery command in your periodic scheduler.' >&2
  fi
fi

if [[ "$service_manager" == systemd ]]; then
  echo "Installed. Configure credentials/listener with $install_dir/$role configure --data-dir $install_dir/data, then systemctl restart unifiedproxy-${role}."
else
  echo "Installed. Configure credentials/listener with $install_dir/$role configure --data-dir $install_dir/data, then rc-service unifiedproxy-${role} restart."
fi
echo 'Nodes require Mihomo: configure --core-path /absolute/path/to/mihomo before enrollment.'
