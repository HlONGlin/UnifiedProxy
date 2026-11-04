# 验收记录与发布门槛

日期：2026-11-03，以下为视觉改造前的验收存档。后续成果由 P6 收尾到
[main](https://github.com/HlONGlin/UnifiedProxy/tree/main)，详见 [P6 记录](design/p6-completion.md)。
此记录不表示已发布公开合并 Release、归档旧远端或切换生产服务。
本地测试不等于公网多机生产验收。

## 本地已经验证

- Windows 完整 go test ./...、go vet ./...；脚本测试使用 Git Bash。
- Docker golang:1.27-bookworm 完整 Go 测试。
- Linux race detector：agent/control/fleet 全部测试和 app 的 Fleet 回归测试通过。
- 三个独立签名节点接入、领取、执行、重启去重；令牌/重放/过期、错误公钥、
  重绑后旧身份拒绝、三次投递、能力门控和金丝雀失败门控。
- 实际子进程内核夹具：坏配置不替换、重载失败回滚、旧服务健康；
  本地配额、未托管入站、计数和本地演进到期时间保留。
- Linux 实际 upstream Caddy/ShadowTLS/ss-rust：固定资产下载、Caddy validate、
  证书加载、正确凭据 CONNECT/目标转发、错误凭据拒绝；后端/ShadowTLS 监听与重启。
- Linux 三个真实 Agent/Mihomo 节点，使用控制端实际生成的聚合订阅：
  hysteria2、AnyTLS、Mieru、ShadowTLS 各三次，共 12 项 HTTPS 数据转发检查通过；
  revision/digest 全部收敛。停止控制端并重启全部 Agent 后，12 项仍通过。
  使用回环网络、测试 CA、真实固定版本二进制；结果在 dist/fleet-live/results.json。
- 真实 systemd 的一次性 Linux 容器：执行安装器、HTTPS API 正常升级、坏候选
  健康检查失败回滚、中断 worker 后由独立 recovery service 恢复、timer 活跃。
  候选版本和运行文件 SHA256 均核对；结果在 dist/fleet-systemd/results.json。
- WebDAV Basic 认证上传/下载恢复、SHA256 不符拒绝、完整草稿保留、
  损坏备份拒绝及跨文件恢复中断测试。这里的 WebDAV 是本地测试服务。
- 实际 Edge 无头浏览器 11 语言 × 三主题共 33 组合、词典完整性及脚本错误检查。
  390px 手机视口无整页横向溢出；本地截图/JSON 在 dist/fleet-browser/。
- Linux amd64/arm64、Windows amd64 的 control/node 构建。
- Dockerfile.fleet 的 control/node 两个镜像实际构建、启动输出版本；
  Compose 配置校验、Release 工作流 actionlint 检查。生成物不提交。

```sh
go test ./...
go vet ./...
node --check web/fleet.js
bash scripts/acceptance/run-local.sh
# Linux，安装 tar/xz，会下载固定上游资产：
MUI_FLEET_INTEGRATION=1 go test ./internal/app -run TestFleetRealSidecarProcesses -count=1 -v
# Python playwright + 已安装 Chromium/Edge：
python scripts/acceptance/browser-smoke.py --binary dist/control.exe --browser /path/to/chromium
# Linux，预先构建对应角色并提供 Mihomo，需 openssl/curl/python3/tar/xz：
python3 scripts/acceptance/local-live.py --control dist/control-linux-amd64 --node dist/node-linux-amd64 --mihomo /path/to/mihomo --with-sidecars
# 只能在全新、可丢弃的 systemd 宿主机/容器中以 root 执行，会安装服务：
python3 scripts/acceptance/systemd-live.py --control dist/control-linux-amd64 --candidate dist/control-linux-candidate
```

systemd 测试候选需另行构建，例如使用
go build -ldflags '-X main.version=fleet-upgrade-test' -o dist/control-linux-candidate ./cmd/control。
Dockerfile.systemd 可构建具备所需工具的一次性验收镜像，
容器需独立的 cgroup namespace 和 systemd 运行权限，不挂载宿主机 cgroup。

## P0–P5 交付与现场门槛

| 阶段 | 已实现 | 现场仍需确认 |
| --- | --- | --- |
| P0 | m-ui 基座、GPL/MIT NOTICE、模型规格、迁移公告 | 正式仓库/名称、公开公告、旧版本冻结与归档时机 |
| P1 | 三模式、身份接入、舰队/流量视图、签名任务；三节点断控制端重启仍转发 | 三台公网服务器、生产 HTTPS/时钟、真实链路断网恢复 |
| P2 | 原生编辑、配额/到期、单写者、批量和回滚；三节点 hy2/AnyTLS/Mieru 订阅转发 | 全部原有 listener 的现场客户端兼容 |
| P3 | Sidecar、自动 SS 后端、中央证书与订阅注入；三节点 ShadowTLS 真实客户端转发 | 公网 Naive 专用客户端、公共 CA/DNS 与目标架构 |
| P4 | WebDAV 本地服务校验/恢复、Release 检查、真实 systemd 更新/回退/中断恢复、金丝雀 | 真实 WebDAV 跨机恢复、生产服务升级演练 |
| P5 | merit 导入/下载日志/映射、11 语言、文档、发布工作流 | 脱敏真实 merit 原节点逐项比对和公开发布 |

check-fleet.py 使用管理员登录检查至少三台在线、核心运行、revision/digest 收敛，
并对 manifest 中每个协议执行实际 SOCKS5/HTTPS CONNECT 数据请求。
它不把 TCP 开端口当成代理成功；Naive 的填充/客户端兼容仍需专用客户端，
ShadowTLS/Mieru/AnyTLS/hy2 等由真实客户端暴露本地 SOCKS5 后测试。

正式发布前必须补：公共 CA/DNS 挑战和续期、真实 WebDAV 备份跨机恢复原控制身份、
三台公网机器及生产服务升级演练、所有协议/导出格式的客户端支持范围。
Release 工作流和 Docker 推送是外部动作，本次未执行。
UP 先公告一个版本周期，保留最后 MIT tag；迁移验收通过后才归档远端。
