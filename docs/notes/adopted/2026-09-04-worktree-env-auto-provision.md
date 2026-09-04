# worktree .env 自动接入(post-checkout 钩子 + example 兜底)

- 日期: 2026-09-04
- 状态: 已采纳

## 决策

新建 worktree 的 .env 接入用 git post-checkout 钩子自动完成:仓库内维护 .githooks/post-checkout,
核心逻辑在 scripts/env-provision.sh —— 目标目录已有 .env 绝不覆盖;否则优先复制主 worktree 的
真实 .env,主 worktree 无 .env 时由 .env.example 生成兜底模板。启用方式为一次性引导
sh scripts/env-hooks-init.sh(配置 core.hooksPath=.githooks,仓库级对本仓库全部 worktree 生效)。
引导说明与行为矩阵见 docs/ops/worktree-env.md,机械自测见 scripts/env-provision-selftest.sh。

## why

.env 含密钥不入版库,新建 worktree 是全新目录,环境变量不会随检出带过来;手工补拷贝易遗漏、
易过期,且并行 worktree 越多事故面越大。post-checkout 在 checkout/switch/worktree add 后都会
触发,是 git 原生能力,零第三方依赖即可把"新建 worktree 必有可用 .env"变成默认行为。

## 放弃了什么(被否决项)

- symlink 指向主 worktree .env:删除/移动主树即断链,且部分工具对 symlink 环境文件行为异常,
  误写真实值的风险面更大。
- 约定手工复制:正是本次要消灭的操作,不解决遗漏与过期。
- 第三方 dotenv/direnv 等工具:引入新运行时依赖,违反本仓库"不新增第三方工具"约束。
- 把钩子拷进 .git/hooks:该目录不入版库,clone 即失效,引导成本更高且脚本无法随仓库演进。

## 关联

- docs/ops/worktree-env.md(使用说明)
- docs/notes/adopted/2026-08-22-worktree-merge-protocol.md(worktree 工作流)
- docs/notes/adopted/2026-08-18-app-env-in-repo.md(注意区分:那是 deployments/app.env 部署密钥入库裁定,与本仓库本地 .env 无关)
