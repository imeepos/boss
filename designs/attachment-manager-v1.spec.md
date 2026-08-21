# 附件管理器设计规格（对应 attachment-manager-v1.png）

## 设计目标

解决当前 AttachmentManager 仅展示 `contentType` 文本、缺乏视觉区分的问题：按 MIME/扩展名识别 9 类文件，按类型渲染彩色图标徽章，让用户一眼区分图片/音频/视频/PDF/文档/表格/压缩包/代码/其他，并支持左侧分类计数导航。

## 给前端的实现提示词（可直接复制）

实现一个 PC 端 admin 附件管理组件，高保真还原附图设计稿。组件既保留原有 `AttachmentManager` 公开 props（`uploaderType`/`uploaderId`/`selectable`/`selectedIds`/`onSelectionChange`）与上传/查询/软删/选择等行为，新增以下能力：

### 整体布局（桌面端 ≥1024px）

- **顶栏**（沿用现有样式）：标题 + 搜索框（文件名）+ 文件类型下拉筛选 + 上传按钮 + 视图切换图标按钮（列表/网格，本期仅前端 UI 切换，数据模型不变）。
- **左侧分类侧栏**（240px 固定宽度，亮色 `#FCFCFD`/暗色 `#0F1E3B`，右侧 1px 边）：9 类 + "全部"，每项 `[彩色图标 18px] 标签 [数字 badge]`。激活项背景 `#FBF5E8`/暗色 `rgba(213,166,58,0.08)`，左侧 3px 金色竖条。点击切换筛选并把 `page` 置 1。
- **右侧主区**：原表格行为保留，行首插入 32×32 圆角 6px 浅底彩色图标徽章（图标 18px 描边 `currentColor`，底色为分类色的 14% alpha）。
- **底部**：原有分页器保持；选中项数量展示改为左下"已选 N 项"+ 批量下载（本期 UI 占位）+ 批量删除。
- **响应式**：<1024px 隐藏左侧分类侧栏，仅保留顶部文件类型下拉。

### 9 类文件分类（核心新增）

| Key | 中文 | 英文 | 马来文 | 颜色（亮） | 匹配规则 |
|---|---|---|---|---|---|
| image | 图片 | Images | Imej | `#176BF2` | `image/*` 或 ext `jpg/jpeg/png/gif/webp/svg/bmp/heic` |
| audio | 音频 | Audio | Audio | `#9333EA` | `audio/*` 或 ext `mp3/wav/aac/flac/ogg/m4a` |
| video | 视频 | Video | Video | `#D94B4B` | `video/*` 或 ext `mp4/avi/mov/mkv/webm/flv` |
| pdf | PDF | PDF | PDF | `#D97706` | `application/pdf` 或 ext `pdf` |
| document | 文档 | Docs | Dokumen | `#2F8F63` | `application/msword`、`application/vnd.openxmlformats-officedocument.wordprocessingml.document`、ext `doc/docx/rtf/odt` |
| spreadsheet | 表格 | Sheets | Hamparan | `#0891B2` | `application/vnd.ms-excel`、`application/vnd.openxmlformats-officedocument.spreadsheetml.sheet`、ext `xls/xlsx/csv/ods` |
| archive | 压缩包 | Archives | Arkib | `#D5A63A` | `application/zip`、`application/x-rar-compressed`、`application/x-7z-compressed`、`application/x-tar`、`application/gzip`、ext `zip/rar/7z/tar/gz` |
| code | 代码 | Code | Kod | `#475569` | ext `json/xml/yaml/yml/sql/sh/js/ts/tsx/jsx/html/css/md/py/go/java/c/cpp/h/hpp` |
| other | 其他 | Other | Lain | `#737A86` | 兜底分类 |

`Color` 用项目现有设计令牌：蓝/绿/红/青与项目状态色一致，金色沿用 `--shell-nav-line` `#D5A63A`，紫/橙/灰为补充但写入 CSS 变量以保证暗色主题。

### 图标 SVG（24 viewBox / stroke 1.8 / round / currentColor）

为 9 类各画一个描边 SVG，统一存放到 `src/components/AttachmentManager/icons/` 下，文件名 = `category.tsx` 内 `categoryKey`（image/audio/video/pdf/document/spreadsheet/archive/code/other）。可使用 `lucide-react` 已安装库中现成图标（如 `FileImage/Music/Video/FileText/FileSpreadsheet/FileArchive/FileCode2/File`），保持视觉一致。

### 计数（左侧分类侧栏）

服务端不返回分类计数（避免破坏契约），改为前端**对当前页内可见数据估算**：`categoryCount` 字段为 `Map<CategoryKey, number>`，由当前 `items` 派生，并在切换时同时调用 `listAttachments` 重拉（后端会按新筛选返回最新数据）。每页展示的数据已代表该类在该页的全部，加 badge 显示该页内数量；当筛选为"全部"时，"全部"badge = `total`（含分页合计），其余分类 badge = 当前页内该类数（前端可见，加 tooltip 提示"仅显示当前页"）。

