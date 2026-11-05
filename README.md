<p align="center">
  <img src="./media/unifiedproxy-banner.svg" alt="UnifiedProxy — Standalone / Control / Agent" width="1200">
</p>

# UnifiedProxy

**一个工作台，管理单机代理与整支节点舰队。**

UnifiedProxy 把 Mihomo 原生配置、用户订阅和跨节点运维放进同一应用。可以从单机面板开始，也可以由控制端统一编辑节点策略，再交给 Agent 校验、应用和回报。日常配置、历史数据迁移与运行维护共用同一套工作流。

[简体中文](/README.md) · [English](/README.en.md) · [Languages](#languages)

[开始部署](#deploy) · [历史备份迁移](#migration) · [工作台预览](#workspace) · [功能验收](docs/design/p5-acceptance.md) · [版本发布](https://github.com/HlONGlin/unifiedproxy/releases)

## 选择部署方式

| 使用场景 | 入口 | 运行方式 |
|---|---|---|
| 部署舰队主机 | `unifiedproxy --control serve` | 登录后进入舰队工作台；创建子节点卡片并从同一网页管理远端 Mihomo |
| 单机运行 | 独立模式 | 管理本机 Mihomo；保留多控网络与订阅聚合 |
| 接受控制端管理 | `unifiedproxy --agent serve` | 显式登记后接入，执行配置与任务；已运行代理在控制端离线时继续工作 |

`control` / `node` 是同一应用的部署入口。节点登记、所有权边界和应急接管见[运行与迁移指南](docs/unifiedproxy-migration.md)。

<a id="deploy"></a>

## 接入与升级

在使用 systemd 或 OpenRC 的 Linux 服务器上，以 `root` 执行：

```bash
bash <(curl -Ls https://raw.githubusercontent.com/HlONGlin/unifiedproxy/main/install.sh)
```

安装结束时会显示一次访问地址和随机凭据：未占用的五位数端口、16 位面板路径、14 位用户名和 16 位密码。程序位于 `/usr/local/unifiedproxy`，数据位于 `/usr/local/unifiedproxy/data`；管理命令和系统服务统一使用 `unifiedproxy` 名称。

<details>
<summary>手动指定端口、路径和凭据</summary>

```bash
bash <(curl -Ls https://raw.githubusercontent.com/HlONGlin/unifiedproxy/main/install.sh) \
  --port 34567 \
  --path MyPanelA1b2C3d4 \
  --username AdminUserA1234 \
  --password 'ChangeThisA12345'
```

</details>

<details>
<summary>使用 Docker Compose</summary>

```bash
git clone https://github.com/HlONGlin/unifiedproxy.git
cd unifiedproxy
cp .env.example .env
docker compose up -d --build
docker compose logs unifiedproxy
```

镜像面向 Linux amd64，内置固定版本并经过校验的 Mihomo。Compose 采用主机网络，新增入站端口无需额外映射。未填写的 `UNIFIEDPROXY_PASSWORD` / `UNIFIEDPROXY_PATH` 在首次启动时生成，并在日志中显示一次。

默认从本仓库构建；发布工作流可提供 `ghcr.io/hlonglin/unifiedproxy:<tag>` 和 `latest`，对应镜像发布后可使用 `docker compose pull`。升级时保留服务名 `unifiedproxy` 及数据卷 `unifiedproxy-data`、`unifiedproxy-core`。

</details>

已有部署运行 `unifiedproxy update`，更新前自动在 `data/backups/` 创建数据快照，保留最近 3 份。安装器保留已有 Mihomo 内核；显式传入 `--update-mihomo` 才替换。UnifiedProxy 发布包提供 `unifiedproxy-linux-<architecture>.tar.gz`（`386`、`amd64`、`arm64`、`armv5`、`armv6`、`armv7`、`s390x`）及 `unifiedproxy-windows-amd64.zip`。

<a id="workspace"></a>

## 工作台与五套配色

以舰队视图查看节点，再进入节点对应的入站、客户端、路由与运行操作。以下图片来自当前界面的中文演示数据。

![UnifiedProxy 舰队工作台](./media/screenshots/unifiedproxy/fleet.png)

<details>
<summary>日常配置：入站、客户端与出站</summary>

**入站与客户端**

![入站与客户端](./media/screenshots/unifiedproxy/inbounds.png)

**客户端配额与到期**

![客户端配额与到期](./media/screenshots/unifiedproxy/client-edit.png)

**出站与策略组**

![出站与策略组](./media/screenshots/unifiedproxy/outbounds.png)

</details>

<details>
<summary>路由与迁移：分流规则、Xray / 3x-ui 导入</summary>

**分流规则**

![分流规则](./media/screenshots/unifiedproxy/routing.png)

**Xray / 3x-ui 导入**

![Xray / 3x-ui 导入](./media/screenshots/unifiedproxy/xray-import.png)

</details>

<details>
<summary>访问入口：登录与移动端订阅</summary>

**登录页**

![登录页](./media/screenshots/unifiedproxy/login.png)

**手机订阅页**

![手机订阅页](./media/screenshots/unifiedproxy/subscription-mobile.png)

</details>

配色和明暗各自独立：`data-color-theme="t0|t1|t2|t3|t4"` × `data-theme="light|dark"`，默认 T0 黑白浅色。五套配色共用布局、线性图标和交互规则。

<details>
<summary>展开五套配色的十种明暗实拍</summary>

| 主题 | 配色参考 | 浅色实拍 | 深色实拍 |
|---|---|---|---|
| T0 · 黑白·墨（默认） | Vercel | <img src="./media/screenshots/unifiedproxy/t0-light.png" alt="T0 浅色" width="240"> | <img src="./media/screenshots/unifiedproxy/t0-dark.png" alt="T0 深色" width="240"> |
| T1 · 暖纸陶土 | Claude / Anthropic | <img src="./media/screenshots/unifiedproxy/t1-light.png" alt="T1 浅色" width="240"> | <img src="./media/screenshots/unifiedproxy/t1-dark.png" alt="T1 深色" width="240"> |
| T2 · 夜紫荧光 | Sentry | <img src="./media/screenshots/unifiedproxy/t2-light.png" alt="T2 浅色" width="240"> | <img src="./media/screenshots/unifiedproxy/t2-dark.png" alt="T2 深色" width="240"> |
| T3 · 翡翠绿 | Supabase | <img src="./media/screenshots/unifiedproxy/t3-light.png" alt="T3 浅色" width="240"> | <img src="./media/screenshots/unifiedproxy/t3-dark.png" alt="T3 深色" width="240"> |
| T4 · 靛紫 | Stripe | <img src="./media/screenshots/unifiedproxy/t4-light.png" alt="T4 浅色" width="240"> | <img src="./media/screenshots/unifiedproxy/t4-dark.png" alt="T4 深色" width="240"> |

</details>

配色参考明确标注为 Vercel、Claude / Anthropic、Sentry、Supabase 和 Stripe。变量映射、色值调整和使用边界见[主题设计记录](docs/design/p2-token-map.md)；逐页检查与对比度测量见[P5 审查](docs/design/p5-acceptance.md)。

<a id="capabilities"></a>

## 可以在这里完成什么

| 工作 | 对应能力 |
|---|---|
| 维护整支舰队 | 节点登记、签名任务、节点草稿、策略发布、回执与审计，失败恢复和应急接管 |
| 配置代理服务 | Mihomo 原生 YAML、入站与客户端、出站与策略组、分流规则、内核及地理数据更新 |
| 交付用户订阅 | 凭据、配额、到期、续期、分享链接与二维码，多种客户端配置及跨面板聚合 |
| 保存并恢复部署 | 原面板备份、旧 UnifiedProxy JSON、3x-ui / Xray 导入、WebDAV 备份、中央证书和健康检查更新 |

<details>
<summary>代理配置与运行观测</summary>

* 仪表盘查看系统资源、流量、连接和 Mihomo 状态；本机面板启动时自动启动内核。
* 表单与原生 YAML 共用配置模型，生成后校验，可预览及下载原始配置；出站同时支持分享链接转换。
* 入站支持 `mixed`、`socks`、`http`、`shadowsocks`、`snell`、`vmess`、`vless`、`trojan`、`hysteria2`、`tuic`、`anytls`、`mieru`、`sudoku`、`shadowquic`、`trusttunnel`、`hysteria2-realm`。
* 可配置 TLS、Reality、ShadowTLS、RestTLS、TLSMirror、JLS、Trojan SS、XHTTP、mKCP、Mekya 及混淆；支持 Mihomo 监听器 YAML 导入。
* 基础设置包含运行模式、直连 IP 版本、IPv6、TCP 并发、统计与日志。出站包含 Snell、WireGuard、OpenVPN，支持自上而下的路由规则与 `AND` 条件。
* 可使用 Cloudflare WARP，以及按国家和运营商选择的 VPNGate 家宽出口；从 MetaCubeX 官方发布源切换内核，更新 GeoIP / GeoSite / MetaDB。

</details>

<details>
<summary>用户、订阅与多控网络</summary>

* 客户端分别设置凭据、流量配额、到期时间、重置周期和到期自动续期，查看在线状态，批量重置流量；入站也支持总量与到期限制。
* 每位用户拥有独立订阅令牌，可设置独立订阅端口。浏览器订阅页显示用量、配额、到期与节点，支持分享链接、二维码及一键导入 Clash Caesar。
* 使用内置的中国 / 俄罗斯 / 伊朗模板，或导入自定义 Mihomo 模板，生成完整客户端配置。
* 完整配置覆盖 Mihomo / Clash、Stash、sing-box、Surge、Surfboard、Loon、Quantumult X、Egern；节点列表覆盖 V2Box、V2RayNG、Shadowrocket 等分享链接客户端。
* 独立模式保留使用签名身份的无主 P2P 多控网络：在可信面板间加密同步入站、聚合订阅。[Clash Caesar](https://github.com/RomanovCaesar/clash-caesar) 导入网络文件并以管理员身份登录后，可经任一可达面板获取各面板或合并订阅。
* 面板与管理菜单支持英语、简体中文、繁体中文、日语、俄语、波斯语、越南语、西班牙语、土耳其语、乌克兰语和巴西葡萄牙语。

</details>

<details>
<summary>舰队、备份与维护保障</summary>

* 同一程序提供独立、控制端和 Agent 模式，使用签名通道下发任务，按节点保存期望配置；应用过程包含校验、热重载和健康检查，失败时恢复。
* 提供 Sidecar 编排、中央证书、WebDAV 备份及健康检查更新；旧节点接入需要显式登记。
* 完整备份包括设置、入站、可选恢复的多控身份、跨面板缓存、订阅模板和多控客户端设备。
* 旧 JSON、原面板 ZIP 与舰队备份的入口及恢复范围见[历史兼容验收](docs/design/legacy-acceptance.md)。

</details>

<a id="migration"></a>

## 把历史数据带过来

| 数据来源 | 迁移结果 | 需要留意 |
|---|---|---|
| 最早及后续 UnifiedProxy `merit.json` | 节点、原凭据、配额、历史合计用量、出站规则和旧订阅别名；原始数据进入受保护恢复档案 | 旧 Agent 重新登记；只存证书路径时补传 PEM；NaïveProxy 按既定豁免处理 |
| 现有面板备份 | 恢复设置、入站、订阅模板及可选多控身份；识别原 ZIP 与舰队备份 | 升级前保留完整数据快照 |
| 3x-ui / Xray | 导入 `x-ui.db`（v2.9.3–v3.10.2）或 `config.json`，将入站、含配额/到期/用量的客户端、出站、负载均衡与路由追加到现有配置 | 不受 Mihomo 支持的配置写入可下载的转换日志 |

旧双向合计用量单独保存为 `legacyCombined`，用于配额及订阅总用量，不伪造上传和下载明细。带原订阅令牌的旧地址保留别名。旧版未实际执行的 `bandwidthLimit` 只归档；多候选 SOCKS5 出站改用轮询分配，候选及认证保留。

迁移步骤见[运行与迁移指南](docs/unifiedproxy-migration.md)，最早备份、回滚和真实 Mihomo 传输的验证范围见[历史兼容验收](docs/design/legacy-acceptance.md)。

<a id="operations"></a>

## 日常维护与证书

| 操作 | 命令 |
|---|---|
| 打开菜单 / 切换中文 | `unifiedproxy` / `unifiedproxy --lang zh-cn` |
| 启动、停止、重启 | `unifiedproxy start` / `unifiedproxy stop` / `unifiedproxy restart` |
| 查看状态与日志 / 清理日志 | `unifiedproxy status` / `unifiedproxy logs` / `unifiedproxy clear-logs` |
| 设置面板 / 调整端口 | `unifiedproxy settings` / `unifiedproxy configure` / `unifiedproxy set-port` |
| 重置凭据 / 面板路径 | `unifiedproxy reset-credentials` / `unifiedproxy reset-path` |
| 更新程序 / 管理菜单 | `unifiedproxy update` / `unifiedproxy update-menu` |
| 开启 / 关闭开机启动 | `unifiedproxy enable` / `unifiedproxy disable` |
| 卸载 | `unifiedproxy uninstall` |

菜单记住所选语言，安装脚本为英文。证书入口：

```bash
unifiedproxy ssl
```

首次交互安装可选择 Let's Encrypt 域名证书、短期 IPv4 证书、已有证书与私钥，或跳过 TLS。菜单提供申请、续期、吊销、查看和应用操作；80 端口不可用时，可通过 Cloudflare DNS API 申请域名及通配符证书。仅含叶子证书的文件会自动补全证书链，以适配 Mihomo Party、FlClash 等客户端。

<a id="docs"></a>

## 按任务查阅文档

| 你要完成的事 | 文档 |
|---|---|
| 部署、登记节点、迁移或应急接管 | [运行与迁移指南](docs/unifiedproxy-migration.md) |
| 对接控制端、Agent 和任务模型 | [模型映射与协议](docs/fleet-protocol.md) |
| 核对旧数据支持范围 | [历史兼容验收](docs/design/legacy-acceptance.md) |
| 了解主题变量及设计参考 | [主题设计记录](docs/design/p2-token-map.md) |
| 查看功能与视觉验证证据 | [P5 审查](docs/design/p5-acceptance.md) · [P6 收尾](docs/design/p6-completion.md) |

构建、测试与预览只在 orca-linux 容器的 `/workspace` 执行；Windows 用于编辑文件与 Git 操作。

```bash
test "$PWD" = /workspace && test -f /.dockerenv
go build ./...
go vet ./...
go test ./...
```

`UNIFIEDPROXY_VERSION` 或 Git 标签指定版本，通过 `-X main.version=...` 注入构建；舰队构建与部署步骤见[指南](docs/unifiedproxy-migration.md)。

## 项目来源与许可

| 来源 | 在 UnifiedProxy 中的用途与声明 |
|---|---|
| UnifiedProxy | Mihomo 面板基础、入站/客户端和订阅等能力；保留 GPL-3.0 归属 |
| 原 UnifiedProxy（MieruPilot / merit） | 舰队生命周期、任务队列、旧备份迁移及 Sidecar 相关能力；保留 [MIT 声明](licenses/UnifiedProxy-MIT.txt) |
| [MetaCubeX/mihomo](https://github.com/MetaCubeX/mihomo) | 代理内核与原生配置模型 |
| [MHSanaei/3x-ui](https://github.com/MHSanaei/3x-ui) · [Sub-Store](https://github.com/sub-store-org/Sub-Store) | 管理流程和客户端订阅格式的参考 |
| [Lucide](https://lucide.dev) | 线性图标，保留 [ISC 许可](web/licenses/LICENSE-Lucide.txt) |

五套配色参考由 VoltAgent/awesome-design-md（MIT）整理，参考对象已在[主题设计记录](docs/design/p2-token-map.md)逐项标注，不代表品牌关联或背书。

合并发行版采用 [GNU 通用公共许可证第 3 版（GPL-3.0）](LICENSE)。完整来源说明见 [NOTICE](NOTICE)，分享链接转换代码的归属见 [转换模块声明](internal/mihomoconvert/NOTICE.md)。

> 请在拥有或获授权管理的服务器和网络中使用，并遵守所在地法律、上游服务商条款及所访问服务的使用政策。

<a id="languages"></a>

<details>
<summary>多语言文档 / Languages</summary>

[简体中文](/README.md) | [English](/README.en.md) | [繁體中文](/README.zh_TW.md) | [日本語](/README.ja.md) | [Русский](/README.ru.md) | [فارسی](/README.fa.md) | [Tiếng Việt](/README.vi.md) | [Español](/README.es.md) | [Türkçe](/README.tr.md) | [Українська](/README.uk.md) | [Português (Brasil)](/README.pt_BR.md)

</details>
