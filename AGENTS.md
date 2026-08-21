
# 无论任何任务 skill: `self-evolving` 总是需要遵守的
> 技能文件地址：`.agents/skills/self-evolving/SKILL.md`
> 安卓构建脚本： scripts/build-install-user-android.sh
> 102服务器信息： imeepos@192.168.0.102
> 使用worktree避免冲突，完成后合并到主分支，清理老分支，不用用户同意
> 写完的东西要立刻存档（commit）
> 禁止直接在主分支上修改任何代码

# 开工前必读（单一事实源，只读不猜）

实现任何功能前，必须先读以下契约文档，字段/状态/术语/域边界一律以此为准，与他处冲突时以契约文档为准：

- `docs/contract/terms.md` —— 订单 12 环节权威清单、状态枚举、关键术语
- `docs/contract/domain-map.md` —— 能力域/Agent/internal包/admin页面 四套命名对齐表
- `docs/contract/fields.md` —— 页面列名 ↔ 字段名 ↔ 状态枚举 三列对齐，命名规则全局强制

编码要求：
- 单个文件不要超过300行，推荐200行以内
- 单个函数/方法不要超过30行，推荐20行以内
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