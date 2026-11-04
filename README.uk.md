[简体中文](/README.md) | [English](/README.en.md) | [繁體中文](/README.zh_TW.md) | [日本語](/README.ja.md) | [Русский](/README.ru.md) | [فارسی](/README.fa.md) | [Tiếng Việt](/README.vi.md) | [Español](/README.es.md) | [Türkçe](/README.tr.md) | [Українська](/README.uk.md) | [Português (Brasil)](/README.pt_BR.md)

<p align="center">
  <img src="./web/media/unifiedproxy.svg" alt="UnifiedProxy" width="96">
</p>

# UnifiedProxy

[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)

UnifiedProxy об’єднує нативну панель Mihomo та керування парком вузлів в автономному режимі, режимі контролера й Agent. Вхідні з’єднання, клієнти, підписки, підписані завдання, Sidecar, сертифікати, WebDAV і оновлення доступні 11 мовами. П’ять палітр незалежні від світлого й темного режимів; типово — чорно-біла T0 та світлий режим.

Почніть із [розгортання та міграції](docs/unifiedproxy-migration.md). Старі резервні копії зберігають облікові дані, використання й адреси підписок; вимоги до сертифікатів, повторної реєстрації вузлів і виняток NaïveProxy наведено у [звіті сумісності](docs/design/legacy-acceptance.md).

> [!IMPORTANT]
> Використовуйте проєкт лише на серверах і в мережах, якими ви володієте або якими вам дозволено керувати. Дотримуйтеся місцевого законодавства, умов провайдерів і правил сервісів, до яких ви звертаєтеся.

### П’ять палітр

`data-theme="light|dark"` × `data-color-theme="t0|t1|t2|t3|t4"`

| Theme | Palette | Reference |
|---|---|---|
| T0 | Чорно-біла (типова) | Vercel |
| T1 | Теплий папір і теракота | Claude / Anthropic |
| T2 | Нічний фіолетовий | Sentry |
| T3 | Смарагдова | Supabase |
| T4 | Індиго | Stripe |

[Tokens / references](docs/design/p2-token-map.md) · [NOTICE](NOTICE)

<p align="center">
  <img src="./media/screenshots/unifiedproxy/inbounds.png" alt="Вхідні з’єднання" width="900">
</p>

## Можливості

**Панель і Mihomo**
* Дашборд зі станом системи, трафіку, з'єднань і Mihomo; Mihomo запускається разом із панеллю
* Генерація та перевірка конфігурації в рідному YAML Mihomo, перегляд і завантаження підсумкового файлу
* Перемикання версій Mihomo з офіційних релізів MetaCubeX, оновлення GeoIP / GeoSite / MetaDB
* Повне резервне копіювання та відновлення: налаштування, входи, ідентичність Multi-control (за бажанням), кеш міжпанельних підписок, шаблони підписок і клієнтські пристрої Multi-control
* **Перехід з 3x-ui в один клік:** імпорт `x-ui.db` (v2.9.3–v3.10.2) або `config.json` від Xray; входи, клієнти з лімітами, термінами й трафіком, вихідні, балансувальники та правила маршрутизації додаються до того, що вже є в панелі, а все, чого Mihomo не вміє, перелічено в журналі перетворення, який можна завантажити
* Інтерфейс англійською, спрощеною та традиційною китайською, японською, російською, перською, в'єтнамською, іспанською, турецькою, українською та бразильською португальською

**Входи та клієнти**
* Слухачі `mixed`, `socks`, `http`, `shadowsocks`, `snell`, `vmess`, `vless`, `trojan`, `hysteria2`, `tuic`, `anytls`, `mieru`, `sudoku`, `shadowquic`, `trusttunnel` і `hysteria2-realm`
* Налаштування TLS, Reality, ShadowTLS, RestTLS, TLSMirror, JLS, Trojan SS, XHTTP, mKCP, Mekya та обфускації
* Облікові дані, ліміти трафіку, термін дії, періодичне скидання, автоподовження та онлайн-статус для кожного клієнта; ліміти й на рівні входу
* Посилання та QR-коди, імпорт входів із YAML-опису слухача Mihomo, масове скидання трафіку

