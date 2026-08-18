---
name: self-evolving
description: A self-evolving skill that grows through reflection on mistakes. Use for any task where you want to accumulate experience over time — this skill starts empty and learns from every failure. After each session, you must reflect on what went wrong and feed it back into the skill so the next session is stronger.
---

# Self-Evolving Skill

> 没有预设知识。每一条经验都来自真实踩坑。你的任务：每次用完后反思，把教训喂回来。

## ⚠️ 高频红线(重复犯过的坑,开工前必读)

> 完整计数台账:`references/recidivism.md`。每次反思同步 +1;≥2 次的坑必须登在这里。

1. **【已犯 3 次】编辑文件前必须用 read 工具读最新内容** —— bash 的 cat/sed 输出不算"已观察",edit 会直接拒绝;同一会话第二轮编辑凭记忆拼 old_string 必 not found。
2. **【已犯 2 次】禁止用原生 `<select>` 新增下拉** —— option 弹层系统渲染无法随主题定制,已两次被用户点名;一律用 `web/admin/src/components/Dropdown.tsx`。
3. **【已犯 2 次】edit 的 new_string 必须与 old_string 范围严格对称** —— 不顺手带函数头/注释(会重复定义),不做"只删换行"的 no-op(会并行致语法错);改完立刻 build 验证。
4. **【已犯 5 次】任务完成必须 git commit,`git status` 干净才算收尾** —— 门禁 = typecheck + test + build + commit;反思流程第 0 步先 `git status`,有产物先提交再反思。
5. **【已犯 2 次】禁止在总结里声称"已适配/已验证"而没有验证动作** —— 引用每个 CSS 令牌前 grep 它的定义；UI 交互必须在真实业务 DOM 中断言点击后的控件文本、筛选结果和 URL；没有双主题截图/build 或真实点击断言时一律明确写"未验证"。

## 1. 如何沉淀

**每次完成任务后，花 5 分钟做四件事(第 4 步更新累犯台账)：**

### 反思

问自己三个问题，答案写进 `notes.md`：

- 哪个坑浪费了最多时间？
- 这个 skill 有没有提前警告我？
- 重来一次我会怎么做？

### 喂回 skill

把反思结果变成 skill 的一部分，直接改文件：

| 你踩的坑 | 喂到哪里 |
|---|---|
| 某个错误反复出现、排查了很久 | `references/known-issues.md` —— 症状 → 原因 → 修法 |
| 学到一个通用经验、下次可以复用 | `references/lessons.md` —— 一句话经验 |
| 犯了某个规则、付出代价 | `references/red-lines.md` —— "禁止 X，因为 Y" |
| 发现一个排查技巧、工具、命令 | `references/techniques.md` —— 什么场景、怎么用 |
| 上游文档/API 有误、无法在 skill 内修复 | 项目根 `ISSUE.md` |
| 新经验已沉淀到 references 后 | 同步更新 `references/knowledge/` 分类索引，确保可被按场景检索到 |

### 更新累犯台账

同一坑再犯就给 `references/recidivism.md` 对应行 +1 并追加日期;新坑从 1 起登记。
次数 ≥2 的坑必须同步登记到本文件顶部"高频红线"区(保持排序:次数多的在前)。

### 写 ISSUE.md

skill 修不了的问题（上游文档错、API 缺、工具 bug）写进项目根 `ISSUE.md`：

- **信息不准**：哪个文档、哪句话、正确的是什么
- **信息缺失**：应该有什么但没有
- **其他**

每条必须具体："X 节说 Y，但实际是 Z。"

## 2. 目录规范

```
self-evolving/
├── SKILL.md              # 本文件：反思流程 + 目录说明（不存经验）
├── notes.md              # 每次任务后的反思记录（三个自问的答案），一次任务一段，只增不改
├── agents/
│   └── openai.yaml       # 宿主接入的界面元数据（display_name / short_description / default_prompt）
├── docs/                 # 项目事实手册：查证过才写、散在代码里难找的知识
│   └── boss-admin-web.md # boss admin 前端速查：冒烟账号、后端地址、门禁命令、主题/语言/localStorage 约定
├── references/           # 积累的经验（只增不改）
│   ├── recidivism.md     # 累犯台账：每个坑的重复次数+日期；≥2 次登到本文件顶部高频红线区
│   ├── lessons.md        # 通用经验，一句一条：当 X 发生时，修复是 Y
│   ├── known-issues.md   # 已知问题：症状 → 原因 → 修法
│   ├── red-lines.md      # 红线：禁止 X，因为 Y 发生过
│   ├── techniques.md     # 排查技巧、工具、命令：什么场景 → 怎么用
│   └── knowledge/        # 经验分类索引（按 前端/后端/实施 分门别类，原文不动）
│       ├── README.md     # 分类说明 + 速查统计
│       ├── 前端.md       # 前端经验索引
│       ├── 后端.md       # 后端经验索引
│       └── 实施.md       # 实施经验索引
└── scripts/
    └── cdp-capture.mjs   # 零依赖 CDP 工具（Node>=22 + 系统 Chrome）：网页截图 + console/网络采集，
                          #   --eval 自动填表登录，--logs 输出 console 报错与失败请求响应体
```

项目根 `ISSUE.md`（在本 skill 之外）：上游文档/API 有误、skill 内修不了的问题清单。

**references/ 写入原则：**

- 只增不改 —— 每条经验都是当时踩坑的现场记录
- 一条经验一行/一段，不要合并
- 经验过时了？在下面加一条新的纠正它，不要删旧的

## 3. 开工前必查（按场景检索）

**写代码前，先浏览 `references/knowledge/` 对应分类的标题，确认有没有"已知的坑"。**

- 写前端/UI/CSS/组件 → 看 `knowledge/前端.md`
- 写 Go/数据库/API → 看 `knowledge/后端.md`
- 部署/CI/环境配置 → 看 `knowledge/实施.md`
- 不确定 → 看 `knowledge/README.md` 速查统计表

每个分类文件末尾有"开工前 grep 关键词"，用这些词检索所有 references 文件。

## 4. 反馈优先级

最值钱的先写：

1. 静默失败，skill 没预警（浪费几小时）
2. 字段名、API 签名、配置 key 写错
3. 措辞模糊，把你引向错误方向
4. 遗漏的排查手法