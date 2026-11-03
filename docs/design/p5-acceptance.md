# P5 五主题与历史升级验收

2026-11-04（Asia/Shanghai）。当前成果在 `codex/unifiedproxy-merge-20261103`；
P6 合并、删除分支和 README 收尾等待用户确认。

## 环境与验收口径

本轮所有依赖、编译、进程、浏览器和测试均在 `orca-linux-unifiedproxy` 容器的
`/workspace`，Go 1.27.0 linux/amd64。Windows 仅文件编辑/查看与 Git。
未以改断言、skip、零值或替身程序代替既有真实内核检查。

以下“通过”严格指所列范围：浏览器真实表单/操作，或 Go 实际调用处理器、
持久化、网络/进程测试。页面可见/截图不能等同于每个业务动作、外部服务实测。
公网 CA/DNS、生产 WARP/VPN、真实用户备份与生产 systemd 升级没有因此被认证。

## 本阶段修复

- 修正主操作文字、备注分段/分页、eyebrow、模板徽章、多控摘要的级联对比度；
  使用既有 tokens，未引入第六配色或改变 P3 结构。
- 登录图标描边统一 1.5px；公开订阅去掉旧 Apple/SF 字体与圆角变量覆盖，
  与面板共用 Inter/中文系统栈、JetBrains Mono fallback 和结构 tokens。
- 修复登录输入框被通用 padding 覆盖导致的图标/占位文字重叠；登录截图使用独立
  未认证会话，避免已有会话重定向后误截首页。十组合桌面/手机登录均已重新验收。
- 产品显示和更新默认来源为 HlONGlin/UnifiedProxy；发布校验拒绝跨仓库资产。
  保留兼容的模块、CLI/服务/数据路径；README 品牌更新留在 P6。
- 恢复选择器接受 JSON/ZIP；Fleet 展示合并历史流量；相同源归档合并节点关联。
  不完整的豁免 Naive 记录不会阻止其余支持节点导入。

## 十组合色彩审查

参考对象在 themes/components/workspace/polish/各功能 CSS 文件头，以及主题选择器
按品牌点名：T0 Vercel、T1 Claude/Anthropic、T2 Sentry、T3 Supabase、T4 Stripe。
原值与可访问性派生依据见 [P2 映射](p2-token-map.md)，归属保留在 NOTICE。

表格数值来自真实页面 `getComputedStyle` 的前景与逐层 alpha 合成背景，
相对亮度换算；每组合跨登录、管理页、弹窗、公开订阅和 Fleet。
忽略隐藏、禁用文字与被活动弹窗遮挡的底页；不是 OCR 或第三方认证工具。
不包含图标、占位文字、全部 hover/active 组合的全面可访问性认证。

| 配色 | 明暗 | 正文最低比值 | 大字最低比值 | 阈值 |
|---|---|---:|---:|---|
| T0 | light | 5.779:1 | 6.490:1 | 4.5 / 3 |
| T0 | dark | 7.130:1 | 11.070:1 | 4.5 / 3 |
| T1 | light | 5.057:1 | 5.371:1 | 4.5 / 3 |
| T1 | dark | 5.057:1 | 9.417:1 | 4.5 / 3 |
| T2 | light | 5.224:1 | 5.983:1 | 4.5 / 3 |
| T2 | dark | 5.224:1 | 10.336:1 | 4.5 / 3 |
| T3 | light | 5.779:1 | 6.218:1 | 4.5 / 3 |
| T3 | dark | 7.858:1 | 9.790:1 | 4.5 / 3 |
| T4 | light | 5.744:1 | 6.142:1 | 4.5 / 3 |
| T4 | dark | 4.581:1 | 7.048:1 | 4.5 / 3 |

以上为最终截图范围（含 Fleet 展开区域与真实未登录页）的测量结果。
同目录 `p5-results.json` 由 `summarize-final.py` 验证原始 JSON 生成，附原始证据 SHA256。

