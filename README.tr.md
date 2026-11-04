[简体中文](/README.md) | [English](/README.en.md) | [繁體中文](/README.zh_TW.md) | [日本語](/README.ja.md) | [Русский](/README.ru.md) | [فارسی](/README.fa.md) | [Tiếng Việt](/README.vi.md) | [Español](/README.es.md) | [Türkçe](/README.tr.md) | [Українська](/README.uk.md) | [Português (Brasil)](/README.pt_BR.md)

<p align="center">
  <img src="./web/media/unifiedproxy.svg" alt="UnifiedProxy" width="96">
</p>

# UnifiedProxy

[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)

UnifiedProxy, Mihomo yerel panelini bağımsız, kontrol ve Agent modlarında düğüm filosu yönetimiyle birleştirir. Gelen bağlantılar, istemciler, abonelikler, imzalı görevler, Sidecar, sertifikalar, WebDAV yedekleri ve güncellemeler 11 dilde yönetilir. Beş renk paleti açık/koyu moddan bağımsızdır; varsayılan T0 siyah beyaz ve açık moddur.

[Dağıtım ve geçiş](docs/unifiedproxy-migration.md) ile başlayın. Eski yedeklerin kimlik bilgileri, kullanım ve abonelik takma adları korunur; sertifika dosyaları, düğüm kaydı ve NaïveProxy istisnası [uyumluluk kaydında](docs/design/legacy-acceptance.md) açıklanır.

> [!IMPORTANT]
> Bu projeyi yalnızca sahibi olduğunuz veya yönetme yetkiniz bulunan sunucu ve ağlarda kullanın. Yerel yasalara, sağlayıcı koşullarına ve eriştiğiniz hizmetlerin politikalarına uyun.

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

<p align="center">
  <img src="./media/screenshots/unifiedproxy/inbounds.png" alt="Gelen bağlantılar" width="900">
</p>

## Özellikler

**Panel ve Mihomo**
* Sistem, trafik, bağlantı ve Mihomo durumunu gösteren gösterge paneli; Mihomo panelle birlikte otomatik başlar
* Mihomo'nun yerel YAML yapılandırmasını oluşturma ve doğrulama, yapılandırmayı önizleme ve indirme
* MetaCubeX'in resmi sürümlerinden Mihomo sürümü değiştirme, GeoIP / GeoSite / MetaDB güncelleme
* Tam yedekleme ve geri yükleme: ayarlar, girişler, Multi-control kimliği (isteğe bağlı), paneller arası önbellek, abonelik şablonları ve Multi-control istemci cihazları
* **3x-ui'den tek tıkla geçiş:** `x-ui.db` (v2.9.3–v3.10.2) veya Xray `config.json` dosyasını içe aktarın; girişler, limit, bitiş tarihi ve trafiğiyle istemciler, çıkışlar, dengeleyiciler ve yönlendirme kuralları panelde zaten olanların yanına eklenir; Mihomo'nun yapamadığı her şey indirilebilir dönüştürme günlüğünde listelenir
* İngilizce, Basitleştirilmiş Çince, Geleneksel Çince, Japonca, Rusça, Farsça, Vietnamca, İspanyolca, Türkçe, Ukraynaca ve Brezilya Portekizcesi arayüz

**Girişler ve istemciler**
* `mixed`, `socks`, `http`, `shadowsocks`, `snell`, `vmess`, `vless`, `trojan`, `hysteria2`, `tuic`, `anytls`, `mieru`, `sudoku`, `shadowquic`, `trusttunnel` ve `hysteria2-realm` dinleyicileri
* TLS, Reality, ShadowTLS, RestTLS, TLSMirror, JLS, Trojan SS, XHTTP, mKCP, Mekya ve gizleme ayarları
* İstemci başına kimlik bilgileri, trafik kotası, son kullanma tarihi, periyodik sıfırlama, otomatik yenileme ve çevrimiçi durum; giriş düzeyinde sınırlar da var
* Paylaşım bağlantıları, QR kodları, Mihomo dinleyici YAML'ından giriş içe aktarma ve toplu trafik sıfırlama

**Abonelikler**
* Kullanım, kota, son kullanma tarihi ve düğümleri gösteren, Clash Caesar içine tek tıkla aktarma sunan tarayıcı abonelik sayfası
* **Abonelik şablonları:** her abonelik, yerleşik şablondan (Çin / Rusya / İran) veya kendi Mihomo şablonunuzdan oluşturulan eksiksiz bir istemci profilidir
* Mihomo/Clash, Stash, sing-box, Surge, Surfboard, Loon, Quantumult X ve Egern için tam profiller; V2Box, V2RayNG, Shadowrocket ve diğer bağlantı tabanlı istemciler için düğüm listeleri
* Kullanıcı başına abonelik belirteci ve isteğe bağlı ayrı abonelik portu

