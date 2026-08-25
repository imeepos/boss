
# 无论任何任务 skill: `self-evolving` 总是需要遵守的
> 技能文件地址：`.agents/skills/self-evolving/SKILL.md`
> 安卓构建脚本： scripts/build-install-user-android.sh
> 102服务器信息： imeepos@192.168.0.102
> 后端对接一律用 102 部署环境： http://192.168.0.102:28080 （admin 前缀 /api/admin/v1；不要本机起服务、不要 mock 数据）
> 使用worktree避免冲突，完成后合并到主分支，清理老分支，不用用户同意
> 写完的东西要立刻存档（commit）
> 禁止直接在主分支上修改任何代码
> 端口冲突有可能其他人在工作，不要打扰，换个端口，结束时关闭清理

## worktree 合并协议（防并行分支冲突，2026-08-22 固化）

> 全文见 `docs/notes/adopted/2026-08-22-worktree-merge-protocol.md`，此处为执行要点。

- **短命分支**：任务完成当天即合并，不过夜；冲突一律在 feature 侧消化，进 main 的合并保持干净。
- **合并前反向同步**：worktree 内先 `git merge main`（本地私有分支可 rebase），解冲突跑门禁，再回主树合并。
- **一次只合一个，合完立刻同步其余**：可用 `scripts/worktree-sync.sh` 一键对全部活跃 worktree 执行 merge main + 门禁。
- **中央登记文件 append-only**：menu.def.ts / App.tsx / i18n types+locale / fields.md 的注册类改动压成独立小提交，不埋进大 feature 提交。
- **收尾四步（硬性，防代码丢失）**：① `git push gitea <分支>`（远端名是 gitea 不是 origin）→ ② 主树 `git merge --ff-only <分支>` → ③ `git worktree remove` → ④ `git branch -d` + `git push gitea --delete`。**执行前先核对 cwd 在主树（`pwd` + `git branch --show-current`）**：在 feature worktree 内执行②是 no-op 假成功（2026-08-28 实例），还会把本地 main 指针误推。
- **ff-merge 失败 ≠ commit 丢失**（commit 安全在分支 ref 上）：失败时严禁删 worktree，唯一动作是回 worktree `git rebase main` 后重试②。
- 误闯并行会话的 worktree 并编辑其未提交文件是事故（2026-08-22 用户点名）；发现半成品先 `git worktree list` 判断归属。

## 迁移编号规则（防并行撞号，2026-08-22 固化）
worktree 只隔离文件，不隔离全局共享的流水资源（迁移号/路由/权限码/契约章节），
各自"最大号+1"必撞（当日 000102/000103 连环撞号两次）。规则：
- 新增迁移前必查两处：`ls migrations | tail`（本树）+ `git for-each-ref refs/heads` 逐分支
  `git ls-tree <分支> -- migrations/`（未合并分支占号）。
- `make check` 的 check-contract-sync D 项机械拦截：同树撞号 + 跨未合并分支撞号，红了必须让号，
  不许进 baseline 豁免（历史撞号 000044/000095 为存量例外，已登记）。
- 让号规则：已合并进 main 者优先，后来者让号；未进任何库的迁移直接改名即可；
  已进库的改名必须同步 `UPDATE schema_migrations`（见 edd0666 先例）。

# 开工前必读（单一事实源，只读不猜）

实现任何功能前，必须先读以下契约文档，字段/状态/术语/域边界一律以此为准，与他处冲突时以契约文档为准：

- `docs/contract/terms.md` —— 订单 12 环节权威清单、状态枚举、关键术语
- `docs/contract/domain-map.md` —— 能力域/Agent/internal包/admin页面 四套命名对齐表
- `docs/contract/fields.md` —— 页面列名 ↔ 字段名 ↔ 状态枚举 三列对齐，命名规则全局强制

编码要求：
- 单个文件不要超过300行，推荐200行以内
- 单个函数/方法不要超过60行，推荐40行以内
- 注释不要废话/套话，函数名/方法名就是最好的注释
- 能复用就不要早轮子
- 不要使用emoji图标
- 读文件请使用Read工具
- 编辑文件前请务必先读取文件


## 决策记录制度（不可逆裁定当天过账）
- 任何不可逆/难逆决策（密钥方案、域分合、ID/序列方案、内置数据方式）落地当天写一篇 dated note 到 `docs/notes/adopted/`，记 why + 放弃了什么；索引见 docs/notes/README.md。
- 决策被推翻不改原文，加 `Amended` 指向新 note。
- finding 编号沿用三套既有体系，不许散落聊天记录：
  - 架构评审：`发现 N.M`（docs/architecture-review.md）
  - 契约对账：`D/A/B/E/G#`（docs/contract/alignment-audit.md）
  - 事后复盘：`docs/postmortem/000N-*`，随修复同提交，不事后补写。

## 提交纪律（revert 可行是硬约束）
- message 一律 `type(scope): subject`（feat/fix/refactor/docs/test/chore/style），正文写机理（why），不只写标题。
- 一次提交 = 一个可独立陈述的变更；feat 带测试、fix 带回归、契约变更带 fields.md 同步，配套随主变更同提交。
- 巨石提交仅限纯结构迁移（零行为变更）；行为变更禁止一锅端（不许 feat+fix+重构混装、不许多个不相关能力塞一个提交）。
- 判断标准：这个提交能否被单独 revert 而不伤邻居？不能就拆。

## 已知环境事实
brew 和 graphviz 都在 /opt/homebrew/bin