| 规则 | 审查 |
|---|---|
| T0 | 黑白 UI；允许独立功能图表/语义数据色；无旧品牌装饰色 |
| T1 | 陶土仅操作/选中/焦点，未用作整块卡片底色 |
| T2 | 荧光 #c2ef4e 仅选中小点，暗色近色像素最高约 0.0003%，亮色 0%，均 <2%；无发光描边 |
| T3 | 品牌绿未替代指标与图表，功能数据 tokens 在五套中一致 |
| T4 | 海军墨正文/表面与靛紫操作分层；dark 最低正文 4.581:1 |
| 共用结构 | 36px 行高、8px 控件圆角、120ms fast、mono 字体及功能图表/语义 tokens 在同明暗五套一致 |

像素占比为截图中距 token 各 RGB 通道 ≤5 的像素比例，不将抗锯齿像素误称为
精确语义面积。截图前等待字体/动画并禁用截图时动画，避免颜色过渡中间帧误判。
T2 浅色 `--highlight` 是紫色主操作色，不是荧光；手机登录页紫色按钮最高约
3.9841% 不适用荧光 <2% 的规则。摘要对两种明暗的全部 T2 截图均重新按原版
荧光 #c2ef4e 取样，不改产品色值，也不放宽荧光阈值。
旧三主题/liquid 视觉资源不再加载；原 logo 兼容资源、SF 图标调用名称和 GPL/ISC
许可证不等于旧视觉仍生效。Lucide sprite 覆盖既有调用名称。

## F01–F36 功能逐项记录

下表 Go 测试均实际执行，不是仅阅读源码。更广的每字段浏览器操作、第三方生产
行为不在某一项局部通过的含义内。`final-browser.py` 使用真实认证与持久化处理器；
其控制内核夹具和 WARP provider 为本地替身，真实内核传输另列。

