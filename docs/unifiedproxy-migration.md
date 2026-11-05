# UnifiedProxy：运行与历史数据迁移指南

合并以 UnifiedProxy 为主体，保留 Mihomo 原生配置、Inbound/Client、订阅模板、3x-ui、
WARP/VPNGate、客户端设备和 11 种语言。外观现为五套配色（T0 黑白默认）× 独立明暗，
旧三主题偏好可迁移。新增舰队、签名任务、期望状态、
Sidecar、中央证书、WebDAV 和更新流程。合并产物遵循 GPL-3.0；UP 的 MIT 声明
保留于 NOTICE 和 licenses/。P6 收尾已将成果快进合并到
[main](https://github.com/HlONGlin/unifiedproxy/tree/main)，
旧 master 原始源码保留于 `archive/legacy-master-20261005` 标签。
main 合并不等于发布公开 Release 或切换生产服务，执行记录见 [P6 收尾](design/p6-completion.md)。

相关文档：[模型与协议](fleet-protocol.md)、[验收与发布](fleet-acceptance.md)。

## 三种模式

同一应用实现支持三种模式；control/node 是方便服务部署的薄入口：

| 模式 | 主程序 | 薄入口 |
| --- | --- | --- |
| 独立 | UnifiedProxy serve | node serve |
| 控制 | UnifiedProxy --control serve | control serve |
| Agent | UnifiedProxy --agent serve | 已接入的 node serve |

独立模式保留原有面板和 Multi-control。控制端不运行本机内核。Agent 保留锁定的
诊断面板及紧急接管入口，没有 agent.json 的节点不会自动入网。

```sh
go test ./...
go vet ./...
bash scripts/build-fleet.sh dev
# 本项目的构建/测试只在 orca-linux /workspace 执行。
```

## 接入

```sh
./control configure --data-dir /srv/control/data --username admin --password-stdin
./control serve --data-dir /srv/control/data
./control fleet-key --data-dir /srv/control/data
```

configure 从标准输入读取密码。默认只监听 127.0.0.1:2053；需要让其他服务器直连时，
使用 `--listen-ip 0.0.0.0` 并开放相应端口。主机可直接使用 HTTP，也可配置面板 TLS
或 HTTPS 反向代理。代理需要同时转发根路径 /fleet/v1/ 和面板 URI。
Agent 的控制 URL 只填源站，例如 `http://203.0.113.10:2053`，不附加管理员面板路径。
公网或局域网 HTTP 均无需证书；签名身份与防重放校验继续生效，但 HTTP 不加密传输。

在控制端主机网页的“舰队”页创建子节点后，页面会立即建立节点卡片并签发绑定该节点的
一次性接入令牌，同时显示一键接入命令。复制整条命令，在 Linux 子主机以 root 粘贴回车，
自动安装节点与缺失的 Mihomo、接入并启动 systemd 服务，无需另贴 token 或手动重启。
令牌有效期 15 分钟、仅用一次、主机磁盘只保存哈希。卡片可重新生成接入信息用于首次
接入失败或身份重绑。公钥应从可信页面或 fleet-key 获取，不在连接时自动信任陌生公钥。

```sh
# Docker、自定义路径或不使用自动脚本时的手动方式：
# 先安装 Mihomo 或通过节点原有面板安装核心：
./node configure --data-dir /srv/node/data --core-path /usr/local/bin/mihomo
./node agent join --data-dir /srv/node/data --control https://control.example --control-key PUBLIC_KEY --token-stdin
# 主机直接使用 HTTP 时，将上一行的 --control 换为 http://主机IP:端口
./node serve --data-dir /srv/node/data
```

上面的手动方式把令牌粘贴到标准输入并结束输入，接入后重启节点。网页生成的一键命令已含
token，可能出现在 shell 历史中，请勿分享；token 通过标准输入交给节点，不写入其进程参数
或服务文件。完整流程见[舰队管理使用指南](fleet-workspace.md)。接入后点击卡片的“管理节点”，或在主机网页顶部节点选择器选择子服务器。原有入站/Client/Mihomo
出站与路由/3x-ui 编辑界面随选择切换到该节点；节点草稿保存在控制端，
无需为子服务器单独打开网页。保存即发布。发布失败明确返回错误，草稿仍保留。
选择“主机全局默认配置”可编辑主机默认设置；该默认配置不会在主机运行代理内核。节点先用真实 Mihomo 验证，
再热重载、健康检查、提交 applied；失败和中断恢复 .previous。

## 所有权与应急

首次下发不会删除不属于该控制策略 ID 集合的本地入站。受管入站及其 Client
在节点只读，未托管入站仍可本地增删改；原始 YAML/全局重置等可能影响受管对象
的操作锁定。Agent/控制端停止 P2P 写入同步，独立模式保留原有 Multi-control。
本地继续执行配额、到期、周期重置、首次上线到期和自动续期，控制端离线不影响
已运行代理。新策略保留运行计数及未显式改变策略的本地续期结果。

```sh
node agent status --data-dir /srv/node/data
node agent break-glass --data-dir /srv/node/data
node agent resume --data-dir /srv/node/data
# resume 后重启节点
node agent relocate --data-dir /srv/node/data --control https://new-control.example
node agent leave --data-dir /srv/node/data
```

relocate 要求新地址证明持有原控制公钥。离网后在控制端吊销身份并重启；
重装使用节点专属重绑令牌，旧身份不能继续上报/领取/确认。purge 停止内核和
Sidecar、成功回执后吊销登记，不递归删除用户磁盘目录。

## merit.json 导入

在舰队的“备份与迁移 → 导入历史数据”选择原 merit.json，并点击“导入并转换”：Node → 稳定 ID 的舰队登记；Mieru/AnyTLS → 原生
listener + Client，保留用户名/密码；Naive/ShadowTLS → Sidecar。重复导入同一
legacy Node ID 不覆盖后续编辑。旧管理员、API Key、历史任务不成为新系统配置。

旧版双向合计流量独立保存为 legacyCombined，配额与月度重置继续计算该值；不伪造上传/下载明细。带 settings.subToken 的原 /<panelPath>/sub?token= 地址保留别名，涵盖同一次导入的全部节点。原始 JSON（含未知字段）进入受保护恢复档案，随 ZIP/Fleet 备份导出，普通 state API 和节点策略不暴露档案。
AnyTLS 的内联 PEM、SNI、指纹完整迁移；只有旧证书路径时禁用该入站并明确提示补传证书。ShadowTLS 默认旧版 aes-256-gcm，保留受管后端。NaïveProxy 记录不足时只归档，不阻断其余协议。
独立面板可导入一个源节点：填写 legacy nodeId（文件仅一个节点时留空），
保留无冲突本地入站和设置；需已安装实际 Mihomo 验证。多台旧节点应在控制端导入。

## Sidecar、订阅与证书

选节点后使用 Sidecar 配方或完整 JSON。ShadowTLS 配方自动分配回环 ss-rust
端口及随机 2022 密钥；Naive 选择中央证书并保存续期绑定。节点只运行白名单
二进制，不通过 shell 执行控制端数据，进程异常退出会重新启动。

Linux 自动安装沿用 UP 固定版本：Caddy forwardproxy v2.11.7-naive2、
ShadowTLS v0.2.25、ss-rust v1.18.2；需要 tar/xz。Caddy 自动包只覆盖 amd64，
其他平台可将兼容二进制放在 data/bin/。来源和下载后 SHA256 记录在 .source.json，
该记录不是发行者的独立签名。各 Caddy 隔离数据目录。Sidecar 明示不计量。

所有原有 listener 保留现有客户端导出与模板。ShadowTLS 以 SS + shadow-tls 插件
进入 Mihomo/Clash 导出；Naive 使用原生 naive+https 链接，不伪装为普通 HTTP
代理，也不声称不支持 Naive 的 Clash 内核能直连。其他格式跳过不能表达的协议。

证书页支持 PEM 导入、ACME 申请、续期、入站/Naive 绑定；先验证配对、域名、
有效期。列表不返回私钥；下发内容写入节点自己的版本目录。ACME 由控制端本地
acme.sh 执行，可设置 UNIFIEDPROXY_ACME_PATH；支持 HTTP-01、dns_cf 及 IP shortlived。
DNS 凭据留在控制端环境。每六小时检查自动续期；真实 CA/DNS 挑战须现场验收。

## 运维与管理 CLI

完整备份含控制身份、任务、期望配置、未发布节点草稿、模板、设备、面板设置和中央证书，必须当作
敏感文件保护。WebDAV 上传数据及 .sha256、恢复前核对，拒绝重定向；跨文件恢复
带持久恢复日志。跨机先离线恢复原 control-identity.json，禁止在线替换现有身份。
恢复后重启控制端重新加载模板和设备。

```sh
sudo bash scripts/install-fleet.sh control ./control-linux-amd64 SHA256
sudo bash scripts/install-fleet.sh node ./node-linux-amd64 SHA256
```

按安装器输出 configure 后启动服务；安装器设置 UNIFIEDPROXY_SERVICE_NAME。面板更新检查
Release 与平台校验和，再下载候选、验证可运行，交独立 systemd transient unit
停服/原子换文件/重启/健康检查，失败回退；Agent 的成功回执等健康确认后才发送。
安装器另外安装每分钟检查的 recovery timer。升级阶段落盘，worker 中断或重启后
超过十分钟仍未完成时，从独立 recovery 二进制恢复旧程序并发送失败回执。
它不把未验证的候选程序报为成功。内核和 Geo 数据任务也要求配置验证与健康检查，
失败恢复原文件；金丝雀只有收到成功回执才推进下一台。
更新检查默认指向 HlONGlin/UnifiedProxy；UNIFIEDPROXY_RELEASE_REPO 可覆盖为兼容发行仓库。
只有该仓库发布了对应角色/架构的资产和校验和才能升级；不会退回下载 UnifiedProxy 上游程序。
32 位 ARM 的自动检查选择兼容所有受支持 CPU 的 armv5 包；可显式指定 linux-armv6/armv7。
Windows 人工换包，Docker 同数据卷重建镜像；它们不声明 systemd 自动升级能力。
Dockerfile.fleet 与 docker-compose.fleet.yml 提供两种角色，Mihomo 使用 core 挂载。

管理员命令通过在线面板登录，不在第二进程并发写 fleet.json：

```sh
control fleet nodes --url https://control.example/PANEL_PATH --username admin --password-stdin
control fleet token --node NODE_ID --url https://control.example/PANEL_PATH --username admin --password-stdin
control fleet batch --json batch.json --url https://control.example/PANEL_PATH --username admin --password-stdin
control ssl import --json certificate.json --url https://control.example/PANEL_PATH --username admin --password-stdin
```

fleet 另支持 task/tasks、publish、revoke、subscriptions、audit、import、backup、
restore、update；ssl 支持 list/issue/renew/import/bind。JSON 对应协议文档的面板 API。
