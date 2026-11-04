[English](/README.md) | [简体中文](/README.zh_CN.md) | [繁體中文](/README.zh_TW.md) | [日本語](/README.ja.md) | [Русский](/README.ru.md) | [فارسی](/README.fa.md) | [Tiếng Việt](/README.vi.md) | [Español](/README.es.md) | [Türkçe](/README.tr.md) | [Українська](/README.uk.md) | [Português (Brasil)](/README.pt_BR.md)

<p align="center">
  <img src="./web/media/unifiedproxy.svg" alt="UnifiedProxy" width="96">
</p>

# UnifiedProxy

[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)

UnifiedProxy combines a Mihomo-native server panel with standalone, control and Agent modes. Manage inbounds, clients, subscriptions, signed fleet tasks, sidecars, central certificates, WebDAV backups and guarded upgrades in 11 languages. The interface uses five palettes with independent light/dark modes; T0 black and white + light is the default.

Start with the [deployment and migration guide](docs/unifiedproxy-migration.md). Supported historical backups can be imported with their credentials, usage and subscription aliases; see the [compatibility record](docs/design/legacy-acceptance.md) for certificate files, node re-enrollment and the NaïveProxy exception.

> [!IMPORTANT]
> Use this project only on servers and networks that you own or are authorized to administer. Follow local law, upstream provider terms, and the policies of the services you access.

### Five palettes

`data-theme="light|dark"` × `data-color-theme="t0|t1|t2|t3|t4"`

| Theme | Palette | Reference |
|---|---|---|
| T0 | Black and white (default) | Vercel |
| T1 | Warm paper and clay | Claude / Anthropic |
| T2 | Night violet | Sentry |
| T3 | Emerald | Supabase |
| T4 | Indigo | Stripe |

[Tokens / references](docs/design/p2-token-map.md) · [NOTICE](NOTICE)

<p align="center">
  <img src="./media/screenshots/unifiedproxy/inbounds.png" alt="Inbounds" width="900">
</p>

## Features

**Panel and Mihomo**
* Dashboard with system, traffic, connection, and Mihomo status; Mihomo starts automatically with the panel
* Native Mihomo YAML generation and validation, raw configuration preview and download
* Mihomo version switching from official MetaCubeX releases, and GeoIP / GeoSite / MetaDB updates
* Full backup and restore: settings, inbounds, Multi-control identity (opt-in), cross-panel cache, subscription templates, and Multi-control client devices
* **Migrate from 3x-ui in one click:** import `x-ui.db` (v2.9.3–v3.10.2) or an Xray `config.json`; inbounds, clients with their quotas, expiry and traffic, outbounds, balancers, and routing rules are added next to what the panel already has, and a downloadable conversion log lists anything Mihomo cannot do
* Interface in English, Simplified Chinese, Traditional Chinese, Japanese, Russian, Persian, Vietnamese, Spanish, Turkish, Ukrainian, and Brazilian Portuguese

**Inbounds and clients**
* `mixed`, `socks`, `http`, `shadowsocks`, `snell`, `vmess`, `vless`, `trojan`, `hysteria2`, `tuic`, `anytls`, `mieru`, `sudoku`, `shadowquic`, `trusttunnel`, and `hysteria2-realm` listeners
* TLS, Reality, ShadowTLS, RestTLS, TLSMirror, JLS, Trojan SS, XHTTP, mKCP, Mekya, and obfuscation settings
* Per-client credentials, traffic quotas, expiry, traffic reset schedules, auto-renewal on expiry, and online state; inbound-level limits too
* Share links, QR codes, inbound import from Mihomo listener YAML, and bulk traffic reset

**Subscriptions**
* Browser subscription page with usage, quota, expiry, nodes, and one-click import into Clash Caesar
* **Subscription templates:** every subscription is a complete client profile built from the built-in China / Russia / Iran template or your own Mihomo template
* Full profiles for Mihomo/Clash, Stash, sing-box, Surge, Surfboard, Loon, Quantumult X, and Egern; node lists for V2Box, V2RayNG, Shadowrocket, and other share-link clients
* Per-user subscription tokens and an optional dedicated subscription port

**Mihomo Settings**
* Basics: routing mode, direct IP version, IPv6, TCP concurrency, statistics, logs, and basic routing
* Outbounds (including Snell, WireGuard, and OpenVPN) and policy groups, with a form editor, native YAML, and share-link conversion
* Ordered routing rules, including `AND` rules
* Cloudflare WARP outbounds and VPNGate residential exits by country and ISP

