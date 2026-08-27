# 经验知识分类索引

> 原始资料在 `references/` 下各文件，只增不改。此处仅做索引，方便按领域快速检索。
>
> 分类规则：按**经验适用场景**而非"这个文件是什么"归类。一条经验可能跨类，优先归入最常使用的场景。

## 四类

| 分类 | 适用于 | 典型场景 |
|:-----|:-------|:---------|
| [前端](前端.md) | web/admin React 页面、CSS/主题、i18n、CDP 截图、UI 组件 | 写页面、改样式、加主题、补语言、截图验证 |
| [Android](android.md) | mobile/worker 与 mobile/user（Kotlin + Compose）、真机 adb 验证 | 写 Compose 页面、导航栈、对接后端 API、真机联调 |
| [后端](后端.md) | Go 服务端、数据库、迁移、API、pgx、契约模型 | 写 API、改 SQL、加迁移、查 500、对字段 |
| [实施](实施.md) | 部署、CI/CD、Docker、环境配置、密钥、基础设施 | 搭环境、配 CI、上线、排查部署问题 |

## 速查索引

> 固定模板(新增后台域 checklist / 102 回放脚本 / cdp token 注入截图 / 公开端点三原则 /
> 前端 cdp 调试 / 通用排障八步 / 表单抽屉化布局 / 双主题双语言适配 / 免登录采集脚本 /
> 新增 Webhook 可订阅事件)见
> [`references/templates.md`](../templates.md),开工前先扫一眼对号入座。

### 按文件来源

| 源文件 | 前端 | 后端 | 实施 | 通用 |
|--------|------|------|------|------|
| `lessons.md` (91 条) | 26 | 31 | 13 | 21 |
| `known-issues.md` (22 条) | 8 | 8 | 2 | 4 |
| `red-lines.md` (19 条) | 7 | 1 | 3 | 8 |
| `techniques.md` (39 条) | 10 | 20 | 7 | 2 |
| `templates.md` (9 模板) | 5 | 2 | 1 | 1 |
| **合计** | **54** | **57** | **24** | **36** |

> 通用类：编辑工具使用、流程规范、模型限制等与具体技术栈无关的经验。

### 专项速查

| 文件 | 用途 | 说明 |
|:-----|:-----|:-----|
| [shadcn-components.md](shadcn-components.md) | shadcn/ui 组件选型 | 63 个组件按用途分类，标注已安装状态，含安装命令与注意事项 |
| [prompt-templates.md](prompt-templates.md) | 提示词模板/指南 | 设计稿→页面布局提示词（Web 版 + Android 版）、DSH Goal objective 编写指南；全文内化在 knowledge/ 下，跨项目可复用 |
| [design-aesthetics.md](design-aesthetics.md) | 设计美学手册 | Stripe/Linear/Vercel/Bloomberg 设计语言精粹（字体纪律、ring shadow、双层焦点、80/15/5 比例、节拍检测、避坑清单）；画稿前必读 |

### 开工前必查

写什么代码前，先看一眼对应分类的标题，确认有没有"已知的坑"：web/h5 看 [前端](前端.md)，mobile 双端看 [Android](android.md)。详见各分类文件末尾的"开工前 grep 关键词"建议。
需要选用 shadcn/ui 组件时，先看 [shadcn-components.md](shadcn-components.md) 确认是否已安装、用途是否匹配。
要把设计稿变成页面提示词、或要建 Goal 写 objective 时，先看 [prompt-templates.md](prompt-templates.md)。