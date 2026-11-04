[English](/README.md) | [简体中文](/README.zh_CN.md) | [繁體中文](/README.zh_TW.md) | [日本語](/README.ja.md) | [Русский](/README.ru.md) | [فارسی](/README.fa.md) | [Tiếng Việt](/README.vi.md) | [Español](/README.es.md) | [Türkçe](/README.tr.md) | [Українська](/README.uk.md) | [Português (Brasil)](/README.pt_BR.md)

<p align="center">
  <img src="./web/media/unifiedproxy.svg" alt="UnifiedProxy" width="96">
</p>

# UnifiedProxy

[![License: GPL-3.0](https://img.shields.io/badge/License-GPL--3.0-blue.svg)](LICENSE)

O UnifiedProxy integra um painel nativo do Mihomo à gestão de nós nos modos independente, controlador e Agent. Gerencie entradas, clientes, assinaturas, tarefas assinadas, Sidecar, certificados, backups WebDAV e atualizações em 11 idiomas. As cinco paletas são independentes do modo claro/escuro; o padrão é T0 preto e branco no modo claro.

Comece por [implantação e migração](docs/unifiedproxy-migration.md). Backups antigos preservam credenciais, consumo e aliases de assinatura; arquivos de certificado, novo registro de nós e a exceção NaïveProxy estão no [registro de compatibilidade](docs/design/legacy-acceptance.md).

> [!IMPORTANT]
> Use este projeto apenas em servidores e redes que sejam seus ou que você tenha autorização para administrar. Respeite a legislação local, os termos dos seus provedores e as políticas dos serviços que você acessa.

### Cinco paletas

`data-theme="light|dark"` × `data-color-theme="t0|t1|t2|t3|t4"`

| Theme | Palette | Reference |
|---|---|---|
| T0 | Preto e branco (padrão) | Vercel |
| T1 | Papel quente e terracota | Claude / Anthropic |
| T2 | Violeta noturno | Sentry |
| T3 | Esmeralda | Supabase |
| T4 | Índigo | Stripe |

[Tokens / references](docs/design/p2-token-map.md) · [NOTICE](NOTICE)

<p align="center">
  <img src="./media/screenshots/unifiedproxy/inbounds.png" alt="Entradas" width="900">
</p>

## Recursos

**Painel e Mihomo**
* Painel inicial com o estado do sistema, do tráfego, das conexões e do Mihomo; o Mihomo inicia automaticamente com o painel
* Geração e validação do YAML nativo do Mihomo, com visualização e download da configuração
* Troca de versão do Mihomo a partir das versões oficiais da MetaCubeX e atualização de GeoIP / GeoSite / MetaDB
* Backup e restauração completos: configurações, entradas, identidade do Multi-control (opcional), cache entre painéis, modelos de assinatura e dispositivos cliente do Multi-control
* **Migração do 3x-ui com um clique:** importe o `x-ui.db` (v2.9.3–v3.10.2) ou o `config.json` do Xray; entradas, clientes com limite, validade e tráfego, saídas, balanceadores e regras de roteamento são adicionados ao que o painel já tem, e um log de conversão para download lista tudo o que o Mihomo não consegue fazer
* Interface em inglês, chinês simplificado, chinês tradicional, japonês, russo, persa, vietnamita, espanhol, turco, ucraniano e português do Brasil

**Entradas e clientes**
* Listeners `mixed`, `socks`, `http`, `shadowsocks`, `snell`, `vmess`, `vless`, `trojan`, `hysteria2`, `tuic`, `anytls`, `mieru`, `sudoku`, `shadowquic`, `trusttunnel` e `hysteria2-realm`
* Configurações de TLS, Reality, ShadowTLS, RestTLS, TLSMirror, JLS, Trojan SS, XHTTP, mKCP, Mekya e ofuscação
* Credenciais, cotas de tráfego, expiração, redefinições periódicas, renovação automática e status online por cliente; também limites por entrada
* Links de compartilhamento, QR codes, importação de entradas a partir do YAML de um listener do Mihomo e redefinição de tráfego em massa

**Assinaturas**
* Página de assinatura no navegador com uso, cota, expiração, nós e importação para o Clash Caesar com um clique
* **Modelos de assinatura:** cada assinatura é um perfil completo de cliente, gerado a partir do modelo embutido (China / Rússia / Irã) ou do seu próprio modelo do Mihomo
* Perfis completos para Mihomo/Clash, Stash, sing-box, Surge, Surfboard, Loon, Quantumult X e Egern; listas de nós para V2Box, V2RayNG, Shadowrocket e outros clientes de links
* Token de assinatura por usuário e porta de assinatura dedicada opcional

**Configurações do Mihomo**
* Basics: modo de roteamento, versão de IP para conexões diretas, IPv6, TCP simultâneo, estatísticas, logs e roteamento básico
* Saídas (incluindo Snell, WireGuard e OpenVPN) e grupos de políticas, com formulário, YAML nativo e conversão de links
* Regras de roteamento avaliadas de cima para baixo, incluindo regras `AND`
* Saídas do Cloudflare WARP e saídas residenciais do VPNGate por país e provedor

**Multi-control**
* Rede P2P de painéis sem nó mestre, com identidades assinadas
* Sincronização criptografada de entradas entre painéis confiáveis
* Agregação de assinaturas entre painéis
* Dispositivos cliente: o [Clash Caesar](https://github.com/RomanovCaesar/clash-caesar) importa o arquivo de rede, entra uma vez com a conta de administrador do painel e então vê todos os painéis e obtém a assinatura de qualquer painel ou a combinada por qualquer painel alcançável

## Capturas de tela

Interface atual do UnifiedProxy com dados de demonstração em chinês. Todas as paletas oferecem os modos claro e escuro.

<table>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/inbounds.png" alt="Entradas"><br><sub>Entradas</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/client-edit.png" alt="Cota e validade"><br><sub>Cota e validade</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/outbounds.png" alt="Saídas"><br><sub>Saídas</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/routing.png" alt="Rotas"><br><sub>Rotas</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/xray-import.png" alt="Importação Xray / 3x-ui"><br><sub>Importação Xray / 3x-ui</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/fleet.png" alt="Frota"><br><sub>Frota</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/login.png" alt="Login"><br><sub>Login</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/subscription-mobile.png" alt="Assinatura móvel"><br><sub>Assinatura móvel</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/t0-light.png" alt="T0 light"><br><sub>T0 light</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/t0-dark.png" alt="T0 dark"><br><sub>T0 dark</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/t1-light.png" alt="T1 light"><br><sub>T1 light</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/t1-dark.png" alt="T1 dark"><br><sub>T1 dark</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/t2-light.png" alt="T2 light"><br><sub>T2 light</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/t2-dark.png" alt="T2 dark"><br><sub>T2 dark</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/t3-light.png" alt="T3 light"><br><sub>T3 light</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/t3-dark.png" alt="T3 dark"><br><sub>T3 dark</sub></td></tr>
  <tr><td width="50%"><img src="./media/screenshots/unifiedproxy/t4-light.png" alt="T4 light"><br><sub>T4 light</sub></td><td width="50%"><img src="./media/screenshots/unifiedproxy/t4-dark.png" alt="T4 dark"><br><sub>T4 dark</sub></td></tr>
</table>

## Instalação no Linux

Execute como `root` em um Linux com systemd ou OpenRC:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/HlONGlin/UnifiedProxy/main/install.sh)
```

O instalador escolhe uma porta livre de cinco dígitos e gera um caminho de painel de 16 caracteres, um usuário de 14 caracteres e uma senha de 16 caracteres. No final, mostra uma única vez a URL de acesso e as credenciais. Você também pode defini-los:

```bash
bash <(curl -Ls https://raw.githubusercontent.com/HlONGlin/UnifiedProxy/main/install.sh) \
  --port 34567 \
  --path MyPanelA1b2C3d4 \
  --username AdminUserA1234 \
  --password 'ChangeThisA12345'
```

O UnifiedProxy é instalado em `/usr/local/m-ui` e guarda os dados em `/usr/local/m-ui/data`. O instalador também baixa um núcleo do Mihomo para a arquitetura atual; um núcleo já instalado é mantido, a menos que você use `--update-mihomo`.

`m-ui update` atualiza o UnifiedProxy mantendo todas as configurações. Antes de atualizar, salva um snapshot do diretório de dados em `data/backups/` e mantém os 3 mais recentes.

Cada versão inclui `m-ui-linux-<architecture>.tar.gz` para `386`, `amd64`, `arm64`, `armv5`, `armv6`, `armv7` e `s390x`, além de `m-ui-windows-amd64.zip`.

## Docker

A imagem Docker é para Linux amd64 e inclui o UnifiedProxy e um núcleo do Mihomo de versão fixa e verificado. O Compose usa a rede do host, então as portas das entradas adicionadas no painel funcionam sem mapeamentos extras.

```bash
git clone https://github.com/HlONGlin/UnifiedProxy.git
cd UnifiedProxy
cp .env.example .env
docker compose up -d --build
docker compose logs m-ui
```

Compile a partir deste repositório. Versões com tags podem publicar `ghcr.io/hlonglin/unifiedproxy:<tag>` e `latest`; use `docker compose pull` quando a imagem correspondente do UnifiedProxy estiver disponível. `MUI_PASSWORD` / `MUI_PATH` vazios são gerados no primeiro início e exibidos uma vez no log. Por compatibilidade, mantenha o serviço `m-ui` e os volumes `m-ui-data`, `m-ui-core` ao atualizar.

## Certificados TLS

A primeira instalação interativa oferece um certificado Let's Encrypt para domínio, um certificado Let's Encrypt de curta duração para IPv4, um certificado existente ou seguir sem TLS. Depois de instalar, execute:

```bash
m-ui ssl
```

O menu permite emitir certificados para domínios e IPs, obter certificados de domínio e curinga pela API DNS da Cloudflare quando a porta 80 não está disponível, renovar, revogar, listar e aplicar um certificado existente. Se o arquivo do certificado tiver apenas o certificado final, o UnifiedProxy completa a cadeia automaticamente, porque clientes como Mihomo Party e FlClash o rejeitam.

## Comandos de gerenciamento

```text
m-ui                  interactive menu
m-ui start | stop | restart | status | logs
m-ui settings | configure | set-port
m-ui reset-credentials | reset-path
m-ui update | update-menu | ssl
m-ui enable | disable | clear-logs | uninstall
m-ui --lang <code>    menu language, e.g. m-ui --lang pt-br
```

O menu fala os mesmos 11 idiomas do painel e lembra a sua escolha. O instalador continua em inglês.

## Documentação

* [Implantação e migração](docs/unifiedproxy-migration.md)
* [Modelo e protocolo](docs/fleet-protocol.md)
* [Funções e compatibilidade](docs/design/legacy-acceptance.md)
* [Temas e referências](docs/design/p2-token-map.md)
* [Revisão P5](docs/design/p5-acceptance.md)
* [Versões](https://github.com/HlONGlin/UnifiedProxy/releases)

## Desenvolvimento

Compile, teste e visualize apenas no contêiner Linux orca-linux, em `/workspace`. Use Windows somente para edição e Git.

```bash
test "$PWD" = /workspace && test -f /.dockerenv
go build ./...
go vet ./...
go test ./...
```

`MUI_VERSION` / Git tag → `-X main.version=...`. [Fleet build / deployment](docs/unifiedproxy-migration.md) · [Design tokens](docs/design/p2-token-map.md).

## Agradecimentos

* [MetaCubeX/mihomo](https://github.com/MetaCubeX/mihomo): o núcleo proxy e o modelo de configuração
* [MHSanaei/3x-ui](https://github.com/MHSanaei/3x-ui): densidade da interface e fluxo de trabalho
* [Sub-Store](https://github.com/sub-store-org/Sub-Store): comportamento de referência dos formatos de assinatura
* m-ui (GPL-3.0), UnifiedProxy (MIT): [NOTICE](NOTICE), [MIT notice](licenses/UnifiedProxy-MIT.txt).
* [Lucide](https://lucide.dev): ISC; [icon attribution](web/licenses/LICENSE-Lucide.txt).
* Vercel / Claude (Anthropic) / Sentry / Supabase / Stripe: [palette references](docs/design/p2-token-map.md), via VoltAgent/awesome-design-md (MIT).

## Licença

O UnifiedProxy é distribuído sob a [GNU General Public License version 3](LICENSE). O código de conversão de links do Mihomo mantém sua atribuição GPL-3.0 em [`internal/mihomoconvert/NOTICE.md`](internal/mihomoconvert/NOTICE.md).
