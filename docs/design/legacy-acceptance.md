# 历史备份升级验收

2026-11-04。源头核对到 `aa45551`（2026-10-05「存放」），而不是其后的
`18224d0`「修正」。`6bed6a3` 只有 .gitattributes。未修改旧版本业务代码。

| 内容 | 验收与行为 |
|---|---|
| 最早原始 JSON | 缺 proxyType 默认 Mieru，稳定 ID、账号密码、端口、标签、日期、备注保存 |
| 中后期数据 | 配额、月度重置、autoDisabled、AnyTLS PEM/SNI/指纹、ShadowTLS 后端与旧 aes-256-gcm 默认值 |
| 流量 | 独立 legacyCombined，不捏造双向明细；用于配额、表格和订阅总使用量；月重置清本期、保留累计 |
| Agent 应用 | 真实 Apply 生命周期使用仓库进程夹具验证首次 seed、重复导入、状态上报、重置后发布不复活旧流量 |
| 旧订阅 | 原 panelPath/sub?token= 地址验证原始凭据与全部导入节点；错误 token 拒绝；重启/再备份/恢复保留 |
| 出站 | SOCKS5 凭据、规则排序、Mieru 入站作用范围、域名后缀/IP 条件、DIRECT/REJECT 保留；多候选转换为负载均衡组 |
| 安全归档 | 原始 JSON 含未知字段、API key、审计、旧身份等进入 protected recovery archive；不进入普通 state API 或 desired，不自动执行旧任务 |
| 失败处理 | 全批预校验；发布失败回滚全部节点、草稿、面板和别名；持久日志支持中断恢复 |
| 原入口 | /api/backup 同时识别旧 JSON/Fleet JSON/原 ZIP；ZIP v1/v2 严格校验保留 |

不能从备份凭空恢复的外部条件：旧证书只保存路径时需补传实际 PEM；旧 Agent 需要
使用新节点专用令牌重新绑定；节点需安装支持版本内核/sidecar；DNS/ACME 和真实服务器
部署需现场验证。NaïveProxy 为用户明确豁免项，配置不完整时仅归档，不阻止其余协议。
旧 bandwidthLimit 原实现只有存储和表单，没有执行代码，数值归档，不声称恢复了限速。
旧多候选 SOCKS5 是逐连接随机，新负载均衡组采用 round-robin；候选和认证均保留，
分配算法不同。此项已明确记录，不能称为逐字节无差异的路由行为。

证据：`dist/design-audit/legacy-regression.log`、`legacy-check.log`。测试含实际 HTTP 上传、
订阅 YAML、磁盘重启和备份恢复、失败回滚；不是仅测试 JSON 解码。进程夹具只证明
控制/应用契约，不能代替真实 Mihomo 网络代理的端到端证明。

P5 补充真实传输：固定 Mihomo v1.19.33，SHA256 与 Dockerfile 中的固定值一致，
最早备份结构从真实导入入口应用后，由另一个真实 Mihomo 客户端访问容器内 HTTP 服务。
Mieru TCP、Mieru UDP、AnyTLS、ShadowTLS + 自动 SS 后端均通过，保留原用户名/密码与
ShadowTLS aes-256-gcm 默认值。AnyTLS/ShadowTLS 仅测试夹具的自签证书放宽验证；
产品默认验证不变。证据：p5-transport.log。另运行原真实 sidecar 启动/重启检查，
见 p5-sidecars.log；这些不是公网部署或 CA/DNS 验收。
相同源 JSON 多次选不同节点导入时，归档合并节点关联而不重复源记录。
不完整 Naive 记录即使端口/凭据无效也仅归档，不阻止其他支持的节点。
