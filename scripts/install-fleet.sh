#!/usr/bin/env bash
set -euo pipefail
umask 077
role="${1:-}"; binary="${2:-}"; checksum="${3:-}"
[[ "$role" == control || "$role" == node ]] || { echo 'Usage: install-fleet.sh control|node BINARY SHA256' >&2; exit 1; }
[[ $EUID -eq 0 ]] || { echo 'Run as root' >&2; exit 1; }
[[ -f "$binary" && "$checksum" =~ ^[a-fA-F0-9]{64}$ ]] || { echo 'Binary and SHA256 are required' >&2; exit 1; }
for dependency in systemctl sha256sum install tar xz; do
  command -v "$dependency" >/dev/null || { echo "Missing $dependency; install systemd, coreutils, tar and xz-utils first." >&2; exit 1; }
done
printf '%s  %s\n' "$checksum" "$binary" | sha256sum --check --strict
install_dir="/usr/local/unifiedproxy-${role}"
mkdir -p "$install_dir/data"
chmod 700 "$install_dir/data"
if [[ -f "$install_dir/$role" ]]; then cp -p "$install_dir/$role" "$install_dir/$role.previous"; fi
install -m 755 "$binary" "$install_dir/$role.next"
"$install_dir/$role.next" version >/dev/null
mv "$install_dir/$role.next" "$install_dir/$role"
install -m 755 "$install_dir/$role" "$install_dir/$role.recovery"
cat > "/etc/systemd/system/unifiedproxy-${role}.service" <<EOF
[Unit]
Description=UnifiedProxy ${role}
After=network-online.target
Wants=network-online.target
[Service]
Type=simple
Environment=UNIFIEDPROXY_SERVICE_NAME=unifiedproxy-$role.service
WorkingDirectory=${install_dir}
ExecStart=${install_dir}/${role} serve --data-dir ${install_dir}/data
Restart=on-failure
RestartSec=5
UMask=0077
KillMode=control-group
[Install]
WantedBy=multi-user.target
EOF
cat > "/etc/systemd/system/unifiedproxy-$role-recovery.service" <<EOF
[Unit]
Description=Recover interrupted UnifiedProxy $role upgrades
[Service]
Type=oneshot
ExecStart=$install_dir/$role.recovery upgrade-recover --data-dir $install_dir/data
UMask=0077
EOF
cat > "/etc/systemd/system/unifiedproxy-$role-recovery.timer" <<EOF
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
echo "Installed. Configure credentials/listener with $install_dir/$role configure --data-dir $install_dir/data, then systemctl restart unifiedproxy-${role}."
echo 'Nodes require Mihomo: configure --core-path /absolute/path/to/mihomo before enrollment.'