| ID | 结果与实际证据 | 验证边界 |
|---|---|---|
| F01 | 通过：错误/正确登录、11 语言、重载保留明暗/配色；PanelPath/LoginLimit/Sessions 测试 | 未穷举每语言每表单文本 |
| F02 | 通过：导航、折叠/展开、390/480/768/1024px 检查、全局 reduced-motion；多视图截图 | 表格使用内部横滚，不强挤每一列 |
| F03 | 通过：实际 state/CPUHistory API，概览/历史弹窗与刷新 | 主机统计为容器而非生产服务器 |
| F04 | 通过：start/stop/restart/test 真实处理器；真实 Mihomo parser 与进程；发布来源/校验测试 | 浏览器生命周期使用进程夹具，未发布 Release |
| F05 | 通过：原生编辑表单保存、克隆、开关、删除、原凭据保留；各协议 render/validate 测试 | 实际网络转发协议见后文 |
| F06 | 通过：搜索命中/空态、disabled/all 筛选、自动刷新启停、分页翻页 | 分页用只读内存两行夹具，未伪造持久业务数据 |
| F07 | 通过：原生新增/批量客户端表单、API 编辑/删除/重置、开关；限额/到期测试 | 设备限制的外部客户端连接未逐一重放 |
| F08 | 通过：链接/二维码/订阅弹窗、实际 SubscriptionProducers/ClientConversion 测试 | 未启动所有第三方客户端 |
| F09 | 通过：导入/导出入口、真实配置测试、确认后两种批量流量重置 | 不把清理入口截图算已删除全部过期客户 |
| F10 | 通过：同步模态及真实 InboundSync 选客户合并、冲突、重放、失败回滚、幂等多目标任务测试 | 非公网跨机延迟验收 |
| F11 | 通过：跨订阅模态、真实 CrossSubscription 拉取/缓存/失败保留/URL 迁移测试 | 远端为本地 HTTP 测试服务 |
| F12 | 通过：导入入口；TestXrayConfigImport、XUIDatabaseImportV2/V3、SQLiteReader；真实 Mihomo 校验 | 未用用户真实 DB；不是所有 Xray 扩展均支持 |
| F13 | 通过：实际 preview/import API；SubTemplateManagerLifecycle、TemplateSubscriptionsForEveryClient | 内置远程模板更新服务未做生产可用性承诺 |
| F14 | 通过：Basics 子页与设置 PUT 读回；Defaults/Persistence/Compilation/NativeCoreConfig 测试 | 非每个输入项鼠标穷举 |
| F15 | 通过：Outbounds 模态；NativeRoundTrip/Groups/Draft/ShareLinks/真实 Mihomo 导入测试 | 不虚称每种出站均实连公网 |
| F16 | 通过：Routing 模态、设置规则读回；AndRules/PortRules/FinalMatch/引用校验测试 | 排序编译契约通过，非所有组合枚举 |
| F17 | 通过：WARP 模态；AccountLifecycleAndConcurrentCreate/Failure/WireGuardConfig/认证测试 | Cloudflare 为本地 provider，未注册生产账号 |
| F18 | 通过：VPNGate 模态；选择、CSV、安全下载、持久化、健康重连、DHCP/AppArmor 回滚测试 | 无真实 VPN 出口；AppArmor parser 缺失的单项明确 skip |
| F19 | 通过：连接快照；浏览器确认关闭全部→DELETE 处理器；视图截图 | 夹具连接为空，不声称逐条关闭活跃公网连接已测 |
| F20 | 通过：真实日志 API/视图、日志多语和脱敏测试 | 不把无错误页面当完整生产日志审计 |
| F21 | 通过：合法 remarkModel 改值→PUT→读回；行为参数/语言/主题迁移验证测试 | 故意非法模板仍被拒绝，未放宽校验 |
| F22 | 通过：ZIP 下载及 multipart 恢复、管理员不覆盖；BackupFormat/身份校验/TLSChain/外部流量测试 | TLS 路径不包含 PEM 时需现场补证书 |
| F23 | 通过：CredentialsEndpointChecksCurrentPairAndRotatesSession、登录限流/过期/重启信号测试 | 不在截图运行中更改真实用户账号 |
| F24 | 通过：订阅设置子页、真实 Token/路径/端口/TLS/ClientSuffix/公开响应测试 | 客户端支持范围保持原有格式约束 |
| F25 | 通过：浏览器生成令牌；真实 MeshAdminAPI/NetworkFile/CustomToken/Rotation 测试 | 不与 Fleet 节点身份混用 |
| F26 | 通过：连接/设备模态；真实 MeshDiscovery/Merge/Disconnect/ClientEnrollSingleUse/设备网络测试 | 三面板业务用本地 HTTP 测试服务 |
| F27 | 通过：ScheduledResetsPerClient、ClientScheduleSurvivesSaves、RenewClient、短月/时区/限制测试 | 时间由确定夹具驱动，不等待自然月变化 |
| F28 | 通过：真实节点增改/令牌吊销/节点吊销；三真实 Agent 接入、期望状态收敛 | 回环 TLS 与隔离测试 CA |
| F29 | 通过：原生模型发布/凭据保存、sidecars 保存；NativeApplyValidationRollbackAndAutonomy/真实 sidecar | 实测转发见后文，非仅端口通 |
| F30 | 通过：批量配额更新保留凭据；无能力节点任务拒绝；TaskLease/Canary/Retry/RestartReceipt 测试 | 离线未协商节点不能被误报执行成功 |
| F31 | 通过：证书展开截图；CentralCertificateSidecarBindingAndSecrecy 验证 PEM 配对/绑定/不泄露私钥 | 公共 ACME/DNS 挑战仍需部署环境验收 |
| F32 | 通过：Fleet backup、草稿保留/中断恢复、WebDAV 上传/校验拒绝、UpgradeRecovery、ReleaseOrigin 测试 | WebDAV 本地服务；本轮未重跑生产 systemd |
| F33 | 通过：最早/后续旧 JSON、原 ZIP/Fleet JSON、转换日志与旧订阅；真实旧凭据转发 | Naive 豁免；外部文件/旧身份限制见兼容文档 |
| F34 | 通过：独立页与本地 sidecar/导入区域；ManagedWriterLock/紧急接管；真实 Agent join/离线重启 | 未操作用户现有节点身份 |
| F35 | 通过：实际公开 token URL、使用量/QR/客户端链接页十主题及手机截图；SubscriptionProducers 测试 | 格式支持不等于每第三方客户端版本均验证 |
| F36 | 通过：18 项真实 native control 行为、确认/Escape、焦点/键盘、动态下拉及零 pageerror | 不将光学装饰已移除误认为功能删减 |

