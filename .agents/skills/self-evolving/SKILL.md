---
name: self-evolving
description: "**[MUST LOAD FIRST]** A self-evolving skill that grows through reflection on mistakes. Use for any task where you want to accumulate experience over time — this skill starts empty and learns from every failure. After each session, you must reflect on what went wrong and feed it back into the skill so the next session is stronger. This skill MUST be loaded at the START of every session before any other work begins, regardless of the task at hand — its high-frequency red lines prevent costly repeated mistakes that waste hours.
---

# Self-Evolving Skill

> 没有预设知识。每一条经验都来自真实踩坑。你的任务：每次用完后反思，把教训喂回来。

## ⚠️ 高频红线(重复犯过的坑,开工前必读)

> 完整计数台账:`references/recidivism.md`。每次反思同步 +1;≥2 次的坑必须登在这里。

1. **【已犯 3 次】编辑文件前必须用 read 工具读最新内容** —— bash 的 cat/sed 输出不算"已观察",edit 会直接拒绝;同一会话第二轮编辑凭记忆拼 old_string 必 not found。
2. **【已犯 2 次】禁止用原生 `<select>` 新增下拉** —— option 弹层系统渲染无法随主题定制,已两次被用户点名;一律用 `web/admin/src/components/Dropdown.tsx`。
3. **【已犯 2 次】edit 的 new_string 必须与 old_string 范围严格对称** —— 不顺手带函数头/注释(会重复定义),不做"只删换行"的 no-op(会并行致语法错);改完立刻 build 验证。
4. **【已犯 5 次】任务完成必须 git commit,`git status` 干净才算收尾** —— 门禁 = typecheck + test + build + commit;反思流程第 0 步先 `git status`,有产物先提交再反思。
5. **【已犯 3 次】禁止在总结里声称"已适配/已验证"而没有验证动作** —— 引用每个 CSS 令牌前 grep 它的定义；UI 交互必须在真实业务 DOM 中断言点击后的控件文本、筛选结果和 URL；没有双主题截图/build 或真实点击断言时一律明确写"未验证"。
6. **【已犯 1 次】禁止假设模型支持图像输入** —— Kimi-k3 不支持图像分析，需要图像分析时应使用专门的工具（如 cdp-capture.mjs + 代码审查）或明确说明"未验证"。
7. **【已犯 1 次】禁止在未检查环境依赖时使用工具** —— 使用 Playwright/Puppeteer 等工具前必须先检查是否已安装，避免运行时报错浪费时间。
8. **【已犯 1 次】浏览器自测调试用 cdp-capture.mjs，playwright 只用于项目 E2E 自动化脚本** —— cdp-capture 零依赖、截图+console/网络采集+自动填表，适合单次验证；playwright 只放 `e2e/` 目录做 CI 自动化冒烟，不做日常调试。混用时用户会再次点名纠正。

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
    ├── cdp-capture.mjs       # 零依赖 CDP 工具（Node>=22 + 系统 Chrome）：网页截图 + console/网络采集，
                              #   --eval 自动填表登录，--logs 输出 console 报错与失败请求响应体
    ├── gpt-image-generate.mjs # 零依赖 gpt-image-2 页面设计稿生成（Node>=22 + .env）
    └── .env                  # API 密钥（不要泄露！）

项目根 `ISSUE.md`（在本 skill 之外）：上游文档/API 有误、skill 内修不了的问题清单。

**references/ 写入原则：**

- 只增不改 —— 每条经验都是当时踩坑的现场记录
- 一条经验一行/一段，不要合并
- 经验过时了？在下面加一条新的纠正它，不要删旧的

## 3. 浏览器截图与 UI 调试（必会工具）

**首选 `cdp-capture.mjs`**，零依赖、功能最全。另一个 `browser-test/screenshot.mjs` 是 playwright 方案，功能弱且要下 ~100MB chromium，不要再用。

### 何时用

- 验证 UI 改动效果（双主题、响应式）
- 排查前端报错（console / 网络失败）
- 需要登录后才能看到的页面

### 用法

```bash
# 基础截图
node .agents/skills/self-evolving/scripts/cdp-capture.mjs <url> <out.png>

# 截图 + 采集 console 报错和失败请求（排查首选）
node .agents/skills/self-evolving/scripts/cdp-capture.mjs <url> <out.png> --logs out.json

# 截图 + 自动填表登录（可重复多次 --eval）
node .agents/skills/self-evolving/scripts/cdp-capture.mjs <url> <out.png> \
  --eval "document.querySelector('#username').value='admin'" \
  --eval "document.querySelector('#password').value='123456'" \
  --eval "document.querySelector('button[type=submit]').click()"

# 自定义视口和等待时间
node .agents/skills/self-evolving/scripts/cdp-capture.mjs <url> <out.png> --width 1440 --height 900 --settle 3000
```

### 截图后

用 `read_image` 工具读取截图文件，分析布局、样式、文字。

### 依赖

- Node >= 22
- macOS 系统 Chrome（`/Applications/Google Chrome.app`）
- 零 npm 依赖

## 4. 页面设计稿生成（gpt-image-2）

**用 `gpt-image-generate.mjs` 调用 gpt-image-2 生成页面设计稿。** API key 在同目录 `.env`，不要泄露。

### 何时用

- 需求讨论阶段，快速出页面视觉稿
- 对比多个设计方案（`--n 2` 一次出两张）
- 给前端开发做参考图

### 用法

