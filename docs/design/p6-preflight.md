# P6 合并前清单（审查存档）

2026-11-04，原工作分支 `codex/unifiedproxy-merge-20261103`。以下保留确认前的
审查清单；用户随后明确授权执行 P6，实际状态见 [P6 收尾记录](p6-completion.md)。

## 合并关系

已执行 `git fetch origin --prune`，核对：

| 引用 | 提交 | 关系 |
|---|---|---|
| main / origin/main | e5b5eac6cb9e3c6c62291495c8fd015e0f178f8a | 当前成果祖先，允许 fast-forward |
| origin/HlONGlin/main | afb444843d2d07abd2b79ecd851bab2d5ae1390c | 当前成果祖先 |
| origin/master | 72b0402dc7a729edcb276bc137705a41a5f0b476 | 另有以下 2 个独有提交 |

## master 独有提交完整文件清单

这些提交含最早实际业务源码，不应未经确认就断言没有保留价值。建议先创建并推送
保留标签（如 `archive/legacy-master-20261005`），确认标签可取回后再删 master 分支。
标签名为提议，尚未创建。

### 72b0402「修正」

2026-10-05 12:35:44 +08:00：

```text
M README.md
A one-click.sh
```

### aa45551「存放」

2026-10-05 10:32:00 +08:00：

```text
A .gitattributes
A .gitignore
A Makefile
A README.md
A build.ps1
A build.sh
A cmd/agent/main.go
A cmd/manager/main.go
A go.mod
A internal/agent/agent.go
A internal/agent/install.go
A internal/manager/agent_api.go
A internal/manager/handlers.go
A internal/manager/server.go
A internal/manager/web/index.html
A internal/mieru/mieru.go
A internal/mieru/mieru_test.go
A internal/model/model.go
A internal/store/store.go
```

## 待确认后执行的收尾

1. 统一 11 份 README 的 UnifiedProxy 品牌、仓库/安装链接、五主题说明和当前截图；
   保留 GPL/NOTICE、兼容 CLI、服务/数据目录、模块导入路径及真实客户端依赖的归属。
2. 安全快进 main，推送后核对远程 main 的提交与树；不强推。
3. main 目前占用在 `E:/脚本编写存放/git/MieruPilot`，该工作树有未提交 README、
   docs/MERGER_NOTICE.md、m-ui/ 和 .opencode/ 会话目录。禁止 reset/clean/覆盖它们。
   实施前先无损保存并核对这些变更，或采用用户确认的独立工作树交接。
4. 在 orca-linux 的 `/workspace` 对 main 复跑 build/vet/test 和关键浏览器/迁移验收。
5. 合并成功并确认保留标签后，删除本次约定的 HlONGlin/main、master、
   codex/unifiedproxy-merge-20261103 的存在的本地/远程分支。
6. 当前另有本地 `key` 和 `upgrade/v1.6.0`。它们不在约定的三个分支内，
   不擅自删除；因此不能伪造“只剩 main”的证明。若要求也删，需另外核对独有提交。

## 发布边界

P5 为 Linux 容器、回环真实多进程和本地服务验收。公网 CA/DNS、用户真实备份、
跨机器生产 WebDAV/服务升级与实际服务器部署不是这些本地通过结果的同义词。
详情见 [P5 审查报告](p5-acceptance.md) 与 [历史备份验收](legacy-acceptance.md)。
