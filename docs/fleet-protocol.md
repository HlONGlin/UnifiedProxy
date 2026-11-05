# 模型映射与协议 v1

| UnifiedProxy | UnifiedProxy 合并模型 | 规则 |
| --- | --- | --- |
| Node.id/name/address | fleet.Node | legacy ID 稳定摘要生成新 ID，重复导入幂等 |
| Node.apiKey | 不迁移 | 单次令牌换发 Ed25519 |
| Mieru Port | mieru Inbound + Client | 凭据/TCP/UDP/配额保留；mita 退休 |
| AnyTLS Port | anytls Inbound + Client | anytls-go 退休；缺证书禁用并记录 |
| Naive Port | Caddy Sidecar | 域名、凭据、PEM 或原 ACME 策略 |
| ShadowTLS Port | shadow-tls + 可选 ssserver Sidecar | 托管后端只绑定回环 |
| TrafficLimit | Client.total，字节 | 本地执行；Sidecar 不计量 |
| TrafficUsed/TrafficRaw | Client.legacyCombined | 保留旧双向合计，用于配额和订阅总用量；不伪造上传/下载明细 |
| 全局旧订阅/token | 旧地址别名 + 新逐用户令牌 | 保留原 panelPath/sub?token= 地址，覆盖同次导入的全部节点 |
| UP Task | fleet.Task | 幂等键、租约、有效期、能力、回执 |

历史数据的迁移与外部条件以[历史兼容验收](design/legacy-acceptance.md)为准；
NaïveProxy 属于明确豁免项，不完整记录仅归档，不阻止其他支持协议导入。

desired.model 是原有 State，不是第二套 Port/UserCred 数据库；控制端额外保存
Node/Token/Task/Rollout/Audit，节点草稿在 fleet-nodes/<id>/。实际 YAML 用原生
渲染器生成。策略摘要排除 revision、保存时间、在线/运行计数和订阅 token。
wire YAML 为 epoch 时点的稳定策略投影，实际应用用节点时钟和保留计数重算，
防止到期边界造成两端伪差异。旧/新策略 ID 的并集界定托管对象，其他本地入站保留。

## 签名通道

已有 multi-control.json 时复用 Ed25519 私钥；新安装由同一身份初始化。
控制 RPC 扩展同一面板 HTTPS 服务，路径 /fleet/v1/；Agent 每 15 秒短 RPC
上报/领取，执行后立即继续。不引入 UP 原长轮询；poll.wait 必须为 0。

Envelope 字段：version=1、sender、audience、timestamp（Unix 秒）、
nonce（48 位十六进制）、payload（JSON）、signature（base64）。
签名将 version/sender/audience/timestamp/nonce/payload SHA256 按换行连接。
请求 audience 为 POST + 空格 + 路径，响应为 response:<请求 nonce>，sender=control。
允许 ±5 分钟偏差，控制端 nonce 先落盘再接受，重启后仍拒绝重放。
Agent 钉住可信控制公钥，拒绝重定向。请求最大 16 MiB，状态/备份最大 64 MiB。

POST /fleet/v1/enroll：token/publicKey/name，返回 nodeId/publicKey/protocol。
poll：status（能力/OS/架构/版本/revision/digest/运行/流量），返回 tasks。
report：状态；result：taskId/status/error/data。重绑/吊销后原公钥不能继续操作。

## 任务及应用

pending → leased → success/failed/unsupported，另有 expired/cancelled/superseded。
两分钟租约先持久化、最多三次投递。幂等键绑定类型及原始 payload，冲突拒绝；
节点持久回执使已完成副作用不重复。不确定的中断非配置操作返回失败，
要求核实后用新任务，不盲目重做。过期 ID 不复活；最新期望状态另建任务收敛。
未知能力等待升级并标 needsUpgrade。金丝雀成功后逐台释放，失败/过期停止。

任务：apply-state、collect-status、test-egress、kernel-switch、geo-update、
cert-renew、agent-upgrade、purge、core-control、traffic-reset、vpngate-install、
vpngate-reconnect。升级 pending-restart 只是本地状态，健康确认才报告成功。

应用：暂存 → 证书/原生模型核对 → Mihomo -t → 持久事务日志 → 保存 .previous →
原子替换 → 热重载/Sidecar 差量切换 → 健康检查 → 写 applied-state → 删除日志。
失败用独立有界上下文回滚，重启恢复未提交事务。相同内容重发不增加策略 revision，
但允许对失败/过期执行创建新任务。

## 面板 API

沿用管理员 session cookie。基础路径 PANEL_PATH/api/fleet：

| 方法/相对路径 | 内容 |
| --- | --- |
| GET / | publicKey/fingerprint/nodes/tasks |
| POST /nodes；GET/PUT /nodes/{id} | 新建/查看/修改 name,address |
| POST /tokens；/tokens/revoke | name、可选 nodeId；或 token |
| POST /nodes/{id}/revoke、publish | 吊销/发布草稿 |
| POST /nodes/{id}/model | 原生 State |
| POST /nodes/{id}/sidecars | 完整 Sidecar 数组 |
| POST /nodes/{id}/sidecar | kind/port/username/password/domain/handshake/certificateName 配方 |
| GET/POST /tasks | 列表；nodeId/kind/idempotencyKey/payload/ttlSeconds |
| GET/POST /batch | Rollout；nodes/action/kind/payload 或 inbound/quota/defaults |
| POST /import/merit | 原 merit.json；无关旧字段忽略 |
| GET /subscriptions、audit | per-user token；操作日志 |
| GET/POST /backup | 导出；action=restore/upload/download |
| GET /certificates | 去掉 PEM/私钥的列表 |
| POST /certificates/import、issue、renew、bind | 证书对象；binding=nodeId+inboundId 或 sidecarId |
| GET /update/check | 可选 role/platform，指定仓库 Release |
| GET/POST /update | 状态；url+sha256 |

原有编辑 API 用 X-Fleet-Node 定位节点，成功保存后发布。Agent 支持 join/break-glass，
独立面板另支持 sidecars、import/merit?nodeId=LEGACY_ID。
