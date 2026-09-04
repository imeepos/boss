# worktree .env 自动接入(方案3)

新建 git worktree 时自动获得可用的 .env,无需手工复制或链接,杜绝切换 worktree 导致的环境变量缺失。

## 机制

- 钩子入口: .githooks/post-checkout —— git checkout / switch / worktree add 之后都会触发。
- 核心逻辑: scripts/env-provision.sh —— 目标目录已有 .env 时直接退出,绝不覆盖;
  否则优先复制主 worktree 的 .env(真实值,内容正确可直接使用);
  主 worktree 没有 .env 时(如全新 clone)用 .env.example 生成兜底模板。
- 真实值只在本机磁盘内复制; .env 已被 .gitignore 忽略,任何密钥不入库。
- 来源全部缺失时输出 [env-provision] ALERT 日志(grep env-provision 可查),不阻断检出。

## 一次性引导(全新 clone 后执行一次)

    sh scripts/env-hooks-init.sh

该命令做两件事:配置 core.hooksPath=.githooks(仓库级配置,对本仓库全部 worktree 生效),
并立即为脚本所在 worktree 补齐 .env。git 配置不随 clone 传播,因此每个 clone 需执行一次;
同一仓库的既有 worktree 无需重复执行。

## 行为矩阵

| 场景 | 结果 |
|---|---|
| worktree add 新 worktree(主 worktree 有 .env) | 自动复制真实 .env |
| worktree add 新 worktree(主 worktree 无 .env) | 由 .env.example 兜底生成 |
| 已有 .env 的 worktree 做任意检出 | .env 原样保留,不覆盖 |
| 主 worktree 已有 .env | 不动 |

## 验证

    sh scripts/env-provision-selftest.sh

沙箱仓库内机械自测,覆盖:新 worktree 自动复制 / example 兜底 / 既有 .env 不被覆盖 /
重复执行幂等 / 全新 clone 引导,全程不接触真实 .env。
