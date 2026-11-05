<p align="center">
  <img src="./media/unifiedproxy-banner.svg" alt="UnifiedProxy — Standalone / Control / Agent" width="1200">
</p>

# UnifiedProxy

[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)

UnifiedProxy объединяет панель с нативной конфигурацией Mihomo и управление парком узлов: автономный режим, контроллер и Agent. Входящие подключения, клиенты, подписки, подписанные задания, Sidecar, сертификаты, WebDAV и обновления доступны на 11 языках. Пять палитр переключаются независимо от светлого и тёмного режима; по умолчанию — чёрно-белая T0 и светлый режим.

Начните с [развёртывания и миграции](docs/unifiedproxy-migration.md). Старые резервные копии сохраняют учётные данные, расход и адреса подписок; требования к файлам сертификатов, повторной регистрации узлов и исключение NaïveProxy описаны в [проверке совместимости](docs/design/legacy-acceptance.md).

[简体中文](/README.md) · [English](/README.en.md) · [Languages](#languages)

<a id="deploy"></a>

## Установка в Linux

Выполните от имени `root` в Linux с systemd или OpenRC:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/HlONGlin/unifiedproxy/main/install.sh)
```

Установщик выбирает свободный пятизначный порт панели и генерирует путь панели из 16 символов, имя пользователя из 14 символов и пароль из 16 символов. Адрес и учётные данные выводятся один раз в конце установки. Их можно задать явно:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/HlONGlin/unifiedproxy/main/install.sh) \
  --port 34567 \
  --path MyPanelA1b2C3d4 \
  --username AdminUserA1234 \
  --password 'ChangeThisA12345'
```

UnifiedProxy устанавливается в `/usr/local/unifiedproxy`, данные хранятся в `/usr/local/unifiedproxy/data`. Установщик также скачивает ядро Mihomo для текущей архитектуры; уже установленное ядро сохраняется, если не указан `--update-mihomo`.

`unifiedproxy update` обновляет UnifiedProxy с сохранением всех настроек. Перед обновлением создаётся снимок каталога данных в `data/backups/`; хранятся три последних снимка.

В релизах UnifiedProxy публикуются `unifiedproxy-linux-<architecture>.tar.gz` для `386`, `amd64`, `arm64`, `armv5`, `armv6`, `armv7` и `s390x`, а также `unifiedproxy-windows-amd64.zip`. Имя службы и команды управления — `unifiedproxy`.

<details>
<summary>Docker</summary>

Образ Docker собран для Linux amd64 и содержит UnifiedProxy и проверенное ядро Mihomo фиксированной версии. Compose использует сеть хоста, поэтому порты входов, добавленных в панели, работают без отдельного проброса портов.

```bash
git clone https://github.com/HlONGlin/unifiedproxy.git
cd unifiedproxy
cp .env.example .env
docker compose up -d --build
docker compose logs unifiedproxy
```

Собирайте из этого репозитория. Релизы с тегами могут публиковать `ghcr.io/hlonglin/unifiedproxy:<tag>` и `latest`; используйте `docker compose pull` после публикации нужного образа UnifiedProxy. Пустые `UNIFIEDPROXY_PASSWORD` / `UNIFIEDPROXY_PATH` генерируются при первом запуске и один раз выводятся в журнал. Сервис Compose называется `unifiedproxy`, тома данных — `unifiedproxy-data` и `unifiedproxy-core`.

</details>

<a id="workspace"></a>

## Скриншоты

![Парк узлов](./media/screenshots/unifiedproxy/fleet.png)

<details>
<summary>Скриншоты</summary>

**Входящие**

![Входящие](./media/screenshots/unifiedproxy/inbounds.png)

**Квоты и срок клиента**

![Квоты и срок клиента](./media/screenshots/unifiedproxy/client-edit.png)

**Исходящие**

![Исходящие](./media/screenshots/unifiedproxy/outbounds.png)

**Маршрутизация**

![Маршрутизация](./media/screenshots/unifiedproxy/routing.png)

**Импорт Xray / 3x-ui**

![Импорт Xray / 3x-ui](./media/screenshots/unifiedproxy/xray-import.png)

**Вход**

