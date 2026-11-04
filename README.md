[简体中文](/README.md) | [English](/README.en.md) | [繁體中文](/README.zh_TW.md) | [日本語](/README.ja.md) | [Русский](/README.ru.md) | [فارسی](/README.fa.md) | [Tiếng Việt](/README.vi.md) | [Español](/README.es.md) | [Türkçe](/README.tr.md) | [Українська](/README.uk.md) | [Português (Brasil)](/README.pt_BR.md)

<p align="center">
  <img src="./web/media/unifiedproxy.svg" alt="UnifiedProxy" width="96">
</p>

# UnifiedProxy

[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)

UnifiedProxy 将 Mihomo 原生面板与舰队管理整合为同一应用，支持独立、控制端和 Agent 三种模式，保留入站、客户端、订阅、多控、签名任务、Sidecar、中央证书、WebDAV 备份及健康检查更新。界面支持 11 种语言，采用五套配色 × 独立明暗，默认 T0 黑白浅色。

从[部署与迁移指南](docs/unifiedproxy-migration.md)开始。历史备份可导入原凭据、用量和订阅别名；证书文件、节点重新绑定与 NaïveProxy 豁免等边界见[兼容验收](docs/design/legacy-acceptance.md)。

> [!IMPORTANT]
> 本项目只应用于您拥有或获授权管理的服务器和网络。请遵守所在地法律、上游服务商条款以及您所访问服务的使用政策。

### 五套配色

`data-theme="light|dark"` × `data-color-theme="t0|t1|t2|t3|t4"`

| 主题 | 配色 | 参考对象 |
|---|---|---|
| T0 | 黑白·墨（默认） | Vercel |
| T1 | 暖纸陶土 | Claude / Anthropic |
| T2 | 夜紫荧光 | Sentry |
| T3 | 翡翠绿 | Supabase |
| T4 | 靛紫 | Stripe |

[主题变量映射与参考对象](docs/design/p2-token-map.md) · [来源与许可说明](NOTICE)

<p align="center">
  <img src="./media/screenshots/unifiedproxy/inbounds.png" alt="入站与客户端" width="900">
</p>

## 功能

**面板与 Mihomo**

* 仪表盘显示系统、流量、连接和 Mihomo 状态；面板启动时自动启动 Mihomo
* 生成并校验 Mihomo 原生 YAML，可预览和下载原始配置
* 从 MetaCubeX 官方发布版本切换 Mihomo 版本，更新 GeoIP / GeoSite / MetaDB
* 完整备份与恢复：设置、入站、多控身份（可选恢复）、跨面板缓存、订阅模板和多控客户端设备
* **从 3x-ui 一键迁移：** 导入 `x-ui.db`（v2.9.3–v3.10.2）或 Xray 的 `config.json`，入站、包含配额、到期时间和用量的客户端、出站、负载均衡和路由规则都会加到面板现有内容旁边；Mihomo 做不到的部分写进可下载的转换日志
* 界面支持英语、简体中文、繁体中文、日语、俄语、波斯语、越南语、西班牙语、土耳其语、乌克兰语和巴西葡萄牙语

**入站与客户端**

* 支持 `mixed`、`socks`、`http`、`shadowsocks`、`snell`、`vmess`、`vless`、`trojan`、`hysteria2`、`tuic`、`anytls`、`mieru`、`sudoku`、`shadowquic`、`trusttunnel` 和 `hysteria2-realm` 监听
* TLS、Reality、ShadowTLS、RestTLS、TLSMirror、JLS、Trojan SS、XHTTP、mKCP、Mekya 和混淆设置
* 每个客户端的凭据、流量配额、到期时间、流量重置周期、到期自动续期和在线状态；入站也可设置总量和到期
* 分享链接、二维码、通过 Mihomo 监听器 YAML 导入入站、批量重置流量

**订阅**

* 浏览器订阅页，显示用量、配额、到期时间和节点，可一键导入 Clash Caesar
* **订阅模板**：每个订阅都是完整的客户端配置，可使用内置的中国 / 俄罗斯 / 伊朗模板，也可导入自己的 Mihomo 模板
* 为 Mihomo/Clash、Stash、sing-box、Surge、Surfboard、Loon、Quantumult X、Egern 生成完整配置；为 V2Box、V2RayNG、Shadowrocket 等分享链接类客户端提供节点列表
* 每个用户独立的订阅令牌，可选独立订阅端口

