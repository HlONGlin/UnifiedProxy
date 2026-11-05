<p align="center">
  <img src="./media/unifiedproxy-banner.svg" alt="UnifiedProxy — Standalone / Control / Agent" width="1200">
</p>

# UnifiedProxy

[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)

UnifiedProxy は Mihomo ネイティブの管理パネルとフリート管理を統合し、単独・コントロール・Agent の各モードに対応します。インバウンド、クライアント、サブスクリプション、署名付きタスク、Sidecar、中央証明書、WebDAV バックアップと更新を 11 言語で管理できます。5 種類の配色と明暗を独立して選べ、既定は T0 白黒のライトです。

[導入と移行](docs/unifiedproxy-migration.md)を参照してください。旧バックアップの認証情報・使用量・サブスクリプション別名を移行できます。証明書ファイル、ノード再登録、NaïveProxy の例外は[互換性記録](docs/design/legacy-acceptance.md)に記載しています。

[简体中文](/README.md) · [English](/README.en.md) · [Languages](#languages)

<a id="deploy"></a>

## Linux へのインストール

systemd または OpenRC を使う Linux で、`root` として実行します：

```bash
bash <(curl -Ls https://raw.githubusercontent.com/HlONGlin/UnifiedProxy/main/install.sh)
```

インストーラーは未使用の 5 桁のパネルポートを選び、16 文字のパネルパス、14 文字のユーザー名、16 文字のパスワードを生成します。アクセス URL と認証情報は最後に一度だけ表示されます。明示的に指定することもできます：

```bash
bash <(curl -Ls https://raw.githubusercontent.com/HlONGlin/UnifiedProxy/main/install.sh) \
  --port 34567 \
  --path MyPanelA1b2C3d4 \
  --username AdminUserA1234 \
  --password 'ChangeThisA12345'
```

UnifiedProxy は `/usr/local/m-ui` にインストールされ、データは `/usr/local/m-ui/data` に保存されます。インストーラーは現在のアーキテクチャ用の Mihomo コアもダウンロードします。既存のコアは `--update-mihomo` を指定しない限り保持されます。

`m-ui update` は設定を保持したまま UnifiedProxy を更新します。更新前にデータディレクトリのスナップショットを `data/backups/` に作成し、最新の 3 つを保持します。

UnifiedProxy のリリースには `386`、`amd64`、`arm64`、`armv5`、`armv6`、`armv7`、`s390x` 向けの `unifiedproxy-linux-<architecture>.tar.gz` と `unifiedproxy-windows-amd64.zip` が含まれます。サービス名と管理コマンドには互換性のある `m-ui` 名を引き続き使用します。

<details>
<summary>Docker</summary>

Docker イメージは Linux amd64 向けで、UnifiedProxy と、バージョン固定かつ検証済みの Mihomo コアを含みます。Compose はホストネットワークを使うため、パネルで追加したインバウンドのポートは追加のポートマッピングなしで使えます。

```bash
git clone https://github.com/HlONGlin/UnifiedProxy.git
cd UnifiedProxy
cp .env.example .env
docker compose up -d --build
docker compose logs m-ui
```

このリポジトリからビルドしてください。タグ付きリリースでは `ghcr.io/hlonglin/unifiedproxy:<tag>` と `latest` を公開できます。対応する UnifiedProxy イメージの公開後に `docker compose pull` を使ってください。空の `MUI_PASSWORD` / `MUI_PATH` は初回に生成されログに一度表示されます。互換性のためサービス名 `m-ui` とボリューム `m-ui-data`、`m-ui-core` を維持し、更新時も保持してください。

</details>

<a id="workspace"></a>

## スクリーンショット

![フリート](./media/screenshots/unifiedproxy/fleet.png)

<details>
<summary>スクリーンショット</summary>

**インバウンド**

![インバウンド](./media/screenshots/unifiedproxy/inbounds.png)

**クライアント上限と期限**

![クライアント上限と期限](./media/screenshots/unifiedproxy/client-edit.png)

**アウトバウンド**

![アウトバウンド](./media/screenshots/unifiedproxy/outbounds.png)

**ルーティング**

![ルーティング](./media/screenshots/unifiedproxy/routing.png)

**Xray / 3x-ui インポート**

![Xray / 3x-ui インポート](./media/screenshots/unifiedproxy/xray-import.png)

**ログイン**

![ログイン](./media/screenshots/unifiedproxy/login.png)

**モバイル購読**

![モバイル購読](./media/screenshots/unifiedproxy/subscription-mobile.png)

</details>

### 5 種類の配色

`data-theme="light|dark"` × `data-color-theme="t0|t1|t2|t3|t4"`

| Theme | Palette | Reference |
|---|---|---|
| T0 | 白黒（既定） | Vercel |
| T1 | 暖色の紙とテラコッタ | Claude / Anthropic |
| T2 | 夜のバイオレット | Sentry |
| T3 | エメラルド | Supabase |
| T4 | インディゴ | Stripe |

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

## 機能

<details>
<summary>Multi-control</summary>

* マスターのない P2P パネルネットワーク（署名付き ID）
* 信頼したパネル間での暗号化されたインバウンド同期
* パネル間サブスクリプションの集約
* クライアント端末：[Clash Caesar](https://github.com/RomanovCaesar/clash-caesar) でネットワークファイルを読み込み、パネル管理者アカウントで一度ログインすれば、全パネルの状態を確認し、接続できる任意のパネル経由で各パネルまたは統合サブスクリプションを取得可能

</details>

<details>
<summary>サブスクリプション</summary>

* 使用量、上限、有効期限、ノードを表示するブラウザ向けサブスクリプションページ（Clash Caesar へワンクリックでインポート可能）
* **サブスクリプションテンプレート**：すべてのサブスクリプションが完全なクライアント設定になります。内蔵の中国 / ロシア / イラン版テンプレート、または独自の Mihomo テンプレートから生成します
* Mihomo/Clash、Stash、sing-box、Surge、Surfboard、Loon、Quantumult X、Egern 向けの完全な設定。V2Box、V2RayNG、Shadowrocket などの共有リンク型クライアントにはノード一覧
* ユーザーごとのサブスクリプショントークン、専用サブスクリプションポート（任意）

</details>

<details>
<summary>インバウンドとクライアント</summary>

* `mixed`、`socks`、`http`、`shadowsocks`、`snell`、`vmess`、`vless`、`trojan`、`hysteria2`、`tuic`、`anytls`、`mieru`、`sudoku`、`shadowquic`、`trusttunnel`、`hysteria2-realm` のリスナー
* TLS、Reality、ShadowTLS、RestTLS、TLSMirror、JLS、Trojan SS、XHTTP、mKCP、Mekya、難読化の設定
* クライアントごとの認証情報、通信量の上限、有効期限、リセット周期、期限切れ時の自動更新、オンライン状態。インバウンド単位の上限も設定可能
* 共有リンク、QR コード、Mihomo リスナー YAML からのインバウンドのインポート、通信量の一括リセット

</details>

<details>
<summary>Mihomo 設定</summary>

* Basics：ルーティングモード、直接接続の IP バージョン、IPv6、TCP 同時接続、統計、ログ、基本ルーティング
* アウトバウンド（Snell、WireGuard、OpenVPN を含む）とポリシーグループ。フォーム、ネイティブ YAML、共有リンク変換に対応
* 上から順に評価されるルーティングルール（`AND` ルールにも対応）
* Cloudflare WARP アウトバウンドと、国と ISP で選べる VPNGate の家庭用回線出口

</details>

<details>
<summary>パネルと Mihomo</summary>

* システム、トラフィック、接続、Mihomo の状態を表示するダッシュボード。Mihomo はパネルと一緒に自動起動します
* Mihomo ネイティブ YAML の生成と検証、生成された設定のプレビューとダウンロード
* MetaCubeX 公式リリースからの Mihomo バージョン切り替え、GeoIP / GeoSite / MetaDB の更新
* 完全なバックアップと復元：設定、インバウンド、Multi-control の ID（任意で復元）、パネル間キャッシュ、サブスクリプションテンプレート、Multi-control のクライアントデバイス
* **3x-ui からワンクリックで移行：** `x-ui.db`（v2.9.3–v3.10.2）または Xray の `config.json` をインポートすると、インバウンド、上限・有効期限・通信量付きのクライアント、アウトバウンド、バランサー、ルーティングルールが既存の内容に追加されます。Mihomo で再現できない部分はダウンロードできる変換ログに記録されます
* 英語、簡体字中国語、繁体字中国語、日本語、ロシア語、ペルシア語、ベトナム語、スペイン語、トルコ語、ウクライナ語、ブラジルポルトガル語の UI

</details>

<a id="operations"></a>

## TLS 証明書

初回の対話型インストールでは、Let's Encrypt のドメイン証明書、短期の Let's Encrypt IPv4 証明書、既存の証明書、または TLS を設定しない、から選べます。インストール後は次を実行します：

```bash
m-ui ssl
```

メニューでは、ドメイン証明書と IP 証明書の発行、ポート 80 が使えない場合の Cloudflare DNS API によるドメインとワイルドカード証明書の発行、更新、失効、一覧表示、既存の証明書の適用ができます。証明書ファイルにリーフ証明書しかない場合、UnifiedProxy は証明書チェーンを自動で補完します。Mihomo Party や FlClash などのクライアントがそのような証明書を拒否するためです。

## 管理コマンド

```text
m-ui                  # 対話型メニューを開く
m-ui start | stop | restart | status | logs
m-ui settings | configure | set-port
m-ui reset-credentials | reset-path
m-ui update | update-menu | ssl
m-ui enable | disable | clear-logs | uninstall
m-ui --lang <code>    メニュー言語。例: m-ui --lang ja
```

管理メニューはパネルと同じ 11 言語に対応し、選んだ言語を記憶します。インストーラーは英語のままです。

<a id="docs"></a>

## ドキュメント

* [導入と移行](docs/unifiedproxy-migration.md)
* [モデルとプロトコル](docs/fleet-protocol.md)
* [機能と互換性](docs/design/legacy-acceptance.md)
* [テーマと参照元](docs/design/p2-token-map.md)
* [P5 画面と機能の検証](docs/design/p5-acceptance.md)
* [リリース](https://github.com/HlONGlin/UnifiedProxy/releases)

## 開発

ビルド・テスト・プレビューは orca-linux の Linux コンテナ内 `/workspace` でのみ実行します。Windows は編集と Git に使用します。

```bash
test "$PWD" = /workspace && test -f /.dockerenv
go build ./...
go vet ./...
go test ./...
```

`MUI_VERSION` / Git tag → `-X main.version=...`. [Fleet build / deployment](docs/unifiedproxy-migration.md) · [Design tokens](docs/design/p2-token-map.md).

## 謝辞

* [MetaCubeX/mihomo](https://github.com/MetaCubeX/mihomo)：プロキシコアと設定モデル
* [MHSanaei/3x-ui](https://github.com/MHSanaei/3x-ui)：画面の情報密度と操作の流れ
* [Sub-Store](https://github.com/sub-store-org/Sub-Store)：クライアント向けサブスクリプション形式の参考実装
* m-ui (GPL-3.0), UnifiedProxy (MIT): [NOTICE](NOTICE), [MIT notice](licenses/UnifiedProxy-MIT.txt).
* [Lucide](https://lucide.dev): ISC; [icon attribution](web/licenses/LICENSE-Lucide.txt).
* 配色の参考対象は Vercel / Claude（Anthropic）/ Sentry / Supabase / Stripe です。資料は VoltAgent/awesome-design-md（MIT）に基づきます。詳しくは[テーマ設計記録](docs/design/p2-token-map.md)をご覧ください。

## ライセンス

UnifiedProxy は [GNU General Public License version 3](LICENSE) で提供されています。Mihomo の共有リンク変換コードの GPL-3.0 に関する表記は [`internal/mihomoconvert/NOTICE.md`](internal/mihomoconvert/NOTICE.md) にあります。

> [!IMPORTANT]
> 本プロジェクトは、ご自身が所有する、または管理を許可されたサーバーとネットワークでのみ使用してください。現地の法律、上流プロバイダーの規約、利用するサービスのポリシーを守ってください。

<a id="languages"></a>

<details>
<summary>多语言文档 / Languages</summary>

[简体中文](/README.md) | [English](/README.en.md) | [繁體中文](/README.zh_TW.md) | [日本語](/README.ja.md) | [Русский](/README.ru.md) | [فارسی](/README.fa.md) | [Tiếng Việt](/README.vi.md) | [Español](/README.es.md) | [Türkçe](/README.tr.md) | [Українська](/README.uk.md) | [Português (Brasil)](/README.pt_BR.md)

</details>