**Mihomo Ayarları**
* Basics: yönlendirme modu, doğrudan bağlantılar için IP sürümü, IPv6, eşzamanlı TCP, istatistikler, günlükler ve temel yönlendirme
* Çıkışlar (Snell, WireGuard ve OpenVPN dahil) ve politika grupları; form, yerel YAML ve bağlantı dönüştürme ile
* Yukarıdan aşağıya değerlendirilen yönlendirme kuralları, `AND` kuralları dahil
* Cloudflare WARP çıkışları ve ülke ile operatöre göre VPNGate ev interneti çıkışları

**Multi-control**
* Ana düğümü olmayan, imzalı kimlikler kullanan P2P panel ağı
* Güvenilen paneller arasında şifreli giriş eşitleme
* Paneller arası abonelik birleştirme
* İstemci cihazlar: [Clash Caesar](https://github.com/RomanovCaesar/clash-caesar) ağ dosyasını içe aktarır, panel yönetici hesabıyla bir kez oturum açar; ardından tüm panelleri görür ve erişebildiği herhangi bir panel üzerinden her panelin veya birleşik aboneliği alır

## Ekran görüntüleri

Güncel UnifiedProxy arayüzü, Çince örnek verilerle gösterilir. Her palet açık ve koyu modu destekler.

<table>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/inbounds.png" alt="Gelen bağlantılar"><br><sub>Gelen bağlantılar</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/client-edit.png" alt="Kota ve son tarih"><br><sub>Kota ve son tarih</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/outbounds.png" alt="Giden bağlantılar"><br><sub>Giden bağlantılar</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/routing.png" alt="Yönlendirme"><br><sub>Yönlendirme</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/xray-import.png" alt="Xray / 3x-ui içe aktarma"><br><sub>Xray / 3x-ui içe aktarma</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/fleet.png" alt="Filo"><br><sub>Filo</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/login.png" alt="Giriş"><br><sub>Giriş</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/subscription-mobile.png" alt="Mobil abonelik"><br><sub>Mobil abonelik</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/t0-light.png" alt="T0 light"><br><sub>T0 light</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/t0-dark.png" alt="T0 dark"><br><sub>T0 dark</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/t1-light.png" alt="T1 light"><br><sub>T1 light</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/t1-dark.png" alt="T1 dark"><br><sub>T1 dark</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/t2-light.png" alt="T2 light"><br><sub>T2 light</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/t2-dark.png" alt="T2 dark"><br><sub>T2 dark</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/t3-light.png" alt="T3 light"><br><sub>T3 light</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/t3-dark.png" alt="T3 dark"><br><sub>T3 dark</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/t4-light.png" alt="T4 light"><br><sub>T4 light</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/t4-dark.png" alt="T4 dark"><br><sub>T4 dark</sub></td></tr>
</table>

## Linux kurulumu

systemd veya OpenRC kullanan bir Linux sisteminde `root` olarak çalıştırın:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/HlONGlin/UnifiedProxy/main/install.sh)
```

Yükleyici boş bir beş haneli panel portu seçer; 16 karakterlik bir panel yolu, 14 karakterlik bir kullanıcı adı ve 16 karakterlik bir parola oluşturur. Erişim adresi ve kimlik bilgileri sonunda yalnızca bir kez gösterilir. Bunları kendiniz de belirleyebilirsiniz:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/HlONGlin/UnifiedProxy/main/install.sh) \
  --port 34567 \
  --path MyPanelA1b2C3d4 \
  --username AdminUserA1234 \
  --password 'ChangeThisA12345'
```

UnifiedProxy `/usr/local/m-ui` dizinine kurulur ve verilerini `/usr/local/m-ui/data` içinde tutar. Yükleyici geçerli mimari için bir Mihomo çekirdeği de indirir; mevcut çekirdek, `--update-mihomo` kullanılmadıkça korunur.

`m-ui update`, tüm ayarları koruyarak UnifiedProxy'yi günceller. Güncellemeden önce veri dizininin anlık görüntüsünü `data/backups/` içine alır ve en yeni 3 tanesini saklar.

Her sürümde `386`, `amd64`, `arm64`, `armv5`, `armv6`, `armv7` ve `s390x` için `m-ui-linux-<architecture>.tar.gz` ile `m-ui-windows-amd64.zip` bulunur.