**Mihomo 设置**

* 基础设置：运行模式、直连 IP 版本、IPv6、TCP 并发、统计、日志和基础分流
* 出站（包括 Snell、WireGuard、OpenVPN）和策略组，支持表单、原生 YAML 和分享链接转换
* 自上而下匹配的路由规则，支持 `AND` 组合规则
* Cloudflare WARP 出站，以及按国家和运营商选择的 VPNGate 家宽出口

**多控网络（Multi-control）**

* 无主节点的 P2P 面板网络，使用签名身份
* 在可信面板之间加密同步入站
* 跨面板订阅聚合
* 客户端设备：[Clash Caesar](https://github.com/RomanovCaesar/clash-caesar) 导入插件文件、用面板管理员账号登录一次，即可查看网络里所有面板的在线情况，并通过任意一个能连上的面板拉取各面板订阅或融合订阅

## 截图

以下为当前 UnifiedProxy 界面的中文演示数据截图；每套配色均支持浅色与深色。

<table>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/inbounds.png" alt="入站与客户端"><br><sub>入站与客户端</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/client-edit.png" alt="客户端配额与到期"><br><sub>客户端配额与到期</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/outbounds.png" alt="出站与策略组"><br><sub>出站与策略组</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/routing.png" alt="分流规则"><br><sub>分流规则</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/xray-import.png" alt="Xray / 3x-ui 导入"><br><sub>Xray / 3x-ui 导入</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/fleet.png" alt="舰队控制台"><br><sub>舰队控制台</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/login.png" alt="登录"><br><sub>登录</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/subscription-mobile.png" alt="手机订阅页"><br><sub>手机订阅页</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/t0-light.png" alt="T0 浅色"><br><sub>T0 浅色</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/t0-dark.png" alt="T0 深色"><br><sub>T0 深色</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/t1-light.png" alt="T1 浅色"><br><sub>T1 浅色</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/t1-dark.png" alt="T1 深色"><br><sub>T1 深色</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/t2-light.png" alt="T2 浅色"><br><sub>T2 浅色</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/t2-dark.png" alt="T2 深色"><br><sub>T2 深色</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/t3-light.png" alt="T3 浅色"><br><sub>T3 浅色</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/t3-dark.png" alt="T3 深色"><br><sub>T3 深色</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/t4-light.png" alt="T4 浅色"><br><sub>T4 浅色</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/t4-dark.png" alt="T4 深色"><br><sub>T4 深色</sub></td></tr>
</table>

## Linux 安装

在使用 systemd 或 OpenRC 的 Linux 上以 `root` 执行：

```bash
bash <(curl -Ls https://raw.githubusercontent.com/HlONGlin/UnifiedProxy/main/install.sh)
```

安装器会选择一个未占用的五位数面板端口，并生成 16 位面板路径、14 位用户名和 16 位密码，安装结束时只打印一次访问地址和凭据。也可以显式指定：

```bash
bash <(curl -Ls https://raw.githubusercontent.com/HlONGlin/UnifiedProxy/main/install.sh) \
  --port 34567 \
  --path MyPanelA1b2C3d4 \
  --username AdminUserA1234 \
  --password 'ChangeThisA12345'
```

UnifiedProxy 安装在 `/usr/local/m-ui`，数据保存在 `/usr/local/m-ui/data`。安装器还会下载适配当前架构的 Mihomo 内核；已有内核默认保留，传入 `--update-mihomo` 才会替换。

`m-ui update` 升级 UnifiedProxy 并保留全部设置。更新前会把数据目录快照到 `data/backups/`，保留最近 3 份。

发布包提供 `386`、`amd64`、`arm64`、`armv5`、`armv6`、`armv7` 和 `s390x` 的 `m-ui-linux-<architecture>.tar.gz`，以及 `m-ui-windows-amd64.zip`。

## Docker

Docker 镜像面向 Linux amd64，包含 UnifiedProxy 和一个固定版本、经过校验的 Mihomo 内核。Compose 使用主机网络，面板里新增的入站端口无需额外端口映射即可使用。

```bash
git clone https://github.com/HlONGlin/UnifiedProxy.git
cd UnifiedProxy
cp .env.example .env
docker compose up -d --build
docker compose logs m-ui
```

从本仓库构建。版本发布工作流可发布 `ghcr.io/hlonglin/unifiedproxy:<tag>` 与 `latest`；确认对应 UnifiedProxy 镜像已发布后再使用 `docker compose pull`。`MUI_PASSWORD` / `MUI_PATH` 为空时首次生成，并在日志打印一次。服务名与数据卷继续使用 `m-ui`、`m-ui-data` 和 `m-ui-core`，升级时保留这些卷。

## TLS 证书

首次交互安装时可以选择：Let's Encrypt 域名证书、Let's Encrypt 短期 IPv4 证书、已有证书和私钥，或跳过 TLS。安装后运行：

```bash
m-ui ssl
```

菜单可以申请域名和 IP 证书；80 端口不可用时可通过 Cloudflare DNS API 申请域名与通配符证书；还可以续期、吊销、查看证书，以及应用已有证书。证书文件只有叶子证书时，UnifiedProxy 会自动补全证书链，因为 Mihomo Party、FlClash 等客户端会拒绝这样的证书。

## 管理命令

```text
m-ui                  # 打开交互菜单
m-ui start | stop | restart | status | logs
m-ui settings | configure | set-port
m-ui reset-credentials | reset-path
m-ui update | update-menu | ssl
m-ui enable | disable | clear-logs | uninstall
m-ui --lang <code>    # 菜单语言，例如：m-ui --lang zh-cn
```

管理菜单支持和面板相同的 11 种语言，并会记住你的选择。安装脚本保持英文。

## 文档

* [部署与迁移](docs/unifiedproxy-migration.md)
* [模型与协议](docs/fleet-protocol.md)
* [功能与兼容验收](docs/design/legacy-acceptance.md)
* [主题变量与参考对象](docs/design/p2-token-map.md)
* [P5 视觉及功能审查](docs/design/p5-acceptance.md)
* [版本发布](https://github.com/HlONGlin/UnifiedProxy/releases)

## 开发

构建、测试与预览仅在 orca-linux 的 Linux 容器 `/workspace` 内执行。Windows 仅用于文件编辑和 Git。

```bash
test "$PWD" = /workspace && test -f /.dockerenv
go build ./...
go vet ./...
go test ./...
```

使用 `MUI_VERSION` 或 Git 标签指定版本，通过 `-X main.version=...` 注入构建。详见[构建与部署](docs/unifiedproxy-migration.md)及[主题变量](docs/design/p2-token-map.md)。

## 致谢

* [MetaCubeX/mihomo](https://github.com/MetaCubeX/mihomo)：代理内核与配置模型
* [MHSanaei/3x-ui](https://github.com/MHSanaei/3x-ui)：界面信息密度与操作流程
* [Sub-Store](https://github.com/sub-store-org/Sub-Store)：客户端订阅格式的参考实现
* m-ui（GPL-3.0）、UnifiedProxy（MIT）：合并来源与许可说明见 [NOTICE](NOTICE) 和 [MIT 许可声明](licenses/UnifiedProxy-MIT.txt)。
* [Lucide](https://lucide.dev)：线性图标，采用 ISC 许可；见[图标来源与许可](web/licenses/LICENSE-Lucide.txt)。
* Vercel / Claude（Anthropic）/ Sentry / Supabase / Stripe：五套配色的[参考对象](docs/design/p2-token-map.md)，参考资料由 VoltAgent/awesome-design-md（MIT）整理。

## 许可证

UnifiedProxy 使用 [GNU 通用公共许可证第 3 版（GPL-3.0）](LICENSE) 授权。Mihomo 分享链接转换代码的 GPL-3.0 来源说明见 [`internal/mihomoconvert/NOTICE.md`](internal/mihomoconvert/NOTICE.md)。
