# P6 合并收尾记录

2026-11-04（Asia/Shanghai）。用户已明确授权执行 P6。

## 本阶段改动

- 11 份 README 统一为 UnifiedProxy，采用五套配色 × 独立明暗，T0 黑白浅色默认。
  功能说明、Linux 安装、TLS、11 语言和管理命令保留，开发说明改为仅在
  orca-linux 容器 `/workspace` 构建、测试、预览。
- README 引用 18 张本轮真实浏览器演示截图，覆盖五配色的十种组合、入站、
  客户端、出站、路由、导入、舰队、登录及手机订阅页。不是设计稿或生产数据。
- 删除 README 中的旧三主题宣传、旧截图与 UnifiedProxy 产品仓库/Wiki/频道链接；
  Mihomo、Clash Caesar 等实际依赖的链接仍保留，GPL/NOTICE/MIT 归属不改写。
- Compose 镜像默认源、Docker OCI 标题/源码地址、Release 标题统一为 UnifiedProxy。
  CLI、压缩包名、服务名、数据目录和卷名统一切换为 UnifiedProxy 命名。
- 浏览器验收增加 `--output`，将 main 的 P6 复验结果单独保存，保留 P5 原始证据。
- Git 切换工作树暴露 Python shebang 被转为 CRLF 的问题，补充 `*.py eol=lf`，
  并保留真实 Mihomo 验证入口的可执行位，使再次检出后仍可启动。
  没有换用虚假内核或跳过原解析检查。
- Compose 原断言固定绑定 UnifiedProxy 上游，随本次明确的发行源改动改为精确匹配
  `ghcr.io/hlonglin/unifiedproxy`，并补查 OCI 品牌/源码地址。原平台、host 网络、
  数据卷、内核版本和 SHA256、发布权限与实际解析检查均继续执行。

## 历史提交保全

master 的两个独有提交 `72b0402「修正」`、`aa45551「存放」` 的完整文件清单
已在 [P6 合并前清单](p6-preflight.md) 列出；用户确认后先保留，再清理分支。

远程保留标签 `archive/legacy-master-20261005` 已推送并读取校验：

```text
annotated tag 1136f6709a38288ad2eb84109f5e1cec6a9a712c
peeled commit 72b0402dc7a729edcb276bc137705a41a5f0b476
```

该提交包含 `aa45551`，删除 master 分支不会使原始源码不可取回。

## 原工作树保护

`E:/脚本编写存放/git/MieruPilot` 原 README 的未提交公告已逐字节备份，
SHA256 与备份一致；另外保存 Git patch 和原始 Git blob。

```text
本地备份：E:/脚本编写存放/git/UnifiedProxy-P6-local-backup-20261104-110100
Git 保存：59eaefff381dbb2a0d69341173f2ca4ac11cc061
永久引用：refs/backup/p6-local-readme-20261104
```

原公告指向旧嵌套目录，未覆盖到新版 README；需要取回时可从上述备份恢复。
`docs/MERGER_NOTICE.md`、`.opencode/` 会话文件也已备份并保留原位；
原 `UnifiedProxy/` 是另一独立 Git 仓库，保留原位及原提交，不混入合并。

## main 复验与交付

main 从 `e5b5eac` 快进到包含 P0–P6 的成果；没有产生合并提交或强推 main。
全部复验均在 main 工作树对应的 orca-linux 容器 `/workspace` 执行：

```text
Go 1.27.0 linux/amd64
go build ./...: PASS
go vet ./...: PASS
go test ./...: PASS
go test -count=1 -json ./...: 539 passed, 6 skipped, 0 failed
oldest-backup real Mihomo transport: PASS
browser regression: 116 assertions, 57 screenshots, 0 measured contrast failures
native controls: 18 PASS
11 README files: local links / 18 assets / themes / brand / fences / RTL PASS
```

P6 浏览器复验为 T0 light 的独立与 Fleet 页面、实际操作和 native 控件；
完整十组合审查沿用 P5 的 543 张截图。摘要验证了 P5 原始证据哈希未改变。
6 个正常 skip 及替代执行范围见 [P5 报告](p5-acceptance.md)；没有新增跳过。

首次 main 复验暴露的换行与旧镜像断言问题修复后，已重新运行以上完整检查。
初次失败日志保留在 `dist/design-audit/p6-first-failure.log`，最终日志和 JSON
在 `dist/design-audit/p6-main.log`、`p6-tests.json`、`p6-native.log` 与 `p6/`。
机器摘要与证据 SHA256 见 [p6-results.json](p6-results.json)。

main 推送后核对远程提交与本地一致，master 的保留标签确认可取回，
再执行约定的分支清理。最终分支证明保存于本地 `dist/design-audit/p6-final-refs.txt`。
原工作树恢复到 main，原未跟踪文件和嵌套仓库保持原位。

未发布新 Release，也未操作生产服务。历史升级范围见 [历史兼容](legacy-acceptance.md)，
五主题和功能逐项证据见 [P5 报告](p5-acceptance.md)。

约定清理 `HlONGlin/main`、`master`、`codex/unifiedproxy-merge-20261103`。
本地 `key`、`upgrade/v1.6.0` 不在约定范围内，予以保留；因此不能声称本地只剩 main。