### 文件类型下拉（顶部）

保留上传者下拉位置（沿用现有逻辑），新增文件类型下拉：`全部/图片/音频/视频/PDF/文档/表格/压缩包/代码/其他`，值映射到上面 9 类 key。本地状态 `typeSel: CategoryKey | ''`，影响下一次请求是否带上 `category` 过滤。

### 后端契约（本期**不修改后端**）

后端 `/attachments` 仅按 `uploaderType/uploaderId/keyword/limit/offset` 过滤，新增的"按类型筛选"在前端本地实现：从 `attachment.contentType` 或 `attachment.fileName` 派生 `categoryKey`，客户端过滤后再分页显示。说明：**前端分页仅作用于"过滤后的结果集"，因此"分页器总数"会出现与 `total` 不一致**——按类型筛选时使用 `filtered.length` 替代 `total` 渲染分页，并在分页器左侧加灰色小字 "客户端过滤 N / 服务端总数 M" 提示（如不需要可省略）。

### i18n 新增 key（同步 3 份 locale + types.ts）

`attachmentManager.fileCategory.image | audio | video | pdf | document | spreadsheet | archive | code | other | all | filteredHint | downloadSelected | tipPageScope`

### 交互与状态

- 表格行 hover：背景 `#FBF5E8`（亮）/`rgba(213,166,58,0.04)`（暗），过渡 150ms。
- 选中行：背景 `--shell-fab-bg` 12% alpha + 左侧 3px 金色竖条；checkbox 在选中态填充 `--shell-fab-bg`。
- 加载态：表格头保留，行内显示骨架行（5 行高 44px 闪烁 1.5s）。
- 空态：表格居中显示 "暂无附件" + 上传图标 CTA。
- 错误态：表格上方红色提示条（同现有 `ErrorBanner`）。
- 类型筛选切换：500ms 过渡，左侧侧栏激活项平滑迁移。

### 数据接口（前端）

```ts
export type FileCategoryKey = 'image' | 'audio' | 'video' | 'pdf' | 'document' | 'spreadsheet' | 'archive' | 'code' | 'other'

export interface FileCategoryMeta {
  key: FileCategoryKey
  label: string
  icon: ComponentType<{ className?: string }>
  /** 主色（hex，仅亮色使用；暗色主题走 shadcn token） */
  color: string
  /** 暗色主题下的颜色 */
  colorDark: string
  /** 匹配测试：返回 true 即归入此类 */
  match: (contentType: string, fileName: string) => boolean
}

export function classifyAttachment(contentType: string, fileName: string): FileCategoryKey
```

### 复用与依赖

- 图标使用 `lucide-react`（项目已安装）；如未安装可手写 24px 描边 SVG。
- Dropdown 用 `components/Dropdown.tsx`，禁止原生 `<select>`。
- Dropdown/Pagination/Table 组件复用现有，Button 用 `ui/button.tsx`（variants: outline/ghost/destructive）。

## 注意事项

- 设计稿中所有数字、时间、人名均为 AI 占位，真实数据以接口返回为准。
- 颜色 hex 来自项目设计令牌（`#176BF2/#9333EA/#D94B4B/#D97706/#2F8F63/#0891B2/#D5A63A/#475569/#737A86`），深色主题下需做透明度调整或单独 token；如项目无对应 token，使用 `tailwind.config.js` 的 `tailwindcss-animate` + `[data-theme="dark"]` 覆盖。
- 不引入额外状态管理库（zustand 等），分类侧栏选中态用 `useState`。
- 文件类型分类逻辑放在 `logic.ts`（纯函数），写单测覆盖边界（MIME 与扩展名冲突/空 contentType/未知扩展名/中文文件名）。
- 不改后端 schema；按类型筛选用客户端过滤实现。
- 不删除任何现有 props，向后兼容 `selectable/uploaderType/uploaderId/selectedIds/onSelectionChange`。
- 文件下载按钮本期只渲染 UI，不接真实下载接口（避免误删原逻辑），可标 `// TODO: 接入 /attachments/:id/content` 注释。

## 验收清单

- [ ] 9 类文件分别渲染对应的彩色图标与标签
- [ ] 左侧分类侧栏显示每类数量 badge，激活态样式正确
- [ ] 顶部文件类型下拉切换可过滤当前列表
- [ ] 双主题（亮/暗）渲染正常，颜色对比度 ≥ 4.5:1
- [ ] typecheck + test + build 通过
- [ ] 单测覆盖 `classifyAttachment` 全部 9 类 + 边界
- [ ] 现有 props 完全兼容，未修改 `AttachmentManagerProps` 接口
- [ ] 不引入新依赖（除非 lucide-react 已存在）