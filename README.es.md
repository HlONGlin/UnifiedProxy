<p align="center">
  <img src="./media/unifiedproxy-banner.svg" alt="UnifiedProxy — Standalone / Control / Agent" width="1200">
</p>

# UnifiedProxy

[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)

UnifiedProxy integra un panel nativo de Mihomo con la gestión de nodos en modos independiente, controlador y Agent. Administra entradas, clientes, suscripciones, tareas firmadas, Sidecar, certificados, copias WebDAV y actualizaciones en 11 idiomas. Las cinco paletas son independientes del modo claro/oscuro; el valor predeterminado es T0 en blanco y negro y modo claro.

Empieza con [despliegue y migración](docs/unifiedproxy-migration.md). Las copias antiguas conservan credenciales, consumo y alias de suscripción; los archivos de certificados, el registro de nodos y la excepción NaïveProxy se detallan en [compatibilidad](docs/design/legacy-acceptance.md).

[简体中文](/README.md) · [English](/README.en.md) · [Languages](#languages)

<a id="deploy"></a>

## Instalación en Linux

Ejecuta como `root` en un sistema Linux con systemd u OpenRC:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/HlONGlin/UnifiedProxy/main/install.sh)
```

El instalador elige un puerto libre de cinco dígitos y genera una ruta de panel de 16 caracteres, un usuario de 14 caracteres y una contraseña de 16 caracteres. Al terminar muestra una sola vez la URL de acceso y las credenciales. También puedes indicarlos tú:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/HlONGlin/UnifiedProxy/main/install.sh) \
  --port 34567 \
  --path MyPanelA1b2C3d4 \
  --username AdminUserA1234 \
  --password 'ChangeThisA12345'
```

UnifiedProxy se instala en `/usr/local/m-ui` y guarda sus datos en `/usr/local/m-ui/data`. El instalador también descarga un núcleo de Mihomo para la arquitectura actual; un núcleo ya instalado se conserva salvo que uses `--update-mihomo`.

`m-ui update` actualiza UnifiedProxy conservando todos los ajustes. Antes de actualizar guarda una instantánea del directorio de datos en `data/backups/` y conserva las 3 más recientes.

Cada versión incluye `m-ui-linux-<architecture>.tar.gz` para `386`, `amd64`, `arm64`, `armv5`, `armv6`, `armv7` y `s390x`, además de `m-ui-windows-amd64.zip`.

<details>
<summary>Docker</summary>

La imagen de Docker es para Linux amd64 e incluye UnifiedProxy y un núcleo de Mihomo de versión fija y verificado. Compose usa la red del host, así que los puertos de las entradas que añadas en el panel funcionan sin mapeos de puertos adicionales.

```bash
git clone https://github.com/HlONGlin/UnifiedProxy.git
cd UnifiedProxy
cp .env.example .env
docker compose up -d --build
docker compose logs m-ui
```

Compila desde este repositorio. Las versiones etiquetadas pueden publicar `ghcr.io/hlonglin/unifiedproxy:<tag>` y `latest`; usa `docker compose pull` cuando esté disponible la imagen correspondiente de UnifiedProxy. Los valores vacíos de `MUI_PASSWORD` / `MUI_PATH` se generan al iniciar y aparecen una vez en el registro. Por compatibilidad se conservan el servicio `m-ui` y los volúmenes `m-ui-data`, `m-ui-core`; mantenlos al actualizar.

</details>

<a id="workspace"></a>

## Capturas de pantalla

![Nodos](./media/screenshots/unifiedproxy/fleet.png)

<details>
<summary>Capturas de pantalla</summary>

**Entradas**

![Entradas](./media/screenshots/unifiedproxy/inbounds.png)

**Cuota y caducidad**

![Cuota y caducidad](./media/screenshots/unifiedproxy/client-edit.png)

**Salidas**

![Salidas](./media/screenshots/unifiedproxy/outbounds.png)

**Rutas**

![Rutas](./media/screenshots/unifiedproxy/routing.png)

**Importación Xray / 3x-ui**

![Importación Xray / 3x-ui](./media/screenshots/unifiedproxy/xray-import.png)

**Inicio de sesión**

![Inicio de sesión](./media/screenshots/unifiedproxy/login.png)