![Вход](./media/screenshots/unifiedproxy/login.png)

**Мобильная подписка**

![Мобильная подписка](./media/screenshots/unifiedproxy/subscription-mobile.png)

</details>

### Пять палитр

`data-theme="light|dark"` × `data-color-theme="t0|t1|t2|t3|t4"`

| Theme | Palette | Reference |
|---|---|---|
| T0 | Чёрно-белая (по умолчанию) | Vercel |
| T1 | Тёплая бумага и терракота | Claude / Anthropic |
| T2 | Ночной фиолетовый | Sentry |
| T3 | Изумрудная | Supabase |
| T4 | Индиго | Stripe |

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

## Возможности

<details>
<summary>Multi-control</summary>

* P2P-сеть панелей без главного узла, с подписанными идентичностями
* Зашифрованная синхронизация входов между доверенными панелями
* Объединение подписок с нескольких панелей
* Клиентские устройства: [Clash Caesar](https://github.com/RomanovCaesar/clash-caesar) импортирует файл сети, один раз входит с учётной записью администратора панели и затем видит все панели и получает подписку любой панели или объединённую через любую доступную панель

</details>

<details>
<summary>Подписки</summary>

* Страница подписки в браузере: расход, лимит, срок действия, узлы и импорт в Clash Caesar в один клик
* **Шаблоны подписок:** каждая подписка — это полный профиль клиента, собранный из встроенного шаблона (Китай / Россия / Иран) или вашего собственного шаблона Mihomo
* Полные профили для Mihomo/Clash, Stash, sing-box, Surge, Surfboard, Loon, Quantumult X и Egern; списки узлов для V2Box, V2RayNG, Shadowrocket и других клиентов со ссылками
* Отдельный токен подписки для каждого пользователя и необязательный отдельный порт подписок

</details>

<details>
<summary>Входы и клиенты</summary>

* Слушатели `mixed`, `socks`, `http`, `shadowsocks`, `snell`, `vmess`, `vless`, `trojan`, `hysteria2`, `tuic`, `anytls`, `mieru`, `sudoku`, `shadowquic`, `trusttunnel` и `hysteria2-realm`
* Настройки TLS, Reality, ShadowTLS, RestTLS, TLSMirror, JLS, Trojan SS, XHTTP, mKCP, Mekya и обфускации
* Учётные данные, лимиты трафика, срок действия, периодический сброс, автопродление и онлайн-статус для каждого клиента; лимиты и на уровне входа
* Ссылки и QR-коды, импорт входов из YAML-описания слушателя Mihomo, массовый сброс трафика

</details>

<details>
<summary>Настройки Mihomo</summary>

* Basics: режим маршрутизации, версия IP для прямых соединений, IPv6, параллельный TCP, статистика, журналы и базовая маршрутизация
* Исходящие (включая Snell, WireGuard и OpenVPN) и группы политик: форма, родной YAML и преобразование ссылок
* Правила маршрутизации, проверяемые сверху вниз, включая правила `AND`
* Исходящие Cloudflare WARP и домашние IP через VPNGate с выбором страны и провайдера

</details>

<details>
<summary>Панель и Mihomo</summary>

* Дашборд с состоянием системы, трафика, соединений и Mihomo; Mihomo запускается вместе с панелью
* Генерация и проверка конфигурации в родном YAML Mihomo, просмотр и скачивание итогового файла
* Переключение версий Mihomo из официальных релизов MetaCubeX, обновление GeoIP / GeoSite / MetaDB
* Полное резервное копирование и восстановление: настройки, входы, идентичность Multi-control (по желанию), кэш межпанельных подписок, шаблоны подписок и клиентские устройства Multi-control
* **Переход с 3x-ui в один клик:** импорт `x-ui.db` (v2.9.3–v3.10.2) или `config.json` от Xray; входы, клиенты с лимитами, сроками и трафиком, исходящие, балансировщики и правила маршрутизации добавляются к тому, что уже есть в панели, а всё, что Mihomo не умеет, перечислено в скачиваемом журнале преобразования
* Интерфейс на английском, упрощённом и традиционном китайском, японском, русском, персидском, вьетнамском, испанском, турецком, украинском и бразильском португальском

</details>

<a id="operations"></a>

## TLS-сертификаты

При первой интерактивной установке можно выбрать сертификат Let's Encrypt для домена, краткосрочный сертификат Let's Encrypt для IPv4, существующую пару сертификат/ключ или пропустить TLS. После установки выполните:

```bash
unifiedproxy ssl
```

В меню можно выпустить сертификаты для домена и IP, получить сертификат для домена и wildcard через Cloudflare DNS API, если порт 80 недоступен, а также продлить, отозвать, посмотреть и применить существующий сертификат. Если файл сертификата содержит только конечный сертификат, UnifiedProxy сам дополняет цепочку: клиенты вроде Mihomo Party и FlClash такие сертификаты отклоняют.

## Команды управления

```text
UnifiedProxy                  # открыть меню
unifiedproxy start | stop | restart | status | logs
unifiedproxy settings | configure | set-port
unifiedproxy reset-credentials | reset-path
unifiedproxy update | update-menu | ssl
unifiedproxy enable | disable | clear-logs | uninstall
unifiedproxy --lang <code>    язык меню, например: unifiedproxy --lang ru
```

Меню поддерживает те же 11 языков, что и панель, и запоминает выбор. Установщик остаётся на английском.

<a id="docs"></a>

## Документация

* [Развёртывание и миграция](docs/unifiedproxy-migration.md)
* [Модель и протокол](docs/fleet-protocol.md)
* [Функции и совместимость](docs/design/legacy-acceptance.md)
* [Темы и источники](docs/design/p2-token-map.md)
* [Проверка P5](docs/design/p5-acceptance.md)
* [Релизы](https://github.com/HlONGlin/unifiedproxy/releases)

## Разработка

Сборка, тестирование и предпросмотр выполняются только в Linux-контейнере orca-linux в `/workspace`. Windows используется для редактирования и Git.

```bash
test "$PWD" = /workspace && test -f /.dockerenv
go build ./...
go vet ./...
go test ./...
```

`UNIFIEDPROXY_VERSION` / Git tag → `-X main.version=...`. [Fleet build / deployment](docs/unifiedproxy-migration.md) · [Design tokens](docs/design/p2-token-map.md).

## Благодарности

* [MetaCubeX/mihomo](https://github.com/MetaCubeX/mihomo): прокси-ядро и модель конфигурации
* [MHSanaei/3x-ui](https://github.com/MHSanaei/3x-ui): плотность интерфейса и логика работы
* [Sub-Store](https://github.com/sub-store-org/Sub-Store): эталонное поведение форматов подписок для клиентов
* UnifiedProxy (GPL-3.0), UnifiedProxy (MIT): [NOTICE](NOTICE), [MIT notice](licenses/UnifiedProxy-MIT.txt).
* [Lucide](https://lucide.dev): ISC; [icon attribution](web/licenses/LICENSE-Lucide.txt).
* Референсы палитр: Vercel / Claude (Anthropic) / Sentry / Supabase / Stripe. Подборка VoltAgent/awesome-design-md распространяется по MIT; подробности приведены в [описании тем](docs/design/p2-token-map.md).

## Лицензия

UnifiedProxy распространяется по лицензии [GNU General Public License version 3](LICENSE). Код преобразования ссылок Mihomo сохраняет указание на GPL-3.0 в [`internal/mihomoconvert/NOTICE.md`](internal/mihomoconvert/NOTICE.md).

> [!IMPORTANT]
> Используйте проект только на серверах и в сетях, которыми вы владеете или которыми вам разрешено управлять. Соблюдайте местное законодательство, условия провайдеров и правила сервисов, к которым вы обращаетесь.

<a id="languages"></a>

<details>
<summary>多语言文档 / Languages</summary>

[简体中文](/README.md) | [English](/README.en.md) | [繁體中文](/README.zh_TW.md) | [日本語](/README.ja.md) | [Русский](/README.ru.md) | [فارسی](/README.fa.md) | [Tiếng Việt](/README.vi.md) | [Español](/README.es.md) | [Türkçe](/README.tr.md) | [Українська](/README.uk.md) | [Português (Brasil)](/README.pt_BR.md)

</details>
