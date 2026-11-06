<p align="center">
  <img src="./media/unifiedproxy-banner.svg" alt="UnifiedProxy — Standalone / Control / Agent" width="1200">
</p>

# UnifiedProxy

**One workspace for a standalone proxy server or an entire node fleet.**

UnifiedProxy brings native Mihomo configuration, user subscriptions and fleet operations into one application. Start with a local panel, or edit node policies on a controller and let Agents validate, apply and report them. Configuration, historical imports and maintenance share the same workflow.

[简体中文](/README.md) · [English](/README.en.md) · [Languages](#languages)

[Deploy](#deploy) · [Migrate a backup](#migration) · [Explore the workspace](#workspace) · [Acceptance records](docs/design/p5-acceptance.md) · [Releases](https://github.com/HlONGlin/unifiedproxy/releases)

## Choose how to run

| Scenario | Entry point | Behavior |
|---|---|---|
| Deploy a fleet host | `unifiedproxy --control serve` | Sign in once, then click a server card to enter its own dashboard, inbounds, users, outbounds and routing |
| Run standalone | Standalone mode | Manage local Mihomo with Multi-control networking and subscription aggregation |
| Enroll a managed node | `unifiedproxy --agent serve` | Join explicitly, apply policies and execute tasks; running proxies continue when the controller is offline |

`control` / `node` are deployment entry points for the same application. Read the [operations and migration guide](docs/unifiedproxy-migration.md) for enrollment, ownership and emergency local control.

<a id="deploy"></a>

## Install and upgrade

On a Linux server using systemd or OpenRC, run as `root`:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/HlONGlin/unifiedproxy/main/install.sh)
```

The installer prints the access URL and generated credentials once: an unused five-digit port, a 16-character panel path, a 14-character username and a 16-character password. The program lives in `/usr/local/unifiedproxy`, with data under `/usr/local/unifiedproxy/data`. The management command and system service use the `unifiedproxy` name.

<details>
<summary>Set the port, panel path and credentials explicitly</summary>

```bash
bash <(curl -Ls https://raw.githubusercontent.com/HlONGlin/unifiedproxy/main/install.sh) \
  --port 34567 \
  --path MyPanelA1b2C3d4 \
  --username AdminUserA1234 \
  --password 'ChangeThisA12345'
```

</details>

<details>
<summary>Use Docker Compose</summary>

```bash
git clone https://github.com/HlONGlin/unifiedproxy.git
cd unifiedproxy
cp .env.example .env
docker compose up -d --build
docker compose logs unifiedproxy
```

The image targets Linux amd64 and includes a pinned, checksum-verified Mihomo core. Host networking lets newly configured inbound ports work without extra mappings. Empty `UNIFIEDPROXY_PASSWORD` / `UNIFIEDPROXY_PATH` values are generated on first start and printed once in the log.

Build from this repository by default. The release workflow can publish `ghcr.io/hlonglin/unifiedproxy:<tag>` and `latest`; use `docker compose pull` once the corresponding image is available. Retain the `unifiedproxy` service and `unifiedproxy-data` / `unifiedproxy-core` volumes when upgrading.

</details>

Existing installations use `unifiedproxy update`. It snapshots data into `data/backups/` before updating and keeps the newest three snapshots. An existing Mihomo core is retained unless `--update-mihomo` is supplied. UnifiedProxy releases provide `unifiedproxy-linux-<architecture>.tar.gz` for `386`, `amd64`, `arm64`, `armv5`, `armv6`, `armv7` and `s390x`, plus `unifiedproxy-windows-amd64.zip`.

<a id="workspace"></a>

## Workspace and palettes

Use the fleet view to find a node, then work with its inbounds, clients, routes and operations. These captures show the current interface with Chinese demo data.

Inbound and client switches wait for the node to apply configuration before reporting success. Disabling an inbound also disconnects its existing sessions. A manually stopped Mihomo core remains stopped across configuration changes and node service restarts. Update both the host and nodes to v1.6.17 for the complete behavior.

![UnifiedProxy fleet workspace](./media/screenshots/unifiedproxy/fleet.png)

<details>
<summary>Daily configuration: inbounds, clients and outbounds</summary>

**Inbounds and clients**

![Inbounds and clients](./media/screenshots/unifiedproxy/inbounds.png)

**Client quota and expiry**

![Client quota and expiry](./media/screenshots/unifiedproxy/client-edit.png)

**Outbounds and policy groups**

![Outbounds and policy groups](./media/screenshots/unifiedproxy/outbounds.png)

</details>

<details>
<summary>Routing and migration: rules and Xray / 3x-ui import</summary>

**Routing rules**

![Routing rules](./media/screenshots/unifiedproxy/routing.png)

**Xray / 3x-ui import**

![Xray / 3x-ui import](./media/screenshots/unifiedproxy/xray-import.png)

</details>

<details>
<summary>Access: sign-in and mobile subscriptions</summary>

**Sign-in**

![Sign-in](./media/screenshots/unifiedproxy/login.png)

**Mobile subscription page**

![Mobile subscription page](./media/screenshots/unifiedproxy/subscription-mobile.png)

</details>

Palette and brightness are independent: `data-color-theme="t0|t1|t2|t3|t4"` × `data-theme="light|dark"`. T0 black and white in light mode is the default. All five palettes share layout, line icons and interaction rules.

<details>
<summary>View all five palettes in light and dark</summary>

| Palette | Reference | Light capture | Dark capture |
|---|---|---|---|
| T0 · Black and white (default) | Vercel | <img src="./media/screenshots/unifiedproxy/t0-light.png" alt="T0 light" width="240"> | <img src="./media/screenshots/unifiedproxy/t0-dark.png" alt="T0 dark" width="240"> |
| T1 · Warm paper and clay | Claude / Anthropic | <img src="./media/screenshots/unifiedproxy/t1-light.png" alt="T1 light" width="240"> | <img src="./media/screenshots/unifiedproxy/t1-dark.png" alt="T1 dark" width="240"> |
| T2 · Night violet | Sentry | <img src="./media/screenshots/unifiedproxy/t2-light.png" alt="T2 light" width="240"> | <img src="./media/screenshots/unifiedproxy/t2-dark.png" alt="T2 dark" width="240"> |
| T3 · Emerald | Supabase | <img src="./media/screenshots/unifiedproxy/t3-light.png" alt="T3 light" width="240"> | <img src="./media/screenshots/unifiedproxy/t3-dark.png" alt="T3 dark" width="240"> |
| T4 · Indigo | Stripe | <img src="./media/screenshots/unifiedproxy/t4-light.png" alt="T4 light" width="240"> | <img src="./media/screenshots/unifiedproxy/t4-dark.png" alt="T4 dark" width="240"> |

</details>

Palette references are Vercel, Claude / Anthropic, Sentry, Supabase and Stripe. The [theme design record](docs/design/p2-token-map.md) documents variable mapping, color adjustments and attribution; the [P5 review](docs/design/p5-acceptance.md) records page coverage and measured contrast.

<a id="capabilities"></a>

## What you can manage

| Task | Capabilities |
|---|---|
| Operate a fleet | Enrollment, signed tasks, node drafts, policy publication, receipts, audit, recovery and emergency local control |
| Configure proxy services | Native Mihomo YAML, inbounds, clients, outbounds, policy groups, routing, core and geodata updates |
| Deliver subscriptions | Credentials, quotas, expiry, renewal, links, QR codes, client profiles and cross-panel aggregation |
| Preserve deployments | Panel backups, legacy UnifiedProxy JSON, 3x-ui / Xray imports, WebDAV, central certificates and health-checked upgrades |

<details>
<summary>Proxy configuration and observability</summary>

* Inspect system resources, traffic, connections and Mihomo status. Local panels start the core automatically.
* Forms and native YAML share one configuration model, with validation, preview and download. Outbounds also support share-link conversion.
* Inbounds support `mixed`, `socks`, `http`, `shadowsocks`, `snell`, `vmess`, `vless`, `trojan`, `hysteria2`, `tuic`, `anytls`, `mieru`, `sudoku`, `shadowquic`, `trusttunnel` and `hysteria2-realm`.
* Configure TLS, Reality, ShadowTLS, RestTLS, TLSMirror, JLS, Trojan SS, XHTTP, mKCP, Mekya and obfuscation; import Mihomo listener YAML.
* Set routing mode, direct IP version, IPv6, TCP concurrency, statistics and logging. Outbounds include Snell, WireGuard and OpenVPN, with ordered routing and `AND` conditions.
* Use Cloudflare WARP or select VPNGate residential exits by country and ISP. Switch core versions from official MetaCubeX releases and update GeoIP / GeoSite / MetaDB.

</details>

<details>
<summary>Users, subscriptions and Multi-control</summary>

* Set per-client credentials, traffic quotas, expiry, reset schedules and renewal on expiry. Inspect online state and reset traffic in bulk; inbound-level quota and expiry limits are supported too.
* Each user gets a subscription token, with an optional dedicated subscription port. The browser page shows usage, quota, expiry and nodes, with share links, QR codes and one-click Clash Caesar import.
* Build complete profiles from the built-in China / Russia / Iran templates or import a custom Mihomo template.
* Full profiles cover Mihomo / Clash, Stash, sing-box, Surge, Surfboard, Loon, Quantumult X and Egern. Node lists cover V2Box, V2RayNG, Shadowrocket and other share-link clients.
* Standalone mode retains the signed, peer-to-peer Multi-control network: synchronize inbounds between trusted panels and aggregate subscriptions. After importing the network file and signing in as an administrator, [Clash Caesar](https://github.com/RomanovCaesar/clash-caesar) can retrieve individual or merged subscriptions through any reachable panel.
* The panel and management menu support English, Simplified Chinese, Traditional Chinese, Japanese, Russian, Persian, Vietnamese, Spanish, Turkish, Ukrainian and Brazilian Portuguese.

</details>

<details>
<summary>Fleet operations, backups and recovery</summary>

* One application provides standalone, control and Agent modes, a signed task channel and per-node desired configuration. Applying a policy includes validation, reload and health checks, with recovery on failure.
* Sidecar orchestration, central certificates, WebDAV backups and health-checked upgrades are available. Existing nodes enroll explicitly.
* Full backups include settings, inbounds, opt-in Multi-control identity, cross-panel cache, subscription templates and Multi-control client devices.
* The [legacy compatibility record](docs/design/legacy-acceptance.md) describes import endpoints and recovery coverage for historical JSON, panel ZIP and fleet backups.

</details>

<a id="migration"></a>

## Bring your existing data

| Source | Migration result | Conditions |
|---|---|---|
| Earliest and later UnifiedProxy `merit.json` | Nodes, original credentials, quotas, combined historical usage, outbound rules and subscription aliases; protected archive of original data | Re-enroll Agents; provide PEM files when only certificate paths were saved; NaïveProxy follows the documented exception |
| Existing panel backups | Settings, inbounds, subscription templates and optional Multi-control identity; original ZIP and fleet backups recognized | Keep a complete data snapshot before upgrading |
| 3x-ui / Xray | Import `x-ui.db` (v2.9.3–v3.10.2) or `config.json`; append inbounds, clients with quota/expiry/usage, outbounds, balancers and routes | A downloadable conversion log records settings unsupported by Mihomo |

Historical combined traffic is stored as `legacyCombined` for quotas and subscription totals without inventing upload/download detail. Old subscription URLs with their original tokens retain aliases. The previously unenforced `bandwidthLimit` is archived; multiple SOCKS5 candidates now use round-robin selection, preserving candidates and authentication.

Follow the [operations and migration guide](docs/unifiedproxy-migration.md). The [compatibility record](docs/design/legacy-acceptance.md) defines the tested scope for the earliest backups, rollback and real Mihomo transports.

<a id="operations"></a>

## Maintenance and certificates

| Action | Commands |
|---|---|
| Open the menu / select English | `unifiedproxy` / `unifiedproxy --lang en` |
| Start, stop, restart | `unifiedproxy start` / `unifiedproxy stop` / `unifiedproxy restart` |
| Inspect status and logs / clear logs | `unifiedproxy status` / `unifiedproxy logs` / `unifiedproxy clear-logs` |
| Configure the panel / change its port | `unifiedproxy settings` / `unifiedproxy configure` / `unifiedproxy set-port` |
| Reset credentials / panel path | `unifiedproxy reset-credentials` / `unifiedproxy reset-path` |
| Update the application / management menu | `unifiedproxy update` / `unifiedproxy update-menu` |
| Enable / disable startup | `unifiedproxy enable` / `unifiedproxy disable` |
| Uninstall | `unifiedproxy uninstall` |

The menu remembers the selected language. The installer uses English. Open certificate management with:

```bash
unifiedproxy ssl
```

The first interactive installation offers a Let's Encrypt domain certificate, a short-lived IPv4 certificate, an existing certificate/key pair or skipping TLS. The menu issues, renews, revokes, lists and applies certificates. Cloudflare DNS API issuance supports domains and wildcards when port 80 is unavailable. Leaf-only files receive a complete chain for clients such as Mihomo Party and FlClash.

<a id="docs"></a>

## Find a guide by task

| Goal | Documentation |
|---|---|
| Deploy, enroll, migrate or take emergency local control | [Operations and migration](docs/unifiedproxy-migration.md) |
| Integrate controllers, Agents and tasks | [Model mapping and protocol](docs/fleet-protocol.md) |
| Check historical data coverage | [Legacy compatibility](docs/design/legacy-acceptance.md) |
| Understand tokens and design references | [Theme design record](docs/design/p2-token-map.md) |
| Inspect functional and visual evidence | [P5 review](docs/design/p5-acceptance.md) · [P6 completion](docs/design/p6-completion.md) |

Build, test and preview only in the orca-linux container at `/workspace`. Use Windows for editing and Git operations.

```bash
test "$PWD" = /workspace && test -f /.dockerenv
go build ./...
go vet ./...
go test ./...
```

Set the version with `UNIFIEDPROXY_VERSION` or a Git tag and inject it using `-X main.version=...`. Fleet build and deployment steps are in the [guide](docs/unifiedproxy-migration.md).

## Origins and licensing

| Source | Role and retained notice |
|---|---|
| UnifiedProxy | Mihomo panel foundation, inbound/client and subscription capabilities; GPL-3.0 attribution retained |
| Original UnifiedProxy (MieruPilot / merit) | Fleet lifecycle, task queue, legacy backup migration and Sidecar capabilities; [MIT notice](licenses/UnifiedProxy-MIT.txt) retained |
| [MetaCubeX/mihomo](https://github.com/MetaCubeX/mihomo) | Proxy core and native configuration model |
| [MHSanaei/3x-ui](https://github.com/MHSanaei/3x-ui) · [Sub-Store](https://github.com/sub-store-org/Sub-Store) | References for management workflows and client subscription formats |
| [Lucide](https://lucide.dev) | Line icons with the [ISC notice](web/licenses/LICENSE-Lucide.txt) retained |

Palette references were assembled by VoltAgent/awesome-design-md (MIT) and are named in the [theme design record](docs/design/p2-token-map.md). These references do not imply affiliation or endorsement.

The combined distribution uses [GNU General Public License version 3](LICENSE). Read [NOTICE](NOTICE) for source attribution and the [conversion module notice](internal/mihomoconvert/NOTICE.md) for sharing-link conversion provenance.

> Use the project on servers and networks you own or are authorized to administer, following local law, provider terms and the policies of services you access.

<a id="languages"></a>

<details>
<summary>多语言文档 / Languages</summary>

[简体中文](/README.md) | [English](/README.en.md) | [繁體中文](/README.zh_TW.md) | [日本語](/README.ja.md) | [Русский](/README.ru.md) | [فارسی](/README.fa.md) | [Tiếng Việt](/README.vi.md) | [Español](/README.es.md) | [Türkçe](/README.tr.md) | [Українська](/README.uk.md) | [Português (Brasil)](/README.pt_BR.md)

</details>