## 历史备份与真实传输

源头核对到最早产生业务数据的 `aa45551`（2026-10-05）。
原始与后续 JSON、ZIP v1/v2、Fleet JSON 导入、重复执行、磁盘重启、再备份恢复、
中断回滚、历史合计流量、订阅 token、PEM/SNI、ShadowTLS 后端及未知字段归档均有测试。
详见 [历史验收](legacy-acceptance.md)，不能把“已安全归档”说成“已自动激活旧身份”。

- `TestLegacyMihomoTransportFromOldestBackup`：两个实际 Mihomo 进程使用导入后的
  原始凭据完成 Mieru TCP、Mieru UDP、AnyTLS、ShadowTLS+SS 四条请求转发。
- `local-live.py --with-sidecars`：实际控制端与三个 Agent、实际聚合订阅生成客户端，
  hysteria2/AnyTLS/Mieru/ShadowTLS 共 12 条 HTTPS 检查全部通过；停止控制端、
  重启三个 Agent 后重复验证仍通过。
- `TestFleetRealSidecarProcesses`：真实固定版 Caddy/ShadowTLS/ss-rust 验证/启停/重启通过。

以上均为 Linux 回环网络；AnyTLS/ShadowTLS 的自签验证放宽只在专用测试内，
产品默认安全策略未放宽。Mihomo v1.19.33 的资产 SHA256 与 Dockerfile 固定值一致。
旧 SOCKS5 多候选从随机分配改为 round-robin，旧未执行 bandwidthLimit 只归档，
只存路径的证书需补 PEM、Agent 需重新绑定，均如实记录。NaïveProxy 按用户豁免。

## 执行结果与证据

```text
stage=p5-final cwd=/workspace system=Linux
go version go1.27.0 linux/amd64
go build ./...: PASS
go vet ./...: PASS
go test ./...: PASS
go test -count=1 -json ./...: 545 test/subtest events, 539 passed, 6 skipped, 0 failed
native controls: 18 PASS
browser: 143 assertions PASS; 543 unique screenshots across standalone, Fleet and operations
rendered text: 10 combinations; no measured contrast failures
real three-node data plane: 12 PASS, controller outage + agent restart PASS
```

6 个 skip：DesignAcceptanceServer 和 RealSidecarProcesses 已另行实际启动/执行；
InboundSync/MultiControl/WarpBrowserFixture 为待显式启用的服务器夹具（其业务 Go 测试
已经运行）；VPNGateAppArmorPolicyParses 缺外部 parser。本轮没有用跳过掩盖失败。

真实 parser 首次因缺 GeoIP MetaDB 自动下载/DNS 失败；补充下载并核对 SHA256 后，
wrapper 预置 geosite.dat、geoip.dat、geoip.metadb，保留原有基线断言和配置，复跑全绿。

机器摘要为 `p5-results.json`；本地原始文件在 `dist/design-audit/`（忽略生成物，不入 Git）：

```text
p5-final-check.log, p5-tests.json, p5-native.log
p5-transport.log, p5-sidecars.log, p5-live.log, p5-live/results.json
p5/browser-results.json, p5/fleet/browser-results.json
p5/operations/browser-results.json, p5/summary.json
p5/{t0-light..t4-dark}/, p5/fleet/{t0-light..t4-dark}/
```

可复跑入口：`check-stage.sh p5-final`、`final-browser.py`、`final-browser.py --fleet`、
`final-browser.py --operations-only`、`final-browser.py --login-only`、`native-controls.py`、
`summarize-final.py`。
真实 parser 环境变量 `MUI_TEST_CORE=/workspace/scripts/acceptance/mihomo-with-geodata.py`、
`UP_TEST_REAL_CORE=/workspace/dist/design-audit/mihomo`。

P6 待处理清单与 master 两个独有提交完整文件列表见 [合并前清单](p6-preflight.md)。
