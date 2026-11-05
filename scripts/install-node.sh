#!/usr/bin/env bash
# UnifiedProxy: install the node, enroll once and start its systemd service.
set +x
set -Eeuo pipefail
umask 077

control=""; control_key=""; node_id=""; version="latest"; token_stdin=0
install_dir="${UNIFIEDPROXY_FLEET_INSTALL_DIR:-/usr/local/unifiedproxy-node}"
work_dir=""; token=""; restart_on_failure=0
info() { printf '[UnifiedProxy] %s\n' "$*"; }
die() { printf '[UnifiedProxy] ERROR: %s\n' "$*" >&2; exit 1; }
cleanup() {
  local result=$?
  token=""
  if [[ $result -ne 0 && $restart_on_failure -eq 1 ]]; then systemctl start unifiedproxy-node >/dev/null 2>&1 || true; fi
  [[ -z "$work_dir" ]] || rm -rf -- "$work_dir"
  return "$result"
}
trap cleanup EXIT
while [[ $# -gt 0 ]]; do
  case "$1" in
    --control|--control-key|--node-id|--version)
      [[ $# -ge 2 ]] || die "$1 requires a value"
      case "$1" in
        --control) control="${2%/}" ;; --control-key) control_key="$2" ;;
        --node-id) node_id="$2" ;; --version) version="$2" ;;
      esac
      shift 2 ;;
    --token-stdin) token_stdin=1; shift ;;
    -h|--help) echo 'Usage: install-node.sh --control HTTP_OR_HTTPS_URL --control-key PUBLIC_KEY --node-id ID --token-stdin [--version TAG]'; exit 0 ;;
    *) die "Unknown option: $1" ;;
  esac
done
[[ "$(uname -s)" == Linux && $EUID -eq 0 ]] || die 'Please run this command as root on Linux.'
[[ "$install_dir" =~ ^/[A-Za-z0-9._/-]+$ && "$install_dir" != / && "$install_dir" != *'/../'* && "$install_dir" != */.. ]] || die 'Invalid installation directory.'
[[ $token_stdin -eq 1 && -n "$control" && -n "$control_key" && -n "$node_id" ]] || die 'Control URL, public key, node ID and --token-stdin are required.'
[[ "$version" == latest || "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([+-][0-9A-Za-z.-]+)?$ ]] || die 'Invalid release tag.'
[[ "$node_id" =~ ^[A-Za-z0-9_-]+$ && ${#node_id} -le 128 ]] || die 'Invalid node ID.'
token="$(head -c 4097)"
[[ -n "$token" && ${#token} -le 4096 && "$token" =~ ^[a-fA-F0-9]+$ ]] || die 'Invalid or missing enrollment token. Generate a fresh command in the host panel.'
command -v systemctl >/dev/null || die 'This automatic installer requires systemd; use manual enrollment for Docker or other init systems.'
systemctl show-environment >/dev/null 2>&1 || die 'systemd is not running on this host.'

missing=0
for dependency in curl python3 sha256sum tar gzip xz install openssl; do command -v "$dependency" >/dev/null || missing=1; done
if [[ $missing -eq 1 ]]; then
  info 'Installing dependencies...'
  if command -v apt-get >/dev/null; then
    apt-get update -y </dev/null
    DEBIAN_FRONTEND=noninteractive apt-get install -y ca-certificates curl python3 coreutils tar gzip xz-utils openssl </dev/null
  elif command -v dnf >/dev/null || command -v yum >/dev/null; then
    if command -v dnf >/dev/null; then package_manager=dnf; else package_manager=yum; fi
    "$package_manager" install -y ca-certificates curl python3 coreutils tar gzip xz openssl </dev/null
  else die 'Install curl, python3, coreutils, tar, gzip, xz and openssl, then run the command again.'; fi
fi
python3 - "$control" "$control_key" <<'PY'
import base64, sys, urllib.parse
url=urllib.parse.urlsplit(sys.argv[1])
if not url.hostname or url.username or url.password or url.query or url.fragment or url.path not in ('','/') or url.scheme not in ('http','https'):
    raise SystemExit('Invalid control URL. Use the HTTP or HTTPS origin without the panel path.')
if len(base64.b64decode(sys.argv[2],validate=True))!=32: raise SystemExit('Invalid control public key.')
PY

# Never detach or overwrite an enrollment belonging to another server card.
data_dir="$install_dir/data"
if [[ -f "$data_dir/agent.json" ]]; then
  python3 - "$data_dir/agent.json" "$node_id" "$control_key" "$control" <<'PY'
import json, sys
with open(sys.argv[1]) as f: cfg=json.load(f)
if (cfg.get('nodeId'),cfg.get('controlKey'),cfg.get('controlURL','').rstrip('/')) != tuple(sys.argv[2:]):
    raise SystemExit('This server is already enrolled elsewhere. Existing data was retained; leave/rebind explicitly before connecting another card.')
PY
  if [[ -f "$data_dir/break-glass" ]]; then die 'Emergency takeover is active. Resume fleet management explicitly before restarting.'; fi
  token=""
  systemctl enable unifiedproxy-node
  systemctl restart unifiedproxy-node
  systemctl is-active --quiet unifiedproxy-node || die 'Node startup failed; inspect journalctl -u unifiedproxy-node.'
  info 'This card is already enrolled. Service restarted; return to the host panel.'
  exit 0
fi

case "$(uname -m)" in
  x86_64|amd64) arch=amd64; core_arch=amd64-compatible ;;
  aarch64|arm64|armv8*) arch=arm64; core_arch=arm64 ;;
  armv7*) arch=armv7; core_arch=armv7 ;; armv6*) arch=armv6; core_arch=armv6 ;; armv5*) arch=armv5; core_arch=armv5 ;;
  i386|i486|i586|i686|x86) arch=386; core_arch=386 ;; s390x) arch=s390x; core_arch=s390x ;;
  *) die "Unsupported architecture: $(uname -m)" ;;
