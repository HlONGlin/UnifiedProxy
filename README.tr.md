<p align="center">
  <img src="./media/unifiedproxy-banner.svg" alt="UnifiedProxy — Standalone / Control / Agent" width="1200">
</p>

# UnifiedProxy

[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)

UnifiedProxy, Mihomo yerel panelini bağımsız, kontrol ve Agent modlarında düğüm filosu yönetimiyle birleştirir. Gelen bağlantılar, istemciler, abonelikler, imzalı görevler, Sidecar, sertifikalar, WebDAV yedekleri ve güncellemeler 11 dilde yönetilir. Beş renk paleti açık/koyu moddan bağımsızdır; varsayılan T0 siyah beyaz ve açık moddur.

[Dağıtım ve geçiş](docs/unifiedproxy-migration.md) ile başlayın. Eski yedeklerin kimlik bilgileri, kullanım ve abonelik takma adları korunur; sertifika dosyaları, düğüm kaydı ve NaïveProxy istisnası [uyumluluk kaydında](docs/design/legacy-acceptance.md) açıklanır.

[简体中文](/README.md) · [English](/README.en.md) · [Languages](#languages)

<a id="deploy"></a>

## Linux kurulumu

systemd veya OpenRC kullanan bir Linux sisteminde `root` olarak çalıştırın:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/HlONGlin/unifiedproxy/main/install.sh)
```

Yükleyici boş bir beş haneli panel portu seçer; 16 karakterlik bir panel yolu, 14 karakterlik bir kullanıcı adı ve 16 karakterlik bir parola oluşturur. Erişim adresi ve kimlik bilgileri sonunda yalnızca bir kez gösterilir. Bunları kendiniz de belirleyebilirsiniz:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/HlONGlin/unifiedproxy/main/install.sh) \
  --port 34567 \
  --path MyPanelA1b2C3d4 \
  --username AdminUserA1234 \
  --password 'ChangeThisA12345'
```

UnifiedProxy `/usr/local/unifiedproxy` dizinine kurulur ve verilerini `/usr/local/unifiedproxy/data` içinde tutar. Yükleyici geçerli mimari için bir Mihomo çekirdeği de indirir; mevcut çekirdek, `--update-mihomo` kullanılmadıkça korunur.

`unifiedproxy update`, tüm ayarları koruyarak UnifiedProxy'yi günceller. Güncellemeden önce veri dizininin anlık görüntüsünü `data/backups/` içine alır ve en yeni 3 tanesini saklar.

UnifiedProxy sürümlerinde `386`, `amd64`, `arm64`, `armv5`, `armv6`, `armv7` ve `s390x` için `unifiedproxy-linux-<architecture>.tar.gz` ile `unifiedproxy-windows-amd64.zip` bulunur; hizmet ve yönetim komutu uyumlu `unifiedproxy` adını korur.

<details>
<summary>Docker</summary>

Docker imajı Linux amd64 içindir ve UnifiedProxy ile sabit sürümlü, doğrulanmış bir Mihomo çekirdeği içerir. Compose ana makine ağını kullanır; panelde eklediğiniz girişlerin portları ek port eşlemesi gerekmeden çalışır.

```bash
git clone https://github.com/HlONGlin/unifiedproxy.git
cd unifiedproxy
cp .env.example .env
docker compose up -d --build
docker compose logs unifiedproxy
```

Bu depodan derleyin. Etiketli sürümler `ghcr.io/hlonglin/unifiedproxy:<tag>` ve `latest` yayımlayabilir; ilgili UnifiedProxy imajı yayımlandıktan sonra `docker compose pull` kullanın. Boş `UNIFIEDPROXY_PASSWORD` / `UNIFIEDPROXY_PATH` ilk açılışta oluşturulur ve günlüğe bir kez yazılır. Compose hizmeti `unifiedproxy`, veri birimleri `unifiedproxy-data` ve `unifiedproxy-core` adlarını kullanır.

</details>

<a id="workspace"></a>

## Ekran görüntüleri

![Filo](./media/screenshots/unifiedproxy/fleet.png)

<details>
<summary>Ekran görüntüleri</summary>

**Gelen bağlantılar**

![Gelen bağlantılar](./media/screenshots/unifiedproxy/inbounds.png)

**Kota ve son tarih**

![Kota ve son tarih](./media/screenshots/unifiedproxy/client-edit.png)

**Giden bağlantılar**

![Giden bağlantılar](./media/screenshots/unifiedproxy/outbounds.png)

**Yönlendirme**

![Yönlendirme](./media/screenshots/unifiedproxy/routing.png)

**Xray / 3x-ui içe aktarma**

![Xray / 3x-ui içe aktarma](./media/screenshots/unifiedproxy/xray-import.png)

**Giriş**

