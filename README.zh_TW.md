<p align="center">
  <img src="./media/unifiedproxy-banner.svg" alt="UnifiedProxy — Standalone / Control / Agent" width="1200">
</p>

# UnifiedProxy

[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)

UnifiedProxy 將 Mihomo 原生面板與艦隊管理整合為同一應用，支援獨立、控制端與 Agent 三種模式，保留入站、用戶端、訂閱、多控、簽章任務、Sidecar、中央憑證、WebDAV 備份與健康檢查更新。介面支援 11 種語言，採用五套配色 × 獨立明暗，預設 T0 黑白淺色。

從[部署與遷移指南](docs/unifiedproxy-migration.md)開始。歷史備份可匯入原帳號密碼、用量與訂閱別名；憑證檔案、節點重新綁定及 NaïveProxy 豁免等邊界見[相容驗收](docs/design/legacy-acceptance.md)。

[简体中文](/README.md) · [English](/README.en.md) · [Languages](#languages)

<a id="deploy"></a>

## Linux 安裝

在使用 systemd 或 OpenRC 的 Linux 上以 `root` 執行：

```bash
bash <(curl -Ls https://raw.githubusercontent.com/HlONGlin/UnifiedProxy/main/install.sh)
```

安裝程式會選擇一個未被占用的五位數面板連接埠，並產生 16 位面板路徑、14 位使用者名稱和 16 位密碼，安裝結束時只顯示一次存取網址和憑證。也可以明確指定：

```bash
bash <(curl -Ls https://raw.githubusercontent.com/HlONGlin/UnifiedProxy/main/install.sh) \
  --port 34567 \
  --path MyPanelA1b2C3d4 \
  --username AdminUserA1234 \
  --password 'ChangeThisA12345'
```

UnifiedProxy 安裝在 `/usr/local/m-ui`，資料儲存在 `/usr/local/m-ui/data`。安裝程式也會下載適用於目前架構的 Mihomo 核心；已有核心預設保留，加上 `--update-mihomo` 才會替換。

`m-ui update` 會升級 UnifiedProxy 並保留全部設定。更新前會把資料目錄快照到 `data/backups/`，保留最近 3 份。

Release 提供 `386`、`amd64`、`arm64`、`armv5`、`armv6`、`armv7` 和 `s390x` 的 `m-ui-linux-<architecture>.tar.gz`，以及 `m-ui-windows-amd64.zip`。

<details>
<summary>Docker</summary>

Docker 映像檔適用於 Linux amd64，內含 UnifiedProxy 和一個固定版本、經過校驗的 Mihomo 核心。Compose 使用 host 網路，面板裡新增的 Inbound 連接埠不需要額外的連接埠對應即可使用。

```bash
git clone https://github.com/HlONGlin/UnifiedProxy.git
cd UnifiedProxy
cp .env.example .env
docker compose up -d --build
docker compose logs m-ui
```

從本儲存庫建置。版本工作流程可發佈 `ghcr.io/hlonglin/unifiedproxy:<tag>` 與 `latest`；確認對應映像已發佈後再執行 `docker compose pull`。`MUI_PASSWORD` / `MUI_PATH` 留空時會在首次啟動生成並輸出一次。為保持相容，服務及資料卷繼續使用 `m-ui`、`m-ui-data`、`m-ui-core`，升級時請保留。

</details>

<a id="workspace"></a>

## 截圖

![艦隊控制台](./media/screenshots/unifiedproxy/fleet.png)

<details>
<summary>截圖</summary>

**入站與用戶端**

![入站與用戶端](./media/screenshots/unifiedproxy/inbounds.png)

**用戶端配額與到期**

![用戶端配額與到期](./media/screenshots/unifiedproxy/client-edit.png)

**出站與策略群組**

![出站與策略群組](./media/screenshots/unifiedproxy/outbounds.png)

**分流規則**

![分流規則](./media/screenshots/unifiedproxy/routing.png)

**Xray / 3x-ui 匯入**

![Xray / 3x-ui 匯入](./media/screenshots/unifiedproxy/xray-import.png)

**登入**

![登入](./media/screenshots/unifiedproxy/login.png)

**手機訂閱頁**

![手機訂閱頁](./media/screenshots/unifiedproxy/subscription-mobile.png)

</details>

### 五套配色

`data-theme="light|dark"` × `data-color-theme="t0|t1|t2|t3|t4"`

| Theme | Palette | Reference |
|---|---|---|
| T0 | 黑白·墨（預設） | Vercel |
| T1 | 暖紙陶土 | Claude / Anthropic |
| T2 | 夜紫螢光 | Sentry |
| T3 | 翡翠綠 | Supabase |
| T4 | 靛紫 | Stripe |

[Tokens / references](docs/design/p2-token-map.md) · [NOTICE](NOTICE)

<details>
<summary>T0–T4 · light / dark</summary>

| | light | dark |
|---|---|---|
| T0 | <img src="./media/screenshots/unifiedproxy/t0-light.png" alt="T0 light" width="240"> | <img src="./media/screenshots/unifiedproxy/t0-dark.png" alt="T0 dark" width="240"> |
| T1 | <img src="./media/screenshots/unifiedproxy/t1-light.png" alt="T1 light" width="240"> | <img src="./media/screenshots/unifiedproxy/t1-dark.png" alt="T1 dark" width="240"> |
| T2 | <img src="./media/screenshots/unifiedproxy/t2-light.png" alt="T2 light" width="240"> | <img src="./media/screenshots/unifiedproxy/t2-dark.png" alt="T2 dark" width="240"> |
| T3 | <img src="./media/screenshots/unifiedproxy/t3-light.png" alt="T3 light" width="240"> | <img src="./media/screenshots/unifiedproxy/t3-dark.png" alt="T3 dark" width="240"> |
| T4 | <img src="./media/screenshots/unifiedproxy/t4-light.png" alt="T4 light" width="240"> | <img src="./media/screenshots/unifiedproxy/t4-dark.png" alt="T4 dark" width="240"> |

</details>

<a id="capabilities"></a>

## 功能

<details>
<summary>Multi-control</summary>

* 沒有主節點的 P2P 面板網路，使用簽章身分
* 在受信任的面板之間加密同步 Inbound
* 跨面板訂閱聚合
* 用戶端裝置：[Clash Caesar](https://github.com/RomanovCaesar/clash-caesar) 匯入外掛檔案、用面板管理員帳號登入一次，即可查看網路裡所有面板的在線情況，並透過任一個能連上的面板拉取各面板訂閱或融合訂閱

</details>

<details>
<summary>訂閱</summary>

* 瀏覽器訂閱頁，顯示用量、配額、到期時間和節點，可一鍵匯入 Clash Caesar
* **訂閱範本**：每個訂閱都是完整的用戶端設定，可使用內建的中國 / 俄羅斯 / 伊朗範本，也可匯入自己的 Mihomo 範本
* 為 Mihomo/Clash、Stash、sing-box、Surge、Surfboard、Loon、Quantumult X、Egern 產生完整設定；為 V2Box、V2RayNG、Shadowrocket 等分享連結類用戶端提供節點清單
* 每個使用者獨立的訂閱 token，可選獨立訂閱連接埠

</details>

<details>
<summary>Inbound 與 Client</summary>

* 支援 `mixed`、`socks`、`http`、`shadowsocks`、`snell`、`vmess`、`vless`、`trojan`、`hysteria2`、`tuic`、`anytls`、`mieru`、`sudoku`、`shadowquic`、`trusttunnel` 和 `hysteria2-realm` 監聽
* TLS、Reality、ShadowTLS、RestTLS、TLSMirror、JLS、Trojan SS、XHTTP、mKCP、Mekya 和混淆設定
* 每個 Client 的憑證、流量配額、到期時間、流量重設週期、到期自動續期和線上狀態；Inbound 也可設定總量和到期
* 分享連結、QR Code、透過 Mihomo listener YAML 匯入 Inbound、批次重設流量

</details>

<details>
<summary>Mihomo 設定</summary>

* Basics：執行模式、直連 IP 版本、IPv6、TCP 並行、統計、日誌和基礎分流
* Outbounds（包括 Snell、WireGuard、OpenVPN）和策略組，支援表單、原生 YAML 和分享連結轉換
* 由上而下比對的路由規則，支援 `AND` 組合規則
* Cloudflare WARP 出站，以及依國家和電信業者選擇的 VPNGate 家用寬頻出口

</details>

<details>
<summary>面板與 Mihomo</summary>

* 儀表板顯示系統、流量、連線和 Mihomo 狀態；面板啟動時自動啟動 Mihomo
* 產生並驗證 Mihomo 原生 YAML，可預覽和下載原始設定
* 從 MetaCubeX 官方 Release 切換 Mihomo 版本，更新 GeoIP / GeoSite / MetaDB
* 完整備份與還原：設定、Inbound、Multi-control 身分（可選還原）、跨面板快取、訂閱範本和 Multi-control 用戶端裝置
* **從 3x-ui 一鍵遷移：** 匯入 `x-ui.db`（v2.9.3–v3.10.2）或 Xray 的 `config.json`，入站、帶限額/到期/流量的用戶端、出站、負載平衡和路由規則都會加到面板現有內容旁邊；Mihomo 做不到的部分寫進可下載的轉換日誌
* 介面支援英文、簡體中文、繁體中文、日文、俄文、波斯文、越南文、西班牙文、土耳其文、烏克蘭文和巴西葡萄牙文

</details>

<a id="operations"></a>

## TLS 憑證

首次互動安裝時可以選擇：Let's Encrypt 網域憑證、Let's Encrypt 短期 IPv4 憑證、既有憑證和私鑰，或略過 TLS。安裝後執行：

```bash
m-ui ssl
```

選單可以申請網域和 IP 憑證；80 連接埠無法使用時，可透過 Cloudflare DNS API 申請網域與萬用字元憑證；也可以續期、撤銷、檢視憑證，以及套用既有憑證。憑證檔案只有葉憑證時，UnifiedProxy 會自動補齊憑證鏈，因為 Mihomo Party、FlClash 等用戶端會拒絕這種憑證。

## 管理指令

```text
m-ui                  # 開啟互動選單
m-ui start | stop | restart | status | logs
m-ui settings | configure | set-port
m-ui reset-credentials | reset-path
m-ui update | update-menu | ssl
m-ui enable | disable | clear-logs | uninstall
m-ui --lang <code>    選單語言，例如：m-ui --lang zh-tw
```

管理選單支援和面板相同的 11 種語言，並會記住你的選擇。安裝程式維持英文。

<a id="docs"></a>

## 文件

* [部署與遷移](docs/unifiedproxy-migration.md)
* [模型與協定](docs/fleet-protocol.md)
* [功能與相容驗收](docs/design/legacy-acceptance.md)
* [主題 tokens 與參考對象](docs/design/p2-token-map.md)
* [P5 視覺與功能審查](docs/design/p5-acceptance.md)
* [版本發佈](https://github.com/HlONGlin/UnifiedProxy/releases)

## 開發

建置、測試與預覽僅在 orca-linux 的 Linux 容器 `/workspace` 執行。Windows 僅用於編輯檔案與 Git。

```bash
test "$PWD" = /workspace && test -f /.dockerenv
go build ./...
go vet ./...
go test ./...
```

`MUI_VERSION` / Git tag → `-X main.version=...`. [Fleet build / deployment](docs/unifiedproxy-migration.md) · [Design tokens](docs/design/p2-token-map.md).

## 致謝

* [MetaCubeX/mihomo](https://github.com/MetaCubeX/mihomo)：代理核心與設定模型
* [MHSanaei/3x-ui](https://github.com/MHSanaei/3x-ui)：介面資訊密度與操作流程
* [Sub-Store](https://github.com/sub-store-org/Sub-Store)：用戶端訂閱格式的參考實作
* m-ui (GPL-3.0), UnifiedProxy (MIT): [NOTICE](NOTICE), [MIT notice](licenses/UnifiedProxy-MIT.txt).
* [Lucide](https://lucide.dev): ISC; [icon attribution](web/licenses/LICENSE-Lucide.txt).
* 配色參考對象為 Vercel / Claude（Anthropic）/ Sentry / Supabase / Stripe；資料由 VoltAgent/awesome-design-md（MIT）整理，詳見[主題設計記錄](docs/design/p2-token-map.md)。

## 授權

UnifiedProxy 採用 [GNU General Public License version 3](LICENSE) 授權。Mihomo 分享連結轉換程式碼的 GPL-3.0 來源說明見 [`internal/mihomoconvert/NOTICE.md`](internal/mihomoconvert/NOTICE.md)。

> [!IMPORTANT]
> 本專案只應用於您擁有或獲授權管理的伺服器和網路。請遵守所在地法律、上游服務商條款，以及您所存取服務的使用政策。

<a id="languages"></a>

<details>
<summary>多语言文档 / Languages</summary>

[简体中文](/README.md) | [English](/README.en.md) | [繁體中文](/README.zh_TW.md) | [日本語](/README.ja.md) | [Русский](/README.ru.md) | [فارسی](/README.fa.md) | [Tiếng Việt](/README.vi.md) | [Español](/README.es.md) | [Türkçe](/README.tr.md) | [Українська](/README.uk.md) | [Português (Brasil)](/README.pt_BR.md)

</details>