esac
work_dir="$(mktemp -d)"
fetch() { curl --fail --silent --show-error --location --retry 3 --connect-timeout 10 --max-time 180 --proto '=https' --proto-redir '=https' "$1" -o "$2"; }
if [[ "$version" == latest ]]; then
  fetch 'https://api.github.com/repos/HlONGlin/UnifiedProxy/releases/latest' "$work_dir/release.json"
  version="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1]))["tag_name"])' "$work_dir/release.json")"
  [[ "$version" =~ ^v[0-9]+\.[0-9]+\.[0-9]+([+-][0-9A-Za-z.-]+)?$ ]] || die 'Invalid release metadata.'
fi
release_base="https://github.com/HlONGlin/UnifiedProxy/releases/download/$version"
asset="node-linux-$arch.tar.gz"
info "Downloading UnifiedProxy $version ($arch)..."
fetch "$release_base/$asset" "$work_dir/$asset"
fetch "$release_base/sha256sums.txt" "$work_dir/sha256sums.txt"
python3 - "$work_dir/sha256sums.txt" "$asset" > "$work_dir/node.sha256" <<'PY'
import re,sys
for line in open(sys.argv[1]):
    fields=line.split()
    if len(fields)==2 and fields[1].lstrip('*') in (sys.argv[2],'./'+sys.argv[2]) and re.fullmatch('[a-fA-F0-9]{64}',fields[0]):
        print(fields[0]+'  '+sys.argv[2]);break
else: raise SystemExit('Release checksum is missing.')
PY
(cd "$work_dir" && sha256sum --check --strict node.sha256)
tar -xOzf "$work_dir/$asset" "node-linux-$arch" > "$work_dir/node"
tar -xOzf "$work_dir/$asset" scripts/install-fleet.sh > "$work_dir/install-fleet.sh"
chmod 755 "$work_dir/node"
[[ "$("$work_dir/node" version)" == "$version" ]] || die 'The downloaded node version does not match the host release.'

