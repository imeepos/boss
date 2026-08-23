# 开发收口计划：基于差距审计与三个未完成 worktree

> 版本 V1.0｜基线：`docs/plan/three-year-gap-audit.md`、`docs/plan/gap-closure-development-plan.md`
> 目标：先收回并完成三个 worktree 的现有改动（CS/AR、补偿对账、税务支付），然后按优先级推进剩余可执行项；所有外部资质/真实环境阻塞必须明确登记，不伪造通过。

## 0. 当前状态盘点

### 0.1 主分支与 worktree

- 主分支：`main @ 2cd2af5`，工作树干净。
- 三个并行 worktree 都停在 `2cd2af5`，**均未提交、均未 push**：
  - `/private/tmp/boss-task1` → `feat/task1-csar-closure`（CS/AR）
  - `/private/tmp/boss-comp-recon` → `feat/compensation-recon-center`（异常补偿与对账）
  - `/Users/imeepos/ymm-001/boss-task3-tax` → `feat/task3-tax-payment-closure`（税务支付）

### 0.2 已知缺陷

1. **迁移号冲突**：三个 worktree 都生成了 `000128_*` 迁移，按 `make check` 的 D 项会被机械拦截。
   - 协调结果：CS/AR 保留 `000128`，补偿 → `000129`，税务 → `000130`。
   - 当前实际状态：CS/AR 仍 `000128`（正确）；补偿仍 `000128`（**未改**）；税务已 `000130`（正确）。
2. **测试/门禁缺失**：三个 worktree 都有代码改动，但没有任一 worktree 跑过全量 Go 测试和契约同步门禁。
3. **提交规范**：agent 未 commit，commit message 主体、why 描述、revert 独立性都未核对。
4. **push 缺失**：分支未推送到 `gitea` 远端，合并前必须先 push。
5. **主分支保护**：必须严格遵守 `worktree merge --ff-only → push → worktree remove → branch -d → push --delete` 的收尾顺序。

### 0.3 已知优化建议（来自开发收口计划与差距审计）

1. **CS/AR 业务闭环**：CS 写入链 + SLA/升级 + 客服工作台；AR 日账龄快照 + 自动催收 + 信用等级 + 承诺还款/核销 + 重试恢复一致性回放。
2. **异常补偿与对账**：统一责任队列、跨域对账、失败重试/回放、四码冲突 4 小时清零率。
3. **税务支付闭环**：外部税务适配器接口、签名/凭据、异步回执、重复/乱序幂等、单票失败回放、微信/支付宝自动对账。
4. **PORT/LOY/多语言**：PORT 客户门户前端完整旅程；LOY 生产异常闭环；user/worker 多语言资源。
5. **性能与 SLO**：100 VU 读 P95 574ms、登录 P95 858ms 超阈值问题。
6. **数据治理底座**：指标目录、血缘、质量责任闭环。
7. **AI 低风险辅助**：模型路由、限额、成本、评测、人工确认。
8. **预测维护 MVP**：故障预测、容量预测、维护优先级。
9. **规模复制准备**：tenant 模型、多时区、多币种、双区域复制。

## 1. 执行顺序（依赖型）

```
P0：迁移号修正 → 全量门禁 → commit → push
P1：合并三个分支到 main
P2：PORT/LOY 多语言 + 性能 SLO（外部依赖少）
P3：数据治理底座（先有指标才有 AI）
P4：AI 低风险辅助（要先治理）
P5：预测维护 + 规模复制（最后做）
```

外部资质或真实环境阻塞的（CN 乐企/PH BIR eIS 真实凭据、微信/支付宝真实资质、双区域复制验证），必须在 user 端确认后才进入对应任务。

## 2. P0 任务清单（立即执行）

### 2.1 补偿对账：迁移号修正

- `migrations/000128_compensation_tasks.up.sql` → `000129_compensation_tasks.up.sql`
- `migrations/000128_compensation_tasks.down.sql` → `000129_compensation_tasks.down.sql`
- 检查所有 Go 文件/SQL 注释是否引用 `000128`，必要时改为 `000129`

### 2.2 CS/AR：补齐全量门禁

- 跑 `/opt/homebrew/bin/go test ./...`
- 跑 `go run ./scripts/check-contract-sync -root .`
- 跑 `git diff --check`
- 三者全绿后再 `git add` + commit + push

### 2.3 补偿对账：补齐全量门禁

- 同上三项 + `git diff --check`
- 三者全绿后 commit + push

### 2.4 税务支付：补齐全量门禁

- 同上
- 迁移已是 `000130`，无需改号
- 三者全绿后 commit + push

### 2.5 合并到主分支（按启动顺序）

1. 先反向同步：`git merge main` 在各 worktree 内
2. 合并到主树：`git merge --ff-only <分支>`
3. 推送：`git push gitea main`
4. 清理：`git worktree remove`、`git branch -d`、`git push gitea --delete`

## 3. P1—P5 任务清单（按 P0 完成后再启动）

每条都按 P0 的同款流水线：独立 worktree → 全量门禁 → commit → push → ff-merge → 清理。**任何一项遇到外部依赖必须停下，登记并请 user 决策**。

## 4. 阻塞登记格式

每次发现阻塞，写一行到 `docs/notes/2026-08-26-gap-closure-blockers.md`：

```text
- [日期] [任务] [阻塞内容] [替代验收路径] [需 user 支持的内容]
```

## 5. 完成判定（每个 work 包）

- 全量 Go 测试 0 失败
- 契约同步门禁 0 红
- `git diff --check` 干净
- commit message 含 why 描述，可独立 revert
- 分支 push 到 `gitea` 远端
- 主分支 ff-merge 后再次推送
- worktree 和临时分支清理

## 6. 不在本次范围

按 `gap-closure-development-plan.md` 第 13 节明确不做：
- WHO 批发结算、RA 营收保障、FMS 防欺诈完整域、SET 结算互连
- 全面微服务化
- 大而全数据湖
- AI 直接执行收费、停机、资源释放、权限、合同
- 无人审批网络变更、全自动设备替换
- 复杂推荐算法、全渠道营销编排、复杂智能催收、独立呼叫中心

## 7. 下一轮入口

P0 完成后立即进入 P1。如果 P0 中任一 worktree 仍有问题，先报告 `git diff --stat` + `git status` + 全量门禁输出，由 user 决定是手动修正还是再派 agent。