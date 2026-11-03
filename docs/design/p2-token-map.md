# P2 — 旧变量到新 tokens 的实现前映射

参考对象：T0 黑白·墨—Vercel；T1 暖纸陶土—Claude/Anthropic；T2 夜紫荧光—Sentry；
T3 翡翠绿—Supabase；T4 靛紫—Stripe。色板由用户提供，经 VoltAgent/awesome-design-md（MIT）整理。
基础应用来自 m-ui（GPL-3.0），本次改造不能声称全部组件从零自研。

用户 2026-11-04 已授权连续推进至 P6；本表在产品修改前落盘，后续按此执行。

| 原变量/来源 | 新 token | 用途/迁移规则 |
|---|---|---|
| --bg、--page-bg、--sc-bg、MD3 background | --canvas | 页面画布 |
| --surface、--card-bg、--sc-card | --surface-1 | 一级表面；原 --surface 仅作兼容别名 |
| --surface-subtle、--input-bg、--sc-muted、--sc-sidebar | --surface-2 | 表单/次表面 |
| --surface-strong、--surface-glass-heavy | --surface-3 | 嵌套、轨道 |
| --surface-glass、--surface-modal | --surface-1 / --surface-2 | 不透明面板/浮层，无玻璃 |
| --text、--text-strong、--sc-fg | --ink | 正文与标题 |
| --muted、--sc-muted-fg | --ink-muted | 次级正文，实测 ≥4.5 |
| --muted-2、--text-secondary | --ink-subtle | 辅助正文；原淡灰仅保留为参考/装饰 token |
| --line、--stroke、--sc-border | --hairline | 分隔 |
| --line-strong、--stroke-2、--sc-input | --hairline-strong | 表单与较强边界 |
| --accent、--teal、--sc-primary、MD3 primary | --primary | 交互色；原品牌 accent 单独存 --reference-accent |
| --teal-2、--teal-3 | --primary-hover / --primary-active | 交互状态 |
| --teal-soft、--sc-accent | --primary-soft | 小面积选中底 |
| --sc-primary-fg、--on-accent | --on-primary | 主操作文字 |
| --sc-ring、旧焦点阴影 | --primary-focus | 2px、50% 透明 outline |
| --green、--orange、--red | --semantic-success/warning/danger | 只用于功能状态；文字另用达标的 *-ink |
| --blue、--indigo、--purple、--cyan | --chart-line | 旧装饰色移除；实际图表使用独立功能色 |
| 旧图表上/下载色 | --chart-line / --chart-line-2 | 主线功能蓝、次线灰虚线、填充 8%；不随 t0–t4 变 |
| --latency-* | --latency-good/mid/bad/dead | ≤100 / ≤250 / >250 / 无效；随明暗适配，五配色一致 |
| --shadow*、--glass-*、filter/backdrop | none | 保留交互/显示行为，移除光学装饰 |
| --radius、--radius-sm、--radius-xs | --rounded-lg/md/sm | 10/8/6；按钮固定 8 |
| --ease、spring 曲线、0.2s 等 | --easing-out/standard/in、--duration-* | 80/120/180/280ms；颜色切换专用 150ms |
| Geist/Roboto/Apple 字体 | --font-sans / --font-mono | Inter + 中文系统栈；JetBrains Mono + 等宽 fallback |
| body.dark、html.dark、ultra-dark | data-theme=light/dark | 旧本机偏好升级；ultra-dark 归并 dark |
| shadcn/md3/liquid 设置 | data-color-theme=t0…t4 | 旧值读入为 t0；新版值存同一 uiTheme 字段，原接口保留 |

## 色值和规则冲突的处理

- 主题配色文件优先于 DESIGN 的旧浅蓝/禁紫规则；t0–t4 优先于通用提示池里的 mono/clay 等示例 ID。
- 原始色板完整保留在 reference tokens。T0 原 mute #8f8f8f 不作小字正文，使用更深的墨色；T1 原 accent #cc785c + 白字不满足 4.5，主操作使用原版 active #a9583e 配白字，原 accent 仍保留为参考色，不伪称原色通过。
- 未提供的 surface 梯度按本色板的画布与表面混合推导；hover/active 从 accent 朝同色深浅推导。具体值见 themes.css 的十块颜色覆盖，结构只定义一次。
- T2 仅第一次选中默认暗色；每配色保存用户手动明暗，切换回来不覆盖。亮色手动选择始终可用。
- T2 不启用粉色点缀；荧光仅当前选中项的小标记（<2%），状态点使用功能色。
- T3 绿仅品牌、主 CTA、选中与焦点；图表/指标/状态仍用功能色。T0 除功能色以外保持黑白。
- 通用提示池 A2 的未闭合围栏、过时浅蓝说明和不存在的源存档不得当作项目事实；以本阶段文档和实际文件为准。

## 与后端的边界

只调整 appearance 的取值验证与页面资源注入，不改业务 API。旧 settings.uiTheme 三个值都能读取和保存，升级为 t0；
备份里的节点、凭据、订阅、证书、流量和路径不因主题迁移而变化。旧备份兼容增强单独提交与验收。

实测修正：T4 dark hover #695fff 配白字为 4.486:1，微调为 #655bf7（同色相、更低明度），保证交互态小字达到 4.5。
