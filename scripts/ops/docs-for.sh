#!/usr/bin/env bash
# docs-for: 任务→必读文档路由表(自进化 F2)。
# 用途:AI 会话/新人开工前先问它,拿「该读哪些文档 + 禁区红线」,不再靠猜翻文档。
# 用法: scripts/ops/docs-for.sh <关键词>
#   关键词支持英文/中文,如: field|字段  migrate|迁移  contract|契约
#   route|路由  accept|验收  pay|支付  deploy|部署  worktree|合并  tz|时区
#   perm|权限  android|安卓
# 无关键词列出全部任务类别。路径均相对仓库根,退出码恒 0(查询型命令)。
set -u

KEY="${1:-}"

show() { # show <类别> — heredoc 内 $ 保持字面,勿加 -e 展开用户输入
  cat
}

case "$KEY" in
  field|字段|col|列)
    show <<'EOF'
== 加字段/改列名 ==
必读:
  docs/contract/fields.md        页面列名-字段名-状态枚举三列对齐(全局强制)
  docs/contract/terms.md         术语与状态枚举权威
禁区:
  - 迁移号必须先查两处: ls migrations | tail + 各未合并分支 ls-tree migrations/
  - 响应 struct json tag 必须 lowerCamelCase(make check B 项拦截)
  - 契约变更必须 fields.md + openapi 同提交
EOF
    ;;
  migrate|迁移|migration)
    show <<'EOF'
== 加迁移 ==
必读:
  AGENTS.md 迁移编号规则段       防并行撞号
  migrations/ 目录命名惯例        up/down 成对
禁区:
  - 禁止不查跨分支占号就新增(D 项门禁红了必须让号,不许进 baseline)
  - 已进库迁移改名必须同步 UPDATE schema_migrations(edd0666 先例)
EOF
    ;;
  contract|契约|api|路由|route)
    show <<'EOF'
== 动契约/加路由 ==
必读:
  docs/contract/domain-map.md    能力域/Agent/包/页面四套命名对齐
  docs/contract/alignment-audit.md  D/A/B/E/G# finding 体系
  scripts/check-contract-sync/   A(实现→契约) A2(契约→实现+方法级) 门禁
禁区:
  - 路由必须先登记 openapi 再实现,或同提交(A 项拦截)
  - 方法级错位 A2 拦截;存量豁免见 check-contract-sync.baseline(加条目必须带整改方向)
EOF
    ;;
  accept|验收|测试|test|造数)
    show <<'EOF'
== 验收/E2E ==
必读:
  docs/ops/patrol-cron.md        孤儿巡检三通道分工
  docs/notes/adopted/2026-08-29-audit-closeout-rulings.md §7
禁区:
  - 造数统一 acc_ 前缀,收尾自清理,不过夜
  - 「孤儿/不一致」结论必须先 SQL 直查权威表,再接口复核读路径
EOF
    ;;
  pay|支付|stripe|钱包|缴费)
    show <<'EOF'
== 支付链路 ==
必读:
  docs/notes/adopted/2026-08-26-payment-hardening.md
  docs/notes/adopted/2026-09-05-stripe-config-backend-and-worker-charge.md
  docs/notes/adopted/2026-09-03-portal-wallet-balance-semantic.md
禁区:
  - Stripe 凭据只在 biz_params 密文,env 兜底已移除,勿再从 env 读
  - 充值余额=预存只进不出;合成客户(负数ID)充值拒绝 42200
  - 钱流对账用 scripts/ops/stripe-recon.sh(渠道 vs payment 表逐笔)
EOF
    ;;
  deploy|部署|102|上线)
    show <<'EOF'
== 部署 ==
必读:
  docs/ops/runbook.md            运维手册
  docs/ops/patrol-cron.md        102 cron 锚点总表
  ISSUE.md CI/部署段             历史怪象(SIGPIPE/容器名/空提交重触发)
禁区:
  - 部署后必须机器复验: scripts/ops/verify-deploy.sh(healthz commit+bundle 指纹)
  - 空提交重触发 deploy-102 无效(Classify diff 为空静默 skip),勿再踩
EOF
    ;;
  worktree|合并|分支|merge)
    show <<'EOF'
== worktree 协作 ==
必读:
  docs/notes/adopted/2026-08-22-worktree-merge-protocol.md
  docs/notes/adopted/2026-08-28-worktree-merge-circuit-breaker.md
禁区:
  - 收尾四步顺序: push gitea 分支 → 主树核对 cwd 后 ff-only → worktree remove → 删分支
  - worktree add 后必须 ls <dir>/.git 确认指针存在再写文件
  - ff-merge 失败严禁删 worktree,唯一动作 rebase 后重试
EOF
    ;;
  tz|时区|时间)
    show <<'EOF'
== 时间/时区 ==
必读:
  docs/notes/adopted/2026-08-21-business-timezone.md
禁区:
  - DB/容器 UTC,展示层转业务时区;自然日切日走 clock 包,勿手写时区换算
EOF
    ;;
  perm|权限|菜单|menu|角色)
    show <<'EOF'
== 权限/菜单 ==
必读:
  docs/notes/adopted/2026-08-22-custom-roles.md
  scripts/check-contract-sync/menuperm.go    E 项对账规则
禁区:
  - 新页面 key 必须配 menu:<key> 权限码迁移登记(E 项拦截)
  - API key 主体边界不跨端(worker key 打 admin 端必拒)
EOF
    ;;
  android|安卓|apk|发布)
    show <<'EOF'
== Android ==
必读:
  docs/notes/adopted/2026-08-27-user-android-release-signing.md
  docs/notes/adopted/2026-08-28-app-release-domain.md
  scripts/build-install-user-android.sh
禁区:
  - release keystore 不入库;APK 验签指纹不过不发
EOF
    ;;
  "")
    show <<'EOF'
docs-for: 任务关键词 → 必读文档 + 禁区。用法: scripts/ops/docs-for.sh <关键词>
类别: field|字段 · migrate|迁移 · contract|契约|api|路由 · accept|验收 ·
      pay|支付 · deploy|部署 · worktree|合并 · tz|时区 · perm|权限 · android
通用必读(任何任务):
  docs/notes/CURRENT.md          现行有效决策单页(先读这页)
  AGENTS.md                      收尾四步/提交纪律/300 行红线
EOF
    ;;
  *)
    echo "未知关键词: $KEY(试 docs-for.sh 无参数看类别表)"; exit 0 ;;
esac