# Reuse an existing core. Fresh nodes get the same pinned core as the Docker image.
core_path="$install_dir/mihomo"
if [[ -f "$data_dir/state.json" ]]; then
  saved_core="$(python3 -c 'import json,sys; print(json.load(open(sys.argv[1])).get("settings",{}).get("corePath", ""))' "$data_dir/state.json")"
  [[ ! -x "$saved_core" ]] || core_path="$saved_core"
fi
if [[ ! -x "$core_path" ]]; then
  core_version=v1.19.33
  core_asset="mihomo-linux-$core_arch-$core_version.gz"
  core_base="https://github.com/MetaCubeX/mihomo/releases/download/$core_version"
  info "Downloading Mihomo $core_version..."
  fetch "$core_base/$core_asset" "$work_dir/$core_asset"
  # Pinned SHA256 digests from the upstream v1.19.33 release assets.
  case "$arch" in
    amd64) core_sum=386508bfb1c72c43b6a0eda73708ecdce82546581b0ebcb16366c53757c622b6 ;;
    arm64) core_sum=ba408b317080ee265c46cb923a50018c1b82cbd708551cdd46a1e3e307396187 ;;
    armv5) core_sum=5981edb813ce9cc112852ba12c0d42a5187b1b0f26cf307629f2d68719fedcb9 ;;
    armv6) core_sum=288d525db3a8bac99ee5497ca0cba3d14c8a2397c7bb5354f84d6240cf907c8d ;;
    armv7) core_sum=c8a8f9bcaab914269fbdca7569e4a9f024cc667ecf12eb43e8ccdc01b3731604 ;;
    386) core_sum=e5580e1b8484e484936b0cefb14f22e9c6e86028c64e14824858fdb5eeedbd3e ;;
    s390x) core_sum=cfc5b164b8f15779ca045ccbf9932b6c3b1813647cf793de1db3406571d11efb ;;
  esac
  printf '%s  %s\n' "$core_sum" "$core_asset" > "$work_dir/core.sha256"
  (cd "$work_dir" && sha256sum --check --strict core.sha256)
  gzip -dc "$work_dir/$core_asset" > "$work_dir/mihomo"
  chmod 755 "$work_dir/mihomo"
  "$work_dir/mihomo" -v >/dev/null
fi

if systemctl is-active --quiet unifiedproxy-node; then
  restart_on_failure=1
  systemctl stop unifiedproxy-node
fi
if [[ -f "$data_dir/state.json" ]]; then
  backup="$install_dir/backups/pre-enroll-$(date +%Y%m%d-%H%M%S)-$$.tar.gz"
  mkdir -p "$install_dir/backups"
  tar -czf "$backup" -C "$install_dir" data
  info "Existing data backed up to $backup"
fi
node_sum="$(sha256sum "$work_dir/node")"; node_sum="${node_sum%% *}"
bash "$work_dir/install-fleet.sh" node "$work_dir/node" "$node_sum"
if [[ -f "$work_dir/mihomo" ]]; then install -m 755 "$work_dir/mihomo" "$core_path"; fi
if [[ ! -f "$data_dir/state.json" ]]; then
  password="$(openssl rand -hex 24)"
  printf '%s' "$password" | "$install_dir/node" configure --data-dir "$data_dir" --listen-ip 127.0.0.1 --port 2053 --username admin --password-stdin --core-path "$core_path" >/dev/null
  password=""
else
  "$install_dir/node" configure --data-dir "$data_dir" --core-path "$core_path" >/dev/null
fi
printf '%s' "$token" | "$install_dir/node" agent join --data-dir "$data_dir" --control "$control" --control-key "$control_key" --token-stdin
token=""
systemctl restart unifiedproxy-node
systemctl is-active --quiet unifiedproxy-node || die 'Node startup failed; inspect journalctl -u unifiedproxy-node.'
restart_on_failure=0
info 'Installed, enrolled and started. Return to the host panel and wait for its heartbeat.'
