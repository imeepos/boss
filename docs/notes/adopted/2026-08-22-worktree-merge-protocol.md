# worktree 多分支合并协议（合并窗口纪律）

日期: 2026-08-22

## 决策

并行 worktree 分支合并回 main 采用"合并窗口"协议，冲突一律在 feature 侧消化，进 main 的合并保持零冲突：

1. **短命分支**：worktree 任务完成当天即合并，不过夜攒大分歧。
2. **合并前反向同步**：worktree 内先 `git merge main`（分支已推远端、历史不可重写时用 merge；本地私有分支可 rebase），解冲突并跑门禁（typecheck/test/build），再回主树合并，此时为快进。
3. **一次只合一个，合完立刻同步其余**：每个分支合入 main 后，马上对其余每个活跃 worktree 执行 `git merge main`。冲突以单分支增量小批次暴露，不在最后合并时爆发累积冲突。
4. **中央登记文件 append-only + 独立小提交**：menu.def.ts / App.tsx / i18n types+三份 locale / fields.md 的注册类改动压成每处一两行的独立提交，冲突退化为 trivial；不埋进大 feature 提交。
5. **全局流水资源跨分支查号让号**：迁移号/权限码/路由码按 AGENTS.md"迁移编号规则"执行——`git for-each-ref` + `git ls-tree <分支> -- migrations/` 查未合并分支占号，`make check` D 项机械拦截，已进 main 者优先、后来者让号。
6. **收尾顺序（硬性，防代码丢失）**：
   ```
   ① push origin <分支>
   ② 主树 git merge --ff-only <分支>   # 失败 → 回 worktree rebase/merge main 后重试
   ③ git worktree remove <目录>        # 仅 merge 成功后允许
   ④ git branch -d <分支> && git push origin --delete <分支>
   ```
   ff-merge 失败 ≠ commit 丢失（commit 在分支 ref 上安全）；失败时严禁删 worktree。

配套脚本 `scripts/worktree-sync.sh`：一键对全部活跃 worktree 执行 merge main + 门禁（协议第 3 步的机械化）。

## why

- 并行分支共享同一基线，任一分支合并后其余基线即过期；本仓库所有功能都触碰同一批中央登记文件（菜单/路由/i18n/迁移号），后合并者必然冲突。
- 冲突暴露越晚代价越大：等到任务完成才合并，面对的是多分支累积差异；合并后立刻同步，每次只解一个分支的增量。
- 2026-08-22 当日三次"ff-merge 失败 → 误删 worktree → commit 被 GC"事故（见 recidivism 台账），收尾顺序必须机械化、逐步可逆。
- 迁移号 000102/000103 当日连环撞号两次：worktree 只隔离文件，不隔离全局流水资源，查号必须跨分支。

## 放弃了什么

- **放弃"任务全部完成才一次性合并"**：大爆炸合并冲突面不可控，改为完成即合。
- **放弃在主树上直接解合并冲突**：冲突在 feature 侧消化，main 永远只接收干净快进/已验证合并；同时维持"禁止直接在主分支改代码"的既有裁定。
- **放弃默认 rebase 所有分支**：已推远端共享的分支 rebase 需 force push、破坏并行协作历史；仅本地私有分支允许 rebase。
- **不引入集中式锁服务/号段分配表**：分支数（≤5）与撞号频率不值得该复杂度；跨分支 ls-tree 查号 + make check 拦截足够。

## 关联

- AGENTS.md"使用worktree避免冲突/清理老分支/禁止主分支改代码"条目
- AGENTS.md"迁移编号规则（防并行撞号，2026-08-22 固化）"
- recidivism.md：ff-merge 误删 worktree（3 次）、迁移撞号（2 次）台账
- scripts/worktree-sync.sh（本决策配套）