**Suscripción móvil**

![Suscripción móvil](./media/screenshots/unifiedproxy/subscription-mobile.png)

</details>

### Cinco paletas

`data-theme="light|dark"` × `data-color-theme="t0|t1|t2|t3|t4"`

| Theme | Palette | Reference |
|---|---|---|
| T0 | Blanco y negro (predeterminada) | Vercel |
| T1 | Papel cálido y terracota | Claude / Anthropic |
| T2 | Violeta nocturno | Sentry |
| T3 | Esmeralda | Supabase |
| T4 | Índigo | Stripe |

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

## Funciones

<details>
<summary>Multi-control</summary>

* Red P2P de paneles sin nodo maestro y con identidades firmadas
* Sincronización cifrada de entradas entre paneles de confianza
* Agregación de suscripciones entre paneles
* Dispositivos cliente: [Clash Caesar](https://github.com/RomanovCaesar/clash-caesar) importa el archivo de red, inicia sesión una vez con la cuenta de administrador del panel y luego ve todos los paneles y obtiene la suscripción de cualquier panel o la combinada a través de cualquier panel alcanzable

</details>

<details>
<summary>Suscripciones</summary>

* Página de suscripción en el navegador con uso, cuota, caducidad, nodos e importación en Clash Caesar con un clic
* **Plantillas de suscripción:** cada suscripción es un perfil completo de cliente, generado a partir de la plantilla integrada (China / Rusia / Irán) o de tu propia plantilla de Mihomo
* Perfiles completos para Mihomo/Clash, Stash, sing-box, Surge, Surfboard, Loon, Quantumult X y Egern; listas de nodos para V2Box, V2RayNG, Shadowrocket y otros clientes de enlaces
* Token de suscripción por usuario y puerto de suscripción dedicado opcional

</details>

<details>
<summary>Entradas y clientes</summary>

* Escuchas `mixed`, `socks`, `http`, `shadowsocks`, `snell`, `vmess`, `vless`, `trojan`, `hysteria2`, `tuic`, `anytls`, `mieru`, `sudoku`, `shadowquic`, `trusttunnel` y `hysteria2-realm`
* Ajustes de TLS, Reality, ShadowTLS, RestTLS, TLSMirror, JLS, Trojan SS, XHTTP, mKCP, Mekya y ofuscación
* Credenciales, cuotas de tráfico, caducidad, reinicios periódicos, renovación automática y estado en línea por cliente; también límites por entrada
* Enlaces para compartir, códigos QR, importación de entradas desde el YAML de un listener de Mihomo y reinicio masivo del tráfico

</details>

<details>
<summary>Ajustes de Mihomo</summary>

* Basics: modo de enrutamiento, versión de IP para conexiones directas, IPv6, TCP concurrente, estadísticas, registros y enrutamiento básico
* Salidas (incluidas Snell, WireGuard y OpenVPN) y grupos de políticas, con formulario, YAML nativo y conversión de enlaces
* Reglas de enrutamiento evaluadas de arriba abajo, incluidas reglas `AND`
* Salidas de Cloudflare WARP y salidas residenciales de VPNGate por país y proveedor

</details>

<details>
<summary>Panel y Mihomo</summary>

* Panel principal con el estado del sistema, el tráfico, las conexiones y Mihomo; Mihomo se inicia automáticamente con el panel
* Generación y validación del YAML nativo de Mihomo, con vista previa y descarga de la configuración
* Cambio de versión de Mihomo desde las versiones oficiales de MetaCubeX y actualización de GeoIP / GeoSite / MetaDB
* Copia de seguridad y restauración completas: ajustes, entradas, identidad de Multi-control (opcional), caché entre paneles, plantillas de suscripción y dispositivos cliente de Multi-control
* **Migración desde 3x-ui con un clic:** importa `x-ui.db` (v2.9.3–v3.10.2) o el `config.json` de Xray; las entradas, los clientes con su límite, caducidad y tráfico, las salidas, los balanceadores y las reglas de enrutamiento se añaden junto a lo que ya tiene el panel, y un registro de conversión descargable enumera lo que Mihomo no puede hacer
* Interfaz en inglés, chino simplificado, chino tradicional, japonés, ruso, persa, vietnamita, español, turco, ucraniano y portugués de Brasil

</details>

<a id="operations"></a>

## Certificados TLS

La primera instalación interactiva ofrece un certificado de Let's Encrypt para un dominio, un certificado de Let's Encrypt de corta duración para IPv4, un certificado propio o seguir sin TLS. Después de instalar, ejecuta:

```bash
m-ui ssl
```

El menú permite emitir certificados para dominios e IP, obtener certificados de dominio y comodín mediante la API DNS de Cloudflare cuando el puerto 80 no está disponible, renovar, revocar, listar y aplicar un certificado existente. Si el archivo del certificado solo contiene el certificado final, UnifiedProxy completa la cadena automáticamente, porque clientes como Mihomo Party y FlClash lo rechazan.

## Comandos de gestión

```text
m-ui                  # abrir el menú interactivo
m-ui start | stop | restart | status | logs
m-ui settings | configure | set-port
m-ui reset-credentials | reset-path
m-ui update | update-menu | ssl
m-ui enable | disable | clear-logs | uninstall
m-ui --lang <code>    idioma del menú, por ejemplo: m-ui --lang es
```

El menú habla los mismos 11 idiomas que el panel y recuerda tu elección. El instalador sigue en inglés.

<a id="docs"></a>

## Documentación

* [Despliegue y migración](docs/unifiedproxy-migration.md)
* [Modelo y protocolo](docs/fleet-protocol.md)
* [Funciones y compatibilidad](docs/design/legacy-acceptance.md)
* [Temas y referencias](docs/design/p2-token-map.md)
* [Revisión P5](docs/design/p5-acceptance.md)
* [Versiones](https://github.com/HlONGlin/UnifiedProxy/releases)

## Desarrollo

Compila, prueba y previsualiza solo en el contenedor Linux orca-linux, en `/workspace`. Windows se usa para edición y Git.

```bash
test "$PWD" = /workspace && test -f /.dockerenv
go build ./...
go vet ./...
go test ./...
```

`MUI_VERSION` / Git tag → `-X main.version=...`. [Fleet build / deployment](docs/unifiedproxy-migration.md) · [Design tokens](docs/design/p2-token-map.md).

## Agradecimientos

* [MetaCubeX/mihomo](https://github.com/MetaCubeX/mihomo): el núcleo proxy y el modelo de configuración
* [MHSanaei/3x-ui](https://github.com/MHSanaei/3x-ui): densidad de la interfaz y flujo de trabajo
* [Sub-Store](https://github.com/sub-store-org/Sub-Store): comportamiento de referencia de los formatos de suscripción
* m-ui (GPL-3.0), UnifiedProxy (MIT): [NOTICE](NOTICE), [MIT notice](licenses/UnifiedProxy-MIT.txt).
* [Lucide](https://lucide.dev): ISC; [icon attribution](web/licenses/LICENSE-Lucide.txt).
* Referencias de las paletas: Vercel / Claude (Anthropic) / Sentry / Supabase / Stripe. La recopilación de VoltAgent/awesome-design-md se distribuye bajo MIT; consulta el [registro de diseño de temas](docs/design/p2-token-map.md).

## Licencia

UnifiedProxy se distribuye bajo la [GNU General Public License version 3](LICENSE). El código de conversión de enlaces de Mihomo conserva su atribución GPL-3.0 en [`internal/mihomoconvert/NOTICE.md`](internal/mihomoconvert/NOTICE.md).

> [!IMPORTANT]
> Usa este proyecto solo en servidores y redes que te pertenezcan o que tengas autorización para administrar. Respeta la legislación local, las condiciones de tus proveedores y las políticas de los servicios a los que accedes.

<a id="languages"></a>

<details>
<summary>多语言文档 / Languages</summary>

[简体中文](/README.md) | [English](/README.en.md) | [繁體中文](/README.zh_TW.md) | [日本語](/README.ja.md) | [Русский](/README.ru.md) | [فارسی](/README.fa.md) | [Tiếng Việt](/README.vi.md) | [Español](/README.es.md) | [Türkçe](/README.tr.md) | [Українська](/README.uk.md) | [Português (Brasil)](/README.pt_BR.md)

</details>
