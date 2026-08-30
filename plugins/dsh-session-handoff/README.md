# @ymm/dsh-session-handoff

会话收尾盘点插件：一轮对话收尾（agent 从 running 转 idle、没有排队工作了）时，
自动盘点项目 git 状态，把「收尾缺口 + 后续开发任务」写入 `<项目根>/.handoff/LATEST.md`，
让项目状态在会话之间不断档、不烂尾。`/handoff` 命令可随时手动触发，不受节流限制。

## 工作原理

1. 监听 `agent/status` 的 `running → idle` 转换（每轮对话收尾）。
2. 对会话 `header.cwd` 指向的项目跑一次只读 git survey（分支/未提交/领先落后/
   未推 commit/残留 worktree/未合并分支/stash），秒级完成。
3. 按仓库收尾协议生成任务清单：
   - P0 主分支上有未提交改动（仓库红线）
   - P1 提交或 stash / 推送远端 / 建上游
   - P2 反向同步落后提交 / 清理残留 worktree / 处理未合并分支
   - P3 stash 待处理
4. 报告写入 `.handoff/LATEST.md`，并自动把 `.handoff/` 加进 `.git/info/exclude`
   （不弄脏 git 状态、不改工作区文件）。下一个会话开工前先读它。

## 安装

```bash
dsh plugin add ./path/to/@ymm/dsh-session-handoff
```

`package.json` 已声明 `dsh.bundle.patch`，add 后自动进 layer stack。

## 配置（cordis.patch.yml 行 config）

| 字段 | 默认 | 说明 |
|---|---|---|
| `outputDir` | `.handoff` | 报告目录（相对项目根） |
| `fileName` | `LATEST.md` | 报告文件名 |
| `minIntervalMs` | `30000` | 同项目两次自动盘点的最小间隔；`/handoff` 不受限 |
| `mainBranch` | `main` | 主分支名，收尾判定基准 |
| `remote` | `gitea` | 远端名，推送建议使用 |
| `autoExclude` | `true` | 自动把 outputDir 写进 .git/info/exclude |

## 开发

```bash
# 依赖：宿主 @deepseek-ai/* 用 vendor 符号链接保证与目标宿主版本逐字对齐
ln -sfn ~/.dsh/vendor/dsh/node_modules/@deepseek-ai node_modules/@deepseek-ai
for p in cordis cosmokit schemastery; do ln -sfn ~/.dsh/vendor/dsh/node_modules/$p node_modules/$p; done
npm install --legacy-peer-deps   # 只装 typescript/vitest/@types/node

npx vitest run          # 130 用例
npm run build           # tsc -> lib/
bash ~/.agents/skills/dsh-plugin-dev/scripts/check.sh   # 交卷门禁
```

## 约定

- 插件只「盘点 + 排任务」，不动手改代码——修是下个会话的事。
- 报告开头一段「给下一个会话」：先读本文件与仓库 AGENTS.md，从 P0 开始。
- 报告由插件覆盖式生成；人工勾选会被下次盘点冲掉，处理完的任务直接删行。