## Docker

Docker imajı Linux amd64 içindir ve UnifiedProxy ile sabit sürümlü, doğrulanmış bir Mihomo çekirdeği içerir. Compose ana makine ağını kullanır; panelde eklediğiniz girişlerin portları ek port eşlemesi gerekmeden çalışır.

```bash
git clone https://github.com/HlONGlin/UnifiedProxy.git
cd UnifiedProxy
cp .env.example .env
docker compose up -d --build
docker compose logs m-ui
```

Bu depodan derleyin. Etiketli sürümler `ghcr.io/hlonglin/unifiedproxy:<tag>` ve `latest` yayımlayabilir; ilgili UnifiedProxy imajı yayımlandıktan sonra `docker compose pull` kullanın. Boş `MUI_PASSWORD` / `MUI_PATH` ilk açılışta oluşturulur ve günlüğe bir kez yazılır. Uyumluluk için `m-ui` hizmeti ve `m-ui-data`, `m-ui-core` birimleri korunur; yükseltirken bu birimleri saklayın.

## TLS sertifikaları

İlk etkileşimli kurulum; alan adı için Let's Encrypt sertifikası, IPv4 için kısa ömürlü Let's Encrypt sertifikası, mevcut bir sertifika ya da TLS'yi atlama seçenekleri sunar. Kurulumdan sonra şunu çalıştırın:

```bash
m-ui ssl
```

Menü; alan adı ve IP sertifikaları almayı, 80 numaralı port kullanılamadığında Cloudflare DNS API ile alan adı ve joker sertifika almayı, yenilemeyi, iptal etmeyi, listelemeyi ve mevcut bir sertifikayı uygulamayı sağlar. Sertifika dosyası yalnızca son sertifikayı içeriyorsa UnifiedProxy zinciri otomatik tamamlar, çünkü Mihomo Party ve FlClash gibi istemciler böyle sertifikaları reddeder.

## Yönetim komutları

```text
m-ui                  interactive menu
m-ui start | stop | restart | status | logs
m-ui settings | configure | set-port
m-ui reset-credentials | reset-path
m-ui update | update-menu | ssl
m-ui enable | disable | clear-logs | uninstall
m-ui --lang <code>    menu language, e.g. m-ui --lang tr
```

Menü, panelle aynı 11 dili destekler ve seçiminizi hatırlar. Yükleyici İngilizce kalır.

## Belgeler

* [Dağıtım ve geçiş](docs/unifiedproxy-migration.md)
* [Model ve protokol](docs/fleet-protocol.md)
* [İşlev ve uyumluluk](docs/design/legacy-acceptance.md)
* [Temalar ve kaynaklar](docs/design/p2-token-map.md)
* [P5 incelemesi](docs/design/p5-acceptance.md)
* [Sürümler](https://github.com/HlONGlin/UnifiedProxy/releases)

## Geliştirme

Derleme, test ve önizleme yalnızca orca-linux Linux konteynerinde `/workspace` içinde yapılır. Windows yalnızca düzenleme ve Git için kullanılır.

```bash
test "$PWD" = /workspace && test -f /.dockerenv
go build ./...
go vet ./...
go test ./...
```

`MUI_VERSION` / Git tag → `-X main.version=...`. [Fleet build / deployment](docs/unifiedproxy-migration.md) · [Design tokens](docs/design/p2-token-map.md).

## Teşekkürler

* [MetaCubeX/mihomo](https://github.com/MetaCubeX/mihomo): proxy çekirdeği ve yapılandırma modeli
* [MHSanaei/3x-ui](https://github.com/MHSanaei/3x-ui): arayüz yoğunluğu ve iş akışı
* [Sub-Store](https://github.com/sub-store-org/Sub-Store): istemci abonelik biçimleri için başvuru davranışı
* m-ui (GPL-3.0), UnifiedProxy (MIT): [NOTICE](NOTICE), [MIT notice](licenses/UnifiedProxy-MIT.txt).
* [Lucide](https://lucide.dev): ISC; [icon attribution](web/licenses/LICENSE-Lucide.txt).
* Vercel / Claude (Anthropic) / Sentry / Supabase / Stripe: [palette references](docs/design/p2-token-map.md), via VoltAgent/awesome-design-md (MIT).

## Lisans

UnifiedProxy, [GNU General Public License version 3](LICENSE) lisansıyla dağıtılır. Mihomo bağlantı dönüştürme kodu GPL-3.0 atfını [`internal/mihomoconvert/NOTICE.md`](internal/mihomoconvert/NOTICE.md) içinde korur.