![Giriş](./media/screenshots/unifiedproxy/login.png)

**Mobil abonelik**

![Mobil abonelik](./media/screenshots/unifiedproxy/subscription-mobile.png)

</details>

### Beş palet

`data-theme="light|dark"` × `data-color-theme="t0|t1|t2|t3|t4"`

| Theme | Palette | Reference |
|---|---|---|
| T0 | Siyah beyaz (varsayılan) | Vercel |
| T1 | Sıcak kâğıt ve kil | Claude / Anthropic |
| T2 | Gece moru | Sentry |
| T3 | Zümrüt | Supabase |
| T4 | İndigo | Stripe |

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

## Özellikler

<details>
<summary>Multi-control</summary>

* Ana düğümü olmayan, imzalı kimlikler kullanan P2P panel ağı
* Güvenilen paneller arasında şifreli giriş eşitleme
* Paneller arası abonelik birleştirme
* İstemci cihazlar: [Clash Caesar](https://github.com/RomanovCaesar/clash-caesar) ağ dosyasını içe aktarır, panel yönetici hesabıyla bir kez oturum açar; ardından tüm panelleri görür ve erişebildiği herhangi bir panel üzerinden her panelin veya birleşik aboneliği alır

</details>

<details>
<summary>Abonelikler</summary>

* Kullanım, kota, son kullanma tarihi ve düğümleri gösteren, Clash Caesar içine tek tıkla aktarma sunan tarayıcı abonelik sayfası
* **Abonelik şablonları:** her abonelik, yerleşik şablondan (Çin / Rusya / İran) veya kendi Mihomo şablonunuzdan oluşturulan eksiksiz bir istemci profilidir
* Mihomo/Clash, Stash, sing-box, Surge, Surfboard, Loon, Quantumult X ve Egern için tam profiller; V2Box, V2RayNG, Shadowrocket ve diğer bağlantı tabanlı istemciler için düğüm listeleri
* Kullanıcı başına abonelik belirteci ve isteğe bağlı ayrı abonelik portu

</details>

<details>
<summary>Girişler ve istemciler</summary>

* `mixed`, `socks`, `http`, `shadowsocks`, `snell`, `vmess`, `vless`, `trojan`, `hysteria2`, `tuic`, `anytls`, `mieru`, `sudoku`, `shadowquic`, `trusttunnel` ve `hysteria2-realm` dinleyicileri
* TLS, Reality, ShadowTLS, RestTLS, TLSMirror, JLS, Trojan SS, XHTTP, mKCP, Mekya ve gizleme ayarları
* İstemci başına kimlik bilgileri, trafik kotası, son kullanma tarihi, periyodik sıfırlama, otomatik yenileme ve çevrimiçi durum; giriş düzeyinde sınırlar da var
* Paylaşım bağlantıları, QR kodları, Mihomo dinleyici YAML'ından giriş içe aktarma ve toplu trafik sıfırlama

</details>

<details>
<summary>Mihomo Ayarları</summary>

* Basics: yönlendirme modu, doğrudan bağlantılar için IP sürümü, IPv6, eşzamanlı TCP, istatistikler, günlükler ve temel yönlendirme
* Çıkışlar (Snell, WireGuard ve OpenVPN dahil) ve politika grupları; form, yerel YAML ve bağlantı dönüştürme ile
* Yukarıdan aşağıya değerlendirilen yönlendirme kuralları, `AND` kuralları dahil
* Cloudflare WARP çıkışları ve ülke ile operatöre göre VPNGate ev interneti çıkışları

</details>

<details>
<summary>Panel ve Mihomo</summary>

* Sistem, trafik, bağlantı ve Mihomo durumunu gösteren gösterge paneli; Mihomo panelle birlikte otomatik başlar
* Mihomo'nun yerel YAML yapılandırmasını oluşturma ve doğrulama, yapılandırmayı önizleme ve indirme
* MetaCubeX'in resmi sürümlerinden Mihomo sürümü değiştirme, GeoIP / GeoSite / MetaDB güncelleme
* Tam yedekleme ve geri yükleme: ayarlar, girişler, Multi-control kimliği (isteğe bağlı), paneller arası önbellek, abonelik şablonları ve Multi-control istemci cihazları
* **3x-ui'den tek tıkla geçiş:** `x-ui.db` (v2.9.3–v3.10.2) veya Xray `config.json` dosyasını içe aktarın; girişler, limit, bitiş tarihi ve trafiğiyle istemciler, çıkışlar, dengeleyiciler ve yönlendirme kuralları panelde zaten olanların yanına eklenir; Mihomo'nun yapamadığı her şey indirilebilir dönüştürme günlüğünde listelenir
* İngilizce, Basitleştirilmiş Çince, Geleneksel Çince, Japonca, Rusça, Farsça, Vietnamca, İspanyolca, Türkçe, Ukraynaca ve Brezilya Portekizcesi arayüz

</details>

<a id="operations"></a>

## TLS sertifikaları

İlk etkileşimli kurulum; alan adı için Let's Encrypt sertifikası, IPv4 için kısa ömürlü Let's Encrypt sertifikası, mevcut bir sertifika ya da TLS'yi atlama seçenekleri sunar. Kurulumdan sonra şunu çalıştırın:

```bash
unifiedproxy ssl
```

Menü; alan adı ve IP sertifikaları almayı, 80 numaralı port kullanılamadığında Cloudflare DNS API ile alan adı ve joker sertifika almayı, yenilemeyi, iptal etmeyi, listelemeyi ve mevcut bir sertifikayı uygulamayı sağlar. Sertifika dosyası yalnızca son sertifikayı içeriyorsa UnifiedProxy zinciri otomatik tamamlar, çünkü Mihomo Party ve FlClash gibi istemciler böyle sertifikaları reddeder.

## Yönetim komutları

```text
UnifiedProxy                  # etkileşimli menüyü aç
unifiedproxy start | stop | restart | status | logs
unifiedproxy settings | configure | set-port
unifiedproxy reset-credentials | reset-path
unifiedproxy update | update-menu | ssl
unifiedproxy enable | disable | clear-logs | uninstall
unifiedproxy --lang <code>    menü dili, örneğin: unifiedproxy --lang tr
```

Menü, panelle aynı 11 dili destekler ve seçiminizi hatırlar. Yükleyici İngilizce kalır.

<a id="docs"></a>

## Belgeler

* [Dağıtım ve geçiş](docs/unifiedproxy-migration.md)
* [Model ve protokol](docs/fleet-protocol.md)
* [İşlev ve uyumluluk](docs/design/legacy-acceptance.md)
* [Temalar ve kaynaklar](docs/design/p2-token-map.md)
* [P5 incelemesi](docs/design/p5-acceptance.md)
* [Sürümler](https://github.com/HlONGlin/unifiedproxy/releases)

## Geliştirme

Derleme, test ve önizleme yalnızca orca-linux Linux konteynerinde `/workspace` içinde yapılır. Windows yalnızca düzenleme ve Git için kullanılır.

```bash
test "$PWD" = /workspace && test -f /.dockerenv
go build ./...
go vet ./...
go test ./...
```

`UNIFIEDPROXY_VERSION` / Git tag → `-X main.version=...`. [Fleet build / deployment](docs/unifiedproxy-migration.md) · [Design tokens](docs/design/p2-token-map.md).

## Teşekkürler

* [MetaCubeX/mihomo](https://github.com/MetaCubeX/mihomo): proxy çekirdeği ve yapılandırma modeli
* [MHSanaei/3x-ui](https://github.com/MHSanaei/3x-ui): arayüz yoğunluğu ve iş akışı
* [Sub-Store](https://github.com/sub-store-org/Sub-Store): istemci abonelik biçimleri için başvuru davranışı
* UnifiedProxy (GPL-3.0), UnifiedProxy (MIT): [NOTICE](NOTICE), [MIT notice](licenses/UnifiedProxy-MIT.txt).
* [Lucide](https://lucide.dev): ISC; [icon attribution](web/licenses/LICENSE-Lucide.txt).
* Palet referansları: Vercel / Claude (Anthropic) / Sentry / Supabase / Stripe. VoltAgent/awesome-design-md derlemesi MIT lisanslıdır; ayrıntılar [tema tasarım kaydında](docs/design/p2-token-map.md) yer alır.

## Lisans

UnifiedProxy, [GNU General Public License version 3](LICENSE) lisansıyla dağıtılır. Mihomo bağlantı dönüştürme kodu GPL-3.0 atfını [`internal/mihomoconvert/NOTICE.md`](internal/mihomoconvert/NOTICE.md) içinde korur.

> [!IMPORTANT]
> Bu projeyi yalnızca sahibi olduğunuz veya yönetme yetkiniz bulunan sunucu ve ağlarda kullanın. Yerel yasalara, sağlayıcı koşullarına ve eriştiğiniz hizmetlerin politikalarına uyun.

<a id="languages"></a>

<details>
<summary>多语言文档 / Languages</summary>

[简体中文](/README.md) | [English](/README.en.md) | [繁體中文](/README.zh_TW.md) | [日本語](/README.ja.md) | [Русский](/README.ru.md) | [فارسی](/README.fa.md) | [Tiếng Việt](/README.vi.md) | [Español](/README.es.md) | [Türkçe](/README.tr.md) | [Українська](/README.uk.md) | [Português (Brasil)](/README.pt_BR.md)

</details>