```bash
# 基础生成（横版，适合仪表盘/列表页）
node .agents/skills/self-evolving/scripts/gpt-image-generate.mjs \
  -p "BOSS系统仪表盘，深色主题，左侧导航栏，顶部4个数据卡片，下方订单列表" \
  -o ./designs/dashboard.png

# 竖版（适合登录页/表单页）
node .agents/skills/self-evolving/scripts/gpt-image-generate.mjs \
  -p "登录页面，居中登录卡片，账号密码输入框，蓝色登录按钮" \
  -o ./designs/login.png --size 1024x1536

# 生成两张方案对比
node .agents/skills/self-evolving/scripts/gpt-image-generate.mjs \
  -p "订单详情页面，包含状态时间线、商品列表、费用明细" \
  -o ./designs/order-detail.png --n 2
```

### 参数

| 参数 | 默认值 | 说明 |
|---|---|---|
| `--prompt / -p` | （必填） | 图片描述，越详细效果越好 |
| `--out / -o` | `./output.png` | 输出路径 |
| `--size` | `1536x1024` | 横版；竖版用 `1024x1536`；方形用 `1024x1024` |
| `--quality` | `auto` | `low`/`medium`/`high`/`auto`，high 很慢 |
| `--n` | `1` | 生成张数 |
| `--style` | `vivid` | `vivid`（鲜艳）或 `natural`（自然） |

### 生成后

用 `read_image` 读取生成的图片，分析设计细节，作为前端开发的参考。

### Prompt 技巧

- 用英文 prompt 效果更稳定，中文也可用
- 描述具体：主题色、布局结构、组件类型、文字内容
- 不需要写"高清"、"4K"等词，gpt-image-2 默认质量足够
- 生成的是设计参考图，不是可直接使用的代码

### 依赖

- Node >= 22
- 同目录 `.env` 中的 `OPENAI_API_KEY` 和 `OPENAI_BASE_URL`
- 零 npm 依赖

## 5. shadcn/ui 组件安装

本项目使用定制设计系统（`tokens.css` + `[data-theme]`），官方 shadcn CLI 生成的组件默认引用 `hsl(var(--primary))` 等标准变量，与本项目 `--color-brand-*`/`--shell-*` 令牌体系不兼容。安装 shadcn-style 组件有两种方式。

### 方式一：官方 CLI（可能不可用）

```bash
# 安装单个组件（自动确认 + 覆盖已有文件）
npx shadcn@latest add button -y --overwrite

# 安装多个组件
npx shadcn@latest add button card input -y --overwrite
```

**已知问题：** 当前环境下官方 CLI 可能因 `ERR_PACKAGE_PATH_NOT_EXPORTED zod/v3` 报错无法启动。若超时或报错，直接走方式二。

CLI 成功跑通后仍需手动修改：组件内的 `hsl(var(--*))` 引用需替换为项目实际令牌（如 `--color-brand-bg`、`--shell-card-bg` 等），并使用 `forwardRef` + `cn()` 包裹以兼容多主题。

### 方式二：本地脚本（推荐）

```bash
# 安装组件（自动解析依赖、重写 import、安装 npm 依赖）
node .agents/skills/self-evolving/scripts/shadcn.mjs add button

# 安装多个组件 + 覆盖已有文件
node .agents/skills/self-evolving/scripts/shadcn.mjs add button card --force

# 仅预览不写入
node .agents/skills/self-evolving/scripts/shadcn.mjs add button --dry-run

# 列出 registry 全部可用组件
node .agents/skills/self-evolving/scripts/shadcn.mjs list

# 初始化 cn() 工具函数（src/lib/cn.ts）
node .agents/skills/self-evolving/scripts/shadcn.mjs init
```

| 参数 | 说明 |
|---|---|
| `--root=<dir>` | 项目根目录，默认 CWD |
| `--style=<name>` | registry 风格，默认读 `components.json` 的 `style` 字段 |
| `--install=<tool>` | 包管理器：`pnpm`/`npm`/`yarn`/`bun`，默认 `pnpm` |
| `--force` | 覆盖已有组件文件（不加则跳过已存在文件） |
| `--skip-tokens` | 不合并 cssVars 到 `src/styles.css` |
| `--dry-run` | 只打印计划动作，不实际写入 |

### 安装后必做

1. 检查组件 CSS 变量引用：`grep -rn -- "var(--" src/components/ui/<name>/` 确认每个令牌在 `tokens.css` 或主题块中有定义
2. 运行门禁：`pnpm typecheck && pnpm test && pnpm build`
3. 双主题目测：分别在 light/dark 下渲染组件，确认颜色正确

### 组件选型

需要新增组件时，先查 `references/knowledge/shadcn-components.md`（63 个组件按用途分类，标注已安装状态），确认是否已安装、用途是否匹配。

## 6. 开工前必查（按场景检索）

**写代码前，先浏览 `references/knowledge/` 对应分类的标题，确认有没有"已知的坑"。**

- 写前端/UI/CSS/组件 → 看 `knowledge/前端.md`
- 安装 shadcn 组件 → 用 `shadcn.mjs`（见第 5 节）
- 写 Go/数据库/API → 看 `knowledge/后端.md`
- 部署/CI/环境配置 → 看 `knowledge/实施.md`
- 浏览器截图/UI 调试 → 用 `cdp-capture.mjs`（见第 3 节）
- 页面设计稿生成 → 用 `gpt-image-generate.mjs`（见第 4 节）
- 不确定 → 看 `knowledge/README.md` 速查统计表

每个分类文件末尾有"开工前 grep 关键词"，用这些词检索所有 references 文件。

## 7. 反馈优先级

最值钱的先写：

1. 静默失败，skill 没预警（浪费几小时）
2. 字段名、API 签名、配置 key 写错
3. 措辞模糊，把你引向错误方向
4. 遗漏的排查手法