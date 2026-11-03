# P4 质感微调

| 项目 | 页面 | 改前问题 | 改法 |
|---|---|---|---|
| 表面层级 | 面板/登录/订阅/下拉 | 局部浮层仍继承不一致表面 | surface-2 + hairline-strong，不透明，无阴影与玻璃 |
| 交互四态 | 表单/按钮/订阅复制/下拉 | 订阅复制仅鼠标；部分 hover 清除焦点 | 统一四态，2px focus 50%，复制元素 Enter/Space 可操作 |
| 节奏 | 列表/设置/指标/订阅 | 数字 tabular 覆盖不足、局部6/14px间距 | 既有4px网格 tokens、mono数字与固定行高；不移动功能入口 |
| 动效 | 所有主题/浮层 | all过渡、卡片 hover 位移、主题切换不限定属性 | 只过渡颜色/透明度；主题150ms只3种颜色；reduce-motion归零 |
| 数据呈现 | 概览/CPU历史 | ch-* 未覆盖，图表面积透明度未统一 | 共享图表色、灰虚线对比、8%面积；资源条和值同语义色 |

未改 P3 信息结构或五套颜色数值。新增 polish.css 在三种页面末尾加载，公开订阅
监听器只为该确定资源增加白名单，不开放管理 API。来源头注保留 Vercel、
Claude/Anthropic、Sentry、Supabase、Stripe 和 m-ui/GPL 归属。

证据：`dist/design-audit/p4-check.log` build/vet/test 全绿；`p4/` 六视图×十组合
60张截图、设置子页和零 pageerror；`native-controls.json` 18项真实交互通过。
这不是 P5 全功能验收，F01–F36 与所有附属界面/移动端在下一阶段验证。