**Multi-control**
* Peer-to-peer panel network with signed identities and no master node
* Encrypted inbound synchronization between trusted panels
* Cross-panel subscription aggregation
* Client devices: [Clash Caesar](https://github.com/RomanovCaesar/clash-caesar) imports the network file, signs in once with the panel admin account, and can then see every panel and pull any panel's or the merged subscription through whichever panel it can reach

## Screenshots

Current UnifiedProxy UI, captured with Chinese demo data. Each palette supports light and dark.

<table>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/inbounds.png" alt="Inbounds"><br><sub>Inbounds</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/client-edit.png" alt="Client quota and expiry"><br><sub>Client quota and expiry</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/outbounds.png" alt="Outbounds"><br><sub>Outbounds</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/routing.png" alt="Routing"><br><sub>Routing</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/xray-import.png" alt="Xray / 3x-ui import"><br><sub>Xray / 3x-ui import</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/fleet.png" alt="Fleet"><br><sub>Fleet</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/login.png" alt="Login"><br><sub>Login</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/subscription-mobile.png" alt="Mobile subscription"><br><sub>Mobile subscription</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/t0-light.png" alt="T0 light"><br><sub>T0 light</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/t0-dark.png" alt="T0 dark"><br><sub>T0 dark</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/t1-light.png" alt="T1 light"><br><sub>T1 light</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/t1-dark.png" alt="T1 dark"><br><sub>T1 dark</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/t2-light.png" alt="T2 light"><br><sub>T2 light</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/t2-dark.png" alt="T2 dark"><br><sub>T2 dark</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/t3-light.png" alt="T3 light"><br><sub>T3 light</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/t3-dark.png" alt="T3 dark"><br><sub>T3 dark</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/t4-light.png" alt="T4 light"><br><sub>T4 light</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/t4-dark.png" alt="T4 dark"><br><sub>T4 dark</sub></td></tr>
</table>

## Linux installation

Run as `root` on a Linux system using systemd or OpenRC:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/HlONGlin/UnifiedProxy/main/install.sh)
```

The installer picks an unused five-digit panel port and generates a 16-character panel path, a 14-character username, and a 16-character password. It prints the access URL and credentials once at the end. You can also set them explicitly:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/HlONGlin/UnifiedProxy/main/install.sh) \
  --port 34567 \
  --path MyPanelA1b2C3d4 \
  --username AdminUserA1234 \
  --password 'ChangeThisA12345'
```

UnifiedProxy is installed under `/usr/local/m-ui`, with its data in `/usr/local/m-ui/data`. The installer also downloads a Mihomo core for the current architecture; an existing core is kept unless you pass `--update-mihomo`.

`m-ui update` upgrades UnifiedProxy and keeps all settings. Before updating, it snapshots the data directory into `data/backups/` and keeps the newest 3 snapshots.

Releases contain `m-ui-linux-<architecture>.tar.gz` for `386`, `amd64`, `arm64`, `armv5`, `armv6`, `armv7`, and `s390x`, plus `m-ui-windows-amd64.zip`.

## Docker

The Docker image targets Linux amd64 and contains UnifiedProxy plus a pinned, checksum-verified Mihomo core. The Compose file uses host networking, so inbound ports added in the panel work without extra port mappings.

```bash
git clone https://github.com/HlONGlin/UnifiedProxy.git
cd UnifiedProxy
cp .env.example .env
docker compose up -d --build
docker compose logs m-ui
```

Build from this repository. Tagged releases may publish `ghcr.io/hlonglin/unifiedproxy:<tag>` and `latest`; use `docker compose pull` only after the corresponding UnifiedProxy image is available. Empty `MUI_PASSWORD` / `MUI_PATH` are generated once and printed in the log. Compatibility service names and data volumes remain `m-ui`, `m-ui-data` and `m-ui-core`; retain these volumes when upgrading.

## TLS certificates

The first interactive installation offers a Let's Encrypt domain certificate, a short-lived Let's Encrypt IPv4 certificate, an existing certificate pair, or skipping TLS. After installation, run:

```bash
m-ui ssl
```

The menu can issue domain and IP certificates, use the Cloudflare DNS API for domain and wildcard certificates when port 80 is unavailable, renew, revoke, list, and apply an existing certificate pair. UnifiedProxy completes certificate files that contain only the leaf certificate, because clients such as Mihomo Party and FlClash reject them.

## Management commands

```text
m-ui                  interactive menu
m-ui start | stop | restart | status | logs
m-ui settings | configure | set-port
m-ui reset-credentials | reset-path
m-ui update | update-menu | ssl
m-ui enable | disable | clear-logs | uninstall
m-ui --lang <code>    menu language, e.g. m-ui --lang zh-cn
```

The menu speaks the same 11 languages as the panel and remembers your choice. The installer stays in English.

## Documentation

* [Deployment and migration](docs/unifiedproxy-migration.md)
* [Model and protocol](docs/fleet-protocol.md)
* [Acceptance and compatibility](docs/design/legacy-acceptance.md)
* [Theme tokens and references](docs/design/p2-token-map.md)
* [P5 visual and functional audit](docs/design/p5-acceptance.md)
* [Releases](https://github.com/HlONGlin/UnifiedProxy/releases)

## Development

Build, test and preview only in the orca-linux Linux container, at `/workspace`. Windows is used for editing and Git only.

```bash
test "$PWD" = /workspace && test -f /.dockerenv
go build ./...
go vet ./...
go test ./...
```

`MUI_VERSION` / Git tag → `-X main.version=...`. [Fleet build / deployment](docs/unifiedproxy-migration.md) · [Design tokens](docs/design/p2-token-map.md).

## Acknowledgements

* [MetaCubeX/mihomo](https://github.com/MetaCubeX/mihomo): the proxy core and configuration model
* [MHSanaei/3x-ui](https://github.com/MHSanaei/3x-ui): interface density and workflow
* [Sub-Store](https://github.com/sub-store-org/Sub-Store): reference behavior for client subscription formats
* m-ui (GPL-3.0), UnifiedProxy (MIT): [NOTICE](NOTICE), [MIT notice](licenses/UnifiedProxy-MIT.txt).
* [Lucide](https://lucide.dev): ISC; [icon attribution](web/licenses/LICENSE-Lucide.txt).
* Vercel / Claude (Anthropic) / Sentry / Supabase / Stripe: [palette references](docs/design/p2-token-map.md), via VoltAgent/awesome-design-md (MIT).

## License

UnifiedProxy is licensed under the [GNU General Public License version 3](LICENSE). The Mihomo sharing-link conversion code keeps its GPL-3.0 attribution in [`internal/mihomoconvert/NOTICE.md`](internal/mihomoconvert/NOTICE.md).
