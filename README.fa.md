<div dir="rtl">

<p align="center">
  <img src="./media/unifiedproxy-banner.svg" alt="UnifiedProxy — Standalone / Control / Agent" width="1200">
</p>

# UnifiedProxy

[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)

UnifiedProxy پنل بومی Mihomo را با مدیریت مجموعهٔ گره‌ها در حالت مستقل، کنترل‌کننده و Agent یکپارچه می‌کند. ورودی‌ها، کاربران، اشتراک‌ها، کارهای امضاشده، Sidecar، گواهی مرکزی، پشتیبان WebDAV و به‌روزرسانی در ۱۱ زبان ارائه می‌شوند. پنج پالت رنگ مستقل از حالت روشن و تیره هستند؛ پیش‌فرض T0 سیاه‌وسفید و روشن است.

از [راهنمای استقرار و مهاجرت](docs/unifiedproxy-migration.md) شروع کنید. اطلاعات ورود، مصرف و نام‌های مستعار اشتراک پشتیبان‌های قدیمی منتقل می‌شوند؛ فایل گواهی، ثبت دوبارهٔ گره‌ها و استثنای NaïveProxy در [گزارش سازگاری](docs/design/legacy-acceptance.md) آمده است.

[简体中文](/README.md) · [English](/README.en.md) · [Languages](#languages)

<a id="deploy"></a>

## نصب روی لینوکس

روی لینوکسی که از systemd یا OpenRC استفاده می‌کند، با کاربر `root` اجرا کنید:

<div dir="ltr">

```bash
bash <(curl -Ls https://raw.githubusercontent.com/HlONGlin/UnifiedProxy/main/install.sh)
```

</div>

نصب‌کننده یک پورت پنج‌رقمی آزاد انتخاب می‌کند و یک مسیر پنل ۱۶ نویسه‌ای، یک نام کاربری ۱۴ نویسه‌ای و یک رمز عبور ۱۶ نویسه‌ای می‌سازد. نشانی دسترسی و اطلاعات ورود فقط یک بار در پایان نمایش داده می‌شود. می‌توانید آن‌ها را خودتان هم تعیین کنید:

<div dir="ltr">

```bash
bash <(curl -Ls https://raw.githubusercontent.com/HlONGlin/UnifiedProxy/main/install.sh) \
  --port 34567 \
  --path MyPanelA1b2C3d4 \
  --username AdminUserA1234 \
  --password 'ChangeThisA12345'
```

</div>

UnifiedProxy در `/usr/local/m-ui` نصب می‌شود و داده‌هایش را در `/usr/local/m-ui/data` نگه می‌دارد. نصب‌کننده یک هستهٔ Mihomo متناسب با معماری فعلی هم دانلود می‌کند؛ هستهٔ موجود حفظ می‌شود، مگر اینکه `--update-mihomo` را بدهید.

`m-ui update` با حفظ همهٔ تنظیمات UnifiedProxy را به‌روز می‌کند. پیش از به‌روزرسانی، از پوشهٔ داده در `data/backups/` تصویر لحظه‌ای می‌گیرد و ۳ نسخهٔ آخر را نگه می‌دارد.

هر انتشار شامل `m-ui-linux-<architecture>.tar.gz` برای `386`، `amd64`، `arm64`، `armv5`، `armv6`، `armv7` و `s390x` و همچنین `m-ui-windows-amd64.zip` است.

<details>
<summary>داکر</summary>

ایمیج داکر برای Linux amd64 است و UnifiedProxy را همراه با یک هستهٔ Mihomo با نسخهٔ ثابت و بررسی‌شده در خود دارد. Compose از شبکهٔ میزبان استفاده می‌کند، بنابراین پورت ورودی‌هایی که در پنل اضافه می‌کنید بدون نگاشت پورت جداگانه کار می‌کنند.

<div dir="ltr">

```bash
git clone https://github.com/HlONGlin/UnifiedProxy.git
cd UnifiedProxy
cp .env.example .env
docker compose up -d --build
docker compose logs m-ui
```

</div>

از همین مخزن بسازید. انتشارهای تگ‌دار می‌توانند `ghcr.io/hlonglin/unifiedproxy:<tag>` و `latest` را منتشر کنند؛ پس از موجود شدن ایمیج مربوط از `docker compose pull` استفاده کنید. مقادیر خالی `MUI_PASSWORD` / `MUI_PATH` در اولین اجرا تولید و یک بار در لاگ نمایش داده می‌شوند. برای سازگاری، سرویس `m-ui` و حجم‌های `m-ui-data` و `m-ui-core` حفظ شده‌اند؛ هنگام ارتقا آن‌ها را نگه دارید.

</details>

<a id="workspace"></a>

## تصاویر

![مجموعهٔ گره‌ها](./media/screenshots/unifiedproxy/fleet.png)

<details>
<summary>تصاویر</summary>

**ورودی‌ها**

![ورودی‌ها](./media/screenshots/unifiedproxy/inbounds.png)

**سهمیه و انقضای کاربر**

![سهمیه و انقضای کاربر](./media/screenshots/unifiedproxy/client-edit.png)

**خروجی‌ها**

![خروجی‌ها](./media/screenshots/unifiedproxy/outbounds.png)

**مسیریابی**

![مسیریابی](./media/screenshots/unifiedproxy/routing.png)

**ورود Xray / 3x-ui**

![ورود Xray / 3x-ui](./media/screenshots/unifiedproxy/xray-import.png)

**ورود**

![ورود](./media/screenshots/unifiedproxy/login.png)

**اشتراک موبایل**

![اشتراک موبایل](./media/screenshots/unifiedproxy/subscription-mobile.png)

</details>

### پنج پالت

`data-theme="light|dark"` × `data-color-theme="t0|t1|t2|t3|t4"`

| Theme | Palette | Reference |
|---|---|---|
| T0 | سیاه‌وسفید (پیش‌فرض) | Vercel |
| T1 | کاغذ گرم و سفالی | Claude / Anthropic |
| T2 | بنفش شب | Sentry |
| T3 | زمردی | Supabase |
| T4 | نیلی | Stripe |

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

## امکانات

<details>
<summary>Multi-control</summary>

* شبکهٔ P2P پنل‌ها بدون گرهٔ اصلی، با هویت‌های امضاشده
* همگام‌سازی رمزگذاری‌شدهٔ ورودی‌ها میان پنل‌های مورد اعتماد
* تجمیع اشتراک‌ها از چند پنل
* دستگاه‌های کلاینت: [Clash Caesar](https://github.com/RomanovCaesar/clash-caesar) فایل شبکه را وارد می‌کند، یک بار با حساب مدیر پنل وارد می‌شود و سپس همهٔ پنل‌ها را می‌بیند و اشتراک هر پنل یا اشتراک ادغام‌شده را از هر پنل در دسترس دریافت می‌کند

</details>

<details>
<summary>اشتراک‌ها</summary>

* صفحهٔ اشتراک در مرورگر با مصرف، سهمیه، تاریخ انقضا، گره‌ها و وارد کردن با یک کلیک به Clash Caesar
* **الگوهای اشتراک:** هر اشتراک یک پیکربندی کامل کلاینت است که از الگوی داخلی (چین / روسیه / ایران) یا الگوی Mihomo خودتان ساخته می‌شود
* پیکربندی کامل برای Mihomo/Clash، Stash، sing-box، Surge، Surfboard، Loon، Quantumult X و Egern؛ فهرست گره‌ها برای V2Box، V2RayNG، Shadowrocket و دیگر کلاینت‌های مبتنی بر لینک
* توکن اشتراک جداگانه برای هر کاربر و پورت اشتراک اختصاصی (اختیاری)

</details>

<details>
<summary>ورودی‌ها و کلاینت‌ها</summary>

* شنونده‌های `mixed`، `socks`، `http`، `shadowsocks`، `snell`، `vmess`، `vless`، `trojan`، `hysteria2`، `tuic`، `anytls`، `mieru`، `sudoku`، `shadowquic`، `trusttunnel` و `hysteria2-realm`
* تنظیمات TLS، Reality، ShadowTLS، RestTLS، TLSMirror، JLS، Trojan SS، XHTTP، mKCP، Mekya و مبهم‌سازی
* اطلاعات احراز هویت، سهمیهٔ ترافیک، تاریخ انقضا، بازنشانی دوره‌ای، تمدید خودکار و وضعیت آنلاین برای هر کلاینت؛ محدودیت در سطح ورودی هم وجود دارد
* لینک‌های اشتراک‌گذاری، کد QR، درون‌ریزی ورودی از YAML شنوندهٔ Mihomo و بازنشانی گروهی ترافیک

</details>

<details>
<summary>تنظیمات Mihomo</summary>

* Basics: حالت مسیریابی، نسخهٔ IP برای اتصال مستقیم، IPv6، TCP همزمان، آمار، لاگ‌ها و مسیریابی پایه
* خروجی‌ها (از جمله Snell، WireGuard و OpenVPN) و گروه‌های سیاست، با فرم، YAML بومی و تبدیل لینک
* قوانین مسیریابی که از بالا به پایین بررسی می‌شوند، از جمله قوانین `AND`
* خروجی‌های Cloudflare WARP و خروجی IP خانگی VPNGate بر اساس کشور و اپراتور

</details>

<details>
<summary>پنل و Mihomo</summary>

* داشبورد وضعیت سیستم، ترافیک، اتصال‌ها و Mihomo؛ Mihomo همراه با پنل به‌طور خودکار اجرا می‌شود
* تولید و اعتبارسنجی پیکربندی YAML بومی Mihomo، پیش‌نمایش و دانلود پیکربندی
* تغییر نسخهٔ Mihomo از انتشارهای رسمی MetaCubeX و به‌روزرسانی GeoIP / GeoSite / MetaDB
* پشتیبان‌گیری و بازیابی کامل: تنظیمات، ورودی‌ها، هویت Multi-control (اختیاری)، حافظهٔ میان‌پنلی، الگوهای اشتراک و دستگاه‌های کلاینت Multi-control
* **مهاجرت یک‌کلیکی از 3x-ui:** فایل `x-ui.db` (نسخهٔ v2.9.3 تا v3.10.2) یا `config.json` مربوط به Xray را وارد کنید؛ ورودی‌ها، کلاینت‌ها با محدودیت، تاریخ انقضا و ترافیک، خروجی‌ها، متعادل‌کننده‌ها و قوانین مسیریابی کنار محتوای فعلی پنل اضافه می‌شوند و هر چیزی که Mihomo پشتیبانی نمی‌کند در گزارش تبدیل قابل دانلود فهرست می‌شود
* رابط کاربری به زبان‌های انگلیسی، چینی ساده‌شده، چینی سنتی، ژاپنی، روسی، فارسی، ویتنامی، اسپانیایی، ترکی، اوکراینی و پرتغالی برزیل

</details>

<a id="operations"></a>

## گواهی‌های TLS

نخستین نصب تعاملی این گزینه‌ها را ارائه می‌دهد: گواهی Let's Encrypt برای دامنه، گواهی کوتاه‌مدت Let's Encrypt برای IPv4، گواهی موجود، یا رد شدن از TLS. پس از نصب اجرا کنید:

<div dir="ltr">

```bash
m-ui ssl
```

</div>

این منو می‌تواند گواهی دامنه و IP صادر کند، وقتی پورت 80 در دسترس نیست با API DNS کلادفلر گواهی دامنه و wildcard بگیرد، و گواهی‌ها را تمدید، ابطال، فهرست و روی پنل اعمال کند. اگر فایل گواهی فقط گواهی نهایی را داشته باشد، UnifiedProxy زنجیرهٔ گواهی را خودکار کامل می‌کند، چون کلاینت‌هایی مانند Mihomo Party و FlClash چنین گواهی‌ای را نمی‌پذیرند.

## دستورهای مدیریت



<div dir="ltr">

```text
m-ui                  # باز کردن منوی تعاملی
m-ui start | stop | restart | status | logs
m-ui settings | configure | set-port
m-ui reset-credentials | reset-path
m-ui update | update-menu | ssl
m-ui enable | disable | clear-logs | uninstall
m-ui --lang <code>    زبان منو، برای نمونه: m-ui --lang fa
```

</div>



منوی مدیریت از همان ۱۱ زبان پنل پشتیبانی می‌کند و انتخاب شما را به خاطر می‌سپارد. نصب‌کننده به انگلیسی باقی می‌ماند.

<a id="docs"></a>

## مستندات

* [استقرار و مهاجرت](docs/unifiedproxy-migration.md)
* [مدل و پروتکل](docs/fleet-protocol.md)
* [امکانات و سازگاری](docs/design/legacy-acceptance.md)
* [رنگ‌ها و منابع](docs/design/p2-token-map.md)
* [بررسی P5](docs/design/p5-acceptance.md)
* [انتشارها](https://github.com/HlONGlin/UnifiedProxy/releases)

## توسعه

ساخت، آزمون و پیش‌نمایش فقط در کانتینر لینوکس orca-linux در `/workspace` انجام می‌شود. ویندوز فقط برای ویرایش و Git است.

<div dir="ltr">

```bash
test "$PWD" = /workspace && test -f /.dockerenv
go build ./...
go vet ./...
go test ./...
```

</div>

`MUI_VERSION` / Git tag → `-X main.version=...`. [Fleet build / deployment](docs/unifiedproxy-migration.md) · [Design tokens](docs/design/p2-token-map.md).

## سپاسگزاری

* [MetaCubeX/mihomo](https://github.com/MetaCubeX/mihomo): هستهٔ پروکسی و مدل پیکربندی
* [MHSanaei/3x-ui](https://github.com/MHSanaei/3x-ui): تراکم اطلاعات و روند کار رابط کاربری
* [Sub-Store](https://github.com/sub-store-org/Sub-Store): رفتار مرجع برای قالب‌های اشتراک کلاینت‌ها
* m-ui (GPL-3.0), UnifiedProxy (MIT): [NOTICE](NOTICE), [MIT notice](licenses/UnifiedProxy-MIT.txt).
* [Lucide](https://lucide.dev): ISC; [icon attribution](web/licenses/LICENSE-Lucide.txt).
* مرجع‌های پالت رنگ Vercel / Claude (Anthropic) / Sentry / Supabase / Stripe هستند. گردآوری VoltAgent/awesome-design-md با مجوز MIT است؛ جزئیات در [سند طراحی پوسته](docs/design/p2-token-map.md) آمده است.

## مجوز

UnifiedProxy تحت مجوز [GNU General Public License version 3](LICENSE) منتشر می‌شود. کد تبدیل لینک Mihomo انتساب GPL-3.0 خود را در [`internal/mihomoconvert/NOTICE.md`](internal/mihomoconvert/NOTICE.md) حفظ می‌کند.

> [!IMPORTANT]
> این پروژه را فقط روی سرورها و شبکه‌هایی به کار ببرید که مالک آن‌ها هستید یا اجازهٔ مدیریتشان را دارید. قوانین محلی، شرایط ارائه‌دهندگان و سیاست‌های سرویس‌هایی را که به آن‌ها دسترسی دارید رعایت کنید.

</div>

<a id="languages"></a>

<details>
<summary>多语言文档 / Languages</summary>

[简体中文](/README.md) | [English](/README.en.md) | [繁體中文](/README.zh_TW.md) | [日本語](/README.ja.md) | [Русский](/README.ru.md) | [فارسی](/README.fa.md) | [Tiếng Việt](/README.vi.md) | [Español](/README.es.md) | [Türkçe](/README.tr.md) | [Українська](/README.uk.md) | [Português (Brasil)](/README.pt_BR.md)

</details>