**Підписки**
* Сторінка підписки в браузері: використання, ліміт, термін дії, вузли та імпорт у Clash Caesar в один клік
* **Шаблони підписок:** кожна підписка — це повний профіль клієнта, зібраний із вбудованого шаблону (Китай / Росія / Іран) або вашого власного шаблону Mihomo
* Повні профілі для Mihomo/Clash, Stash, sing-box, Surge, Surfboard, Loon, Quantumult X і Egern; списки вузлів для V2Box, V2RayNG, Shadowrocket та інших клієнтів із посиланнями
* Окремий токен підписки для кожного користувача та необов'язковий окремий порт підписок

**Налаштування Mihomo**
* Basics: режим маршрутизації, версія IP для прямих з'єднань, IPv6, паралельний TCP, статистика, журнали та базова маршрутизація
* Вихідні (зокрема Snell, WireGuard та OpenVPN) і групи політик: форма, рідний YAML і перетворення посилань
* Правила маршрутизації, що перевіряються згори донизу, зокрема правила `AND`
* Вихідні Cloudflare WARP і домашні IP через VPNGate з вибором країни та провайдера

**Multi-control**
* P2P-мережа панелей без головного вузла, з підписаними ідентичностями
* Зашифрована синхронізація входів між довіреними панелями
* Об'єднання підписок із кількох панелей
* Клієнтські пристрої: [Clash Caesar](https://github.com/RomanovCaesar/clash-caesar) імпортує файл мережі, один раз входить з обліковим записом адміністратора панелі, а потім бачить усі панелі й отримує підписку будь-якої панелі або об’єднану через будь-яку доступну панель

## Знімки екрана

Поточний інтерфейс UnifiedProxy з китайськими демонстраційними даними. Кожна палітра має світлий і темний режими.

<table>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/inbounds.png" alt="Вхідні з’єднання"><br><sub>Вхідні з’єднання</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/client-edit.png" alt="Квоти й термін дії"><br><sub>Квоти й термін дії</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/outbounds.png" alt="Вихідні з’єднання"><br><sub>Вихідні з’єднання</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/routing.png" alt="Маршрутизація"><br><sub>Маршрутизація</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/xray-import.png" alt="Імпорт Xray / 3x-ui"><br><sub>Імпорт Xray / 3x-ui</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/fleet.png" alt="Парк вузлів"><br><sub>Парк вузлів</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/login.png" alt="Вхід"><br><sub>Вхід</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/subscription-mobile.png" alt="Мобільна підписка"><br><sub>Мобільна підписка</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/t0-light.png" alt="T0 light"><br><sub>T0 light</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/t0-dark.png" alt="T0 dark"><br><sub>T0 dark</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/t1-light.png" alt="T1 light"><br><sub>T1 light</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/t1-dark.png" alt="T1 dark"><br><sub>T1 dark</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/t2-light.png" alt="T2 light"><br><sub>T2 light</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/t2-dark.png" alt="T2 dark"><br><sub>T2 dark</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/t3-light.png" alt="T3 light"><br><sub>T3 light</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/t3-dark.png" alt="T3 dark"><br><sub>T3 dark</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/t4-light.png" alt="T4 light"><br><sub>T4 light</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/t4-dark.png" alt="T4 dark"><br><sub>T4 dark</sub></td></tr>
</table>

## Встановлення в Linux

Виконайте від імені `root` у Linux із systemd або OpenRC:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/HlONGlin/UnifiedProxy/main/install.sh)
```

Інсталятор обирає вільний п'ятизначний порт панелі та генерує шлях панелі з 16 символів, ім'я користувача з 14 символів і пароль із 16 символів. Адреса та облікові дані виводяться один раз наприкінці встановлення. Їх можна задати явно:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/HlONGlin/UnifiedProxy/main/install.sh) \
  --port 34567 \
  --path MyPanelA1b2C3d4 \
  --username AdminUserA1234 \
  --password 'ChangeThisA12345'
```

UnifiedProxy встановлюється в `/usr/local/m-ui`, дані зберігаються в `/usr/local/m-ui/data`. Інсталятор також завантажує ядро Mihomo для поточної архітектури; наявне ядро зберігається, якщо не вказано `--update-mihomo`.

`m-ui update` оновлює UnifiedProxy зі збереженням усіх налаштувань. Перед оновленням створюється знімок каталогу даних у `data/backups/`; зберігаються три останні знімки.

