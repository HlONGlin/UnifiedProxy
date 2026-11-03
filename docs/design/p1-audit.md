# P1 — 只读审计

2026-11-04。对象为 `6236d1c` 的前端和旧备份源代码。本阶段未修改产品文件。
机器可读的逐文件 ID、表单、按钮、页签、动态模板 ID、API 路径、变量及资源完整目录见
[p1-source-inventory.json](p1-source-inventory.json)，不是只列导航栏的抽样清单。

## M-UI 视觉身份与行为清单

| 文件/身份 | 作用 | 有功能行为 | 后续等价处理 |
|---|---|---|---|
| theme-shadcn.css | 默认黑白样式、选择态、表单、弹窗 | CSS 状态 | tokens 与统一组件；P2 并行保留，P3 移除 |
| theme-md3.css | MD3 色彩、密度、圆角、状态 | CSS 状态 | 同上 |
| subscription-themes.css | 公开订阅页的 shadcn/MD3 分支 | CSS 状态 | 与面板共用双维 tokens |
| index/login/subscription 内联 CSS | 基础结构、玻璃色值、Ultra Dark、动画 | 响应式、显示隐藏 | 保留结构/显示语义，换 token；Ultra Dark 迁移为 dark |
| liquid-glass.css | Apple 材质变量、玻璃叠层、背景、通用控件 | CSS 状态 | 统一基础组件 |
| liquid-optics.css/js | 鼠标光学高光、镜片与观察器 | 指针跟踪（装饰） | 清除装饰；原控件事件不变 |
| liquid-glass-gl.css/js | WebGL 壁纸折射 | 装饰渲染 | 移除装饰与壁纸 |
| liquid-button.css/js | 玻璃按钮、指针反馈、弹簧 | 点击/键盘反馈 | 原生 button 与四态；原 action 事件保留 |
| liquid-sheet.css | 弹窗/抽屉材质 | CSS 展开状态 | modal/sheet token 组件 |
| liquid-tabbar.css/js | 玻璃页签与移动选中指示 | tab active、键盘和 ink 同步 | 等价分段控件，保留 tab 事件 |
| liquid-toggle.css/js | checkbox 上的光学拖拽开关 | checkbox、键盘、禁用、change | 原生可访问开关，保留 checked/disabled/change |
| liquid-toolbar.css | 工具栏材质 | 无业务行为 | hairline 工具栏 |
| swiftui-components.js | 自定义 picker/combobox、键盘搜索与动态 toast | 有；不可整体删除 | 保留选项、搜索、键盘、change、disabled、toast API，换平面皮肤并更名 |
| sf-symbols.js | SF 形状 sprite 与 SF/AI 辅助函数 | 图标 API | 完整名称映射至 1.5px Lucide 线性图标 |
| icons-material.js | 替换 SF sprite 的 Material 形状 | 同名图标 | 移除视觉集，保留图标名称覆盖率 |
| icons-lucide.js | shadcn 线性 sprite | 同名 sprite | 提取统一图标，保留 Lucide 许可证 |
| fonts.css/fonts | Geist 与 Roboto 字体 | 文本渲染 | Inter/system 中文、JetBrains Mono；原备份字体路径仍可兼容 |
| media/m-ui_logo*.png | m-ui 品牌 | 无 | UnifiedProxy 自有 SVG 标识 |
| media/glass-wallpaper.svg | 玻璃背景 | 无 | 移除 |
| mac-traffic-lights、ambient/aurora、Apple 文案 | 视觉身份装饰 | 无 | 移除装饰；真实功能文案保留 |
| ui_theme.go/main.go Settings.UITheme | 三种外观设置、资源过滤与持久化 | 有；旧数据依赖 | 旧值输入仍接受并迁移，界面仅展示 t0–t4；不影响其他 Settings |

## 功能与页面对照表

ID 为后续实际回归记录的主键。除注明外入口均在 index.html。子控件详见 JSON 目录。

