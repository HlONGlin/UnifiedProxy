# P3 实现前雷同点清单

| 雷同点 | 替代方案 | 保留行为 |
|---|---|---|
| 侧栏主题首项/隐藏顶栏 | 常驻操作顶栏，侧栏只放功能 | 切主题/刷新/退出/折叠/移动抽屉 |
| 大型环形仪表 | 紧凑 stat-card + 横向 metric-bar | 原 ID/实时数值/CPU 历史 |
| 两列等权卡片堆叠 | 管理通栏 + 四列信息，hairline 分隔 | 原按钮与信息 |
| 玻璃层/彩色投影 | 不透明 surface + hairline | 弹窗/选项/分页/表单 |
| 居中动态品牌登录卡 | 左品牌右表单，移动端单列 | 登录/语言/主题/显示密码 |
| 胶囊 CTA | 每屏一个主 CTA，按钮8px | 原点击事件 |
| SF/Material | Lucide 1.5px rounded stroke | SF/AI 与 symbol ID API |

按登录→概览→列表→fleet→多控→订阅→mihomo的顺序共用组件，保留 m-ui/GPL 与 Lucide/ISC 归属。

## 阶段证据

- 外壳/主视觉/密度/CTA 已落地；P2 与 P3 十主题截图可直接对照（dist/design-audit/p2 与 p3）。
- sf-symbols/Material/三主题/liquid-* 已按清单移除。原 SF/AI 名称契约保存在 icon-contract.json；Lucide 线性路径全覆盖。
- 18 项真实浏览器交互通过，包括原生值/表单/Space、拖动只触发一次、取消不触发、拖动中禁用、RTL、分段键盘跳过禁用、动态控件和下拉 Escape/键盘。
- 原 tests/liquid-toggle.cjs 入口保留并转向上述等价行为检查；纯光学弹簧/镜片截图夹具说明已替换，功能断言未用 skip 掩盖。
- 所有功能面板继承统一密度、表面、四态；完整真实业务操作按 P1 的 F01–F36 在 P5 逐项回归，尚不声称全部业务验收。