У релізах є `m-ui-linux-<architecture>.tar.gz` для `386`, `amd64`, `arm64`, `armv5`, `armv6`, `armv7` і `s390x`, а також `m-ui-windows-amd64.zip`.

## Docker

Образ Docker зібрано для Linux amd64; він містить UnifiedProxy і перевірене ядро Mihomo фіксованої версії. Compose використовує мережу хоста, тому порти входів, доданих у панелі, працюють без окремого прокидання портів.

```bash
git clone https://github.com/HlONGlin/UnifiedProxy.git
cd UnifiedProxy
cp .env.example .env
docker compose up -d --build
docker compose logs m-ui
```

Збирайте з цього репозиторію. Релізи з тегами можуть публікувати `ghcr.io/hlonglin/unifiedproxy:<tag>` і `latest`; використовуйте `docker compose pull` після появи відповідного образу UnifiedProxy. Порожні `MUI_PASSWORD` / `MUI_PATH` створюються при першому запуску й один раз виводяться в журнал. Для сумісності збережено сервіс `m-ui` та томи `m-ui-data`, `m-ui-core`; зберігайте їх під час оновлення.

## TLS-сертифікати

Під час першого інтерактивного встановлення можна обрати сертифікат Let's Encrypt для домену, короткостроковий сертифікат Let's Encrypt для IPv4, наявну пару сертифікат/ключ або пропустити TLS. Після встановлення виконайте:

```bash
m-ui ssl
```

У меню можна випустити сертифікати для домену та IP, отримати сертифікат для домену й wildcard через Cloudflare DNS API, якщо порт 80 недоступний, а також поновити, відкликати, переглянути й застосувати наявний сертифікат. Якщо файл сертифіката містить лише кінцевий сертифікат, UnifiedProxy сам доповнює ланцюжок: клієнти на кшталт Mihomo Party і FlClash такі сертифікати відхиляють.

## Команди керування

```text
m-ui                  interactive menu
m-ui start | stop | restart | status | logs
m-ui settings | configure | set-port
m-ui reset-credentials | reset-path
m-ui update | update-menu | ssl
m-ui enable | disable | clear-logs | uninstall
m-ui --lang <code>    menu language, e.g. m-ui --lang uk
```

Меню підтримує ті самі 11 мов, що й панель, і запам'ятовує вибір. Інсталятор залишається англійською.

## Документація

* [Розгортання та міграція](docs/unifiedproxy-migration.md)
* [Модель і протокол](docs/fleet-protocol.md)
* [Функції й сумісність](docs/design/legacy-acceptance.md)
* [Теми та джерела](docs/design/p2-token-map.md)
* [Перевірка P5](docs/design/p5-acceptance.md)
* [Релізи](https://github.com/HlONGlin/UnifiedProxy/releases)

## Розробка

Збирання, тести й попередній перегляд виконуються лише в Linux-контейнері orca-linux у `/workspace`. Windows використовується для редагування й Git.

```bash
test "$PWD" = /workspace && test -f /.dockerenv
go build ./...
go vet ./...
go test ./...
```

`MUI_VERSION` / Git tag → `-X main.version=...`. [Fleet build / deployment](docs/unifiedproxy-migration.md) · [Design tokens](docs/design/p2-token-map.md).

## Подяки

* [MetaCubeX/mihomo](https://github.com/MetaCubeX/mihomo): проксі-ядро та модель конфігурації
* [MHSanaei/3x-ui](https://github.com/MHSanaei/3x-ui): щільність інтерфейсу та логіка роботи
* [Sub-Store](https://github.com/sub-store-org/Sub-Store): еталонна поведінка форматів підписок для клієнтів
* m-ui (GPL-3.0), UnifiedProxy (MIT): [NOTICE](NOTICE), [MIT notice](licenses/UnifiedProxy-MIT.txt).
* [Lucide](https://lucide.dev): ISC; [icon attribution](web/licenses/LICENSE-Lucide.txt).
* Vercel / Claude (Anthropic) / Sentry / Supabase / Stripe: [palette references](docs/design/p2-token-map.md), via VoltAgent/awesome-design-md (MIT).

## Ліцензія

UnifiedProxy поширюється за ліцензією [GNU General Public License version 3](LICENSE). Код перетворення посилань Mihomo зберігає вказівку на GPL-3.0 у [`internal/mihomoconvert/NOTICE.md`](internal/mihomoconvert/NOTICE.md).