| ID | 页面/入口及操作范围 | JS/CSS 实现 |
|---|---|---|
| F01 | 登录：账号密码、显示密码、错误/禁用状态、11 语言、明暗、外观菜单、自定义 panel URI | login.html；ui_theme.go |
| F02 | 外壳：全部导航、折叠、移动抽屉、顶栏刷新、退出、回到顶部、安全提示关闭 | index.html；sf-symbols；三主题 |
| F03 | 概览：CPU/RAM/Swap/Disk、CPU 历史范围、运行信息、流量实时刷新 | index.html 的 renderOverview/renderCPUHistory |
| F04 | 概览：Mihomo 启停/重启/版本/下载/更新、geo 更新、端口/系统信息 | index.html；system.go |
| F05 | 入站：添加、编辑、克隆、开关、删除、协议/传输/TLS/Reality/额外安全模式 | index.html 的 inbound-form；mihomo-settings.css |
| F06 | 入站列表：搜索、状态筛选、分页、选择、自动刷新间隔 | index.html 的 inbounds-table/filterbar |
| F07 | 客户端：增删改、批量添加、流量/到期/设备/IP 限制、开关、重置 | index.html 的 client-form/client-table |
| F08 | 客户端分享：复制 URL、二维码、订阅、Clash/sing-box 导出 | index.html；qrious2.min.js；client_subscriptions.go |
| F09 | General Actions：导入、导出所有 URLs/订阅、配置测试、批量重置/清理 | index.html 的 runGeneralAction |
| F10 | 入站同步：目标面板矩阵、确认、进度、取消/重试 | inbound-sync.js/css |
| F11 | 跨订阅：拉取、缓存、进度、筛选、导出/复制/二维码 | cross-subscriptions.js/css |
| F12 | Xray/3x-ui：JSON/SQLite/link 导入、预览、选择、报告下载 | xray-import.js/css |
| F13 | 订阅模板：内置/自定义列表、查看、导入、启用、更新/删除 | sub-templates.js/css |
| F14 | Mihomo Basics：全部 details、基础/高级字段、保存、重启 | mihomo-settings.js/css；index.html |
| F15 | Mihomo Outbounds：创建/编辑/克隆/删除、form/YAML/link、测试、批量导入 | mihomo-settings.js/css |
| F16 | Mihomo Routing：增删改、排序、逻辑规则、MATCH 兜底 | mihomo-settings.js/css |
| F17 | WARP：信息、注册/登录、更新、解绑、出站配置 | mihomo-settings.js 的 warp-modal |
| F18 | VPNGate：列表、查询、安装状态、证书/协议选择、连接槽、重连、出站 | mihomo-settings.js 的 vpngate-modal |
| F19 | 在线连接：快照、筛选、关闭单项/全部、周期刷新 | index.html view-connections；从概览入口到达 |
| F20 | 日志：实时面板/Mihomo、刷新、筛选/查看 | index.html view-logs；从概览入口到达 |
| F21 | 面板设置 General：remark、地址/URI/session/分页/IPinfo/语言/外观 | index.html settings-form |
| F22 | 面板设置：通知、证书、外部流量、时区、备份/恢复、日志 | index.html settings-section；maintenance.go |
| F23 | Authentication：账号密码、保存、登录限制、重启 | index.html credentials-form |
| F24 | Subscription：全部订阅设置、路径、端口/TLS、格式/品牌 | index.html 的动态 subscription tab |
| F25 | Multi-control：本机信息保存、令牌生成/禁用、导入导出网络文件 | multi-control.js/css |
| F26 | Multi-control：连接/断开、阻止/解封、同步、客户端设备撤销 | multi-control.js/css；mc-* modals |
| F27 | 客户端计划：流量/启停日程、时区和周期 | client-schedule.js；traffic_schedule.go |
| F28 | Fleet：节点登记/编辑、enroll/rebind/revoke、目标节点切换 | fleet.js；fleet_console.go |
| F29 | Fleet：模型草稿/发布、原生入站/客户端、sidecar JSON/配方 | fleet.js；fleet_runtime.go/fleet_sidecars.go |
| F30 | Fleet：任务、批量动作、能力门槛、实时状态/流量/回执 | fleet.js；fleet_batch.go |
| F31 | Fleet：中央证书导入/申请/续期/绑定 | fleet.js；fleet_certificates.go |
| F32 | Fleet：更新检查/升级、备份下载/恢复、WebDAV 上传/下载 | fleet.js；fleet_backup.go/fleet_upgrade.go |
| F33 | Fleet：旧备份导入、转换日志、聚合订阅 | fleet.js；fleet_import.go/fleet_subscription.go |
| F34 | Standalone/Agent：join、紧急接管、sidecar 保存、旧单节点导入 | fleet.js；fleet_standalone.go |
| F35 | 公开订阅页面：流量/到期、明暗、语言、协议/客户端链接、二维码/复制 | subscription.html；subscription.go |
| F36 | 通用交互：确认/提示框、select/combobox、toast、Escape/焦点/键盘 | dialog.js；swiftui-components.js；index.html |

## 历史备份盘点

| 来源 | 结构/实际源代码证据 | 当前缺口 |
|---|---|---|
| 最早有业务数据的 `aa45551`（其后的 `18224d0` 是修正提交）（2026-10-05） | store.persisted：admin/settings.subToken/nodes；Port 无 proxyType 时为 Mieru | 原始 Node remark/日期与 subToken 未迁移 |
| 后续 merit / UnifiedProxy 1.x | 上述增加 panelPath、egress、流量/限速/自动禁用、TLS/SNI、ShadowTLS 后端、审计 | 当前转换器只读部分字段；流量与旧订阅丢弃 |
| m-ui 备份 ZIP | manifest + state.json + peers/cross/templates 等文件 | 原路径保留并测试，不得用旧 JSON 导入覆盖 |
| 合并版 Fleet JSON | format=1 / panel / fleet / files / drafts | 现有事务恢复与回滚保留，新增兼容接入不得弱化校验 |

初始 `6bed6a3` 只有 .gitattributes，不是可产生备份的业务版本。迁移夹具以最早实际持久化结构为起点。
旧 Agent API key、待执行任务、进程运行标志不能成为新版有效身份/任务；保留管理员访问与 panel URI 的安全边界，明确列出重新绑定步骤。

基线检查见 `dist/design-audit/p0-baseline.log`。浏览器依赖仅在指定容器安装。
