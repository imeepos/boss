# 官网内容发布域（cms）落地决策

日期：2026-08-28 ｜ 状态：adopted

## 决策

1. **单一内容表 `cms_posts` + category 区分**（NEWS/ARTICLE），不做多内容类型表。
   Why：官网阶段只需动态/文章两类，分类是可空维度的过滤；多表会让 admin 页、公开读、迁移各翻倍。
   放弃了：独立 news 表 + articles 表的两表方案（结构 90% 重复）。
2. **公开只读走 admin 前缀免鉴权子路由**（`/api/admin/v1/site/posts`，仅 PUBLISHED），
   沿用 partner 入驻公开提交先例（registerPartnerPublicRoutes）。
   放弃了：新建 `/api/public/v1` 独立路由组（多一条无鉴权暴露面，需独立限流/审计策略，收益不成正比）；
   放弃了：复用 openplat HMAC 通道（官网匿名访问无 AppId 概念）。
3. **正文 Markdown 而非富文本 HTML**。Why：富文本 HTML 有 XSS 面，Markdown 渲染在客户端可控；
   admin 编辑器先用 textarea + 预览，不引重型所见即所得依赖。
4. **首页继续寄生 web/admin bundle**，本决策只新增"动态与新闻"区块拉公开接口；
   是否拆独立 portal 不在本期裁定。
5. **发布语义**：status DRAFT→PUBLISHED 时后端落 published_at=now()；OFFLINE 下线不删数据；
   定时发布（SCHEDULED）暂不做，published_at 即发布时刻。

## 依据

社区共识（Ghost/Strapi/textpattern 类内容模型）：title/slug/summary/cover/content +
DRAFT/PUBLISHED 状态机 + 公开端只吐已发布 + version 自增防并发覆盖。本地范式抄
cs_knowledge_articles（000118）。

## Amended（2026-08-24 本轮修订，见 feat/cms-editor-categories）

1. **决策 1（分类）修订**：分类从 NEWS/ARTICLE 枚举改为自定义字典表
   `cms_categories`（000138）。Why：运营需要自定义分类与排序/启停管理；
   放弃了：保留双枚举（无法扩展）。存量约束：code 被文章引用时禁删/禁改 code。
2. **决策 3（编辑器）修订**：保留"正文 Markdown 而非富文本 HTML"的 XSS 裁定，
   但编辑体验升级为工具栏 + react-markdown 双栏实时预览（Ghost 式），
   图片经附件域（MinIO/S3）上传后以 `](att/N)` 引用；公开读由后端重写为
   `/site/posts/:slug/img/:attId`（仅"该文引用 + image/*"，防枚举）。
   放弃了：引入 TipTap 类所见即所得（HTML 输出重新打开 XSS 面 + 重依赖）。
3. 新增裁定：添加/编辑走独立路由页（/boss/site/new、/:postId），
   列表页只留列表，内联表单废弃。

## Amended（2026-08-27 本轮修订，见 feat/cms-multilang）

1. **决策 1（内容模型）修订：cms_posts 加语言维度 `lang`（000155）**。
   Why：官网界面已三语言（zh-CN/en-US/ms-MY），内容仍单语言导致切语言后文章
   文案不跟随，是产品级缺口。方案：同 slug 多语言变体各存一行，唯一键从 slug
   改为 `(slug, lang)`，公开 URL `/news/:slug?lang=` 不变换路径只换参数；
   公开读缺变体回退默认语言（不 404）。放弃了：独立 i18n 翻译子表
   `cms_post_i18n(post_id, lang, ...)`（文章身份/版本/状态需在父子表间拆解，
   现有 version 自增与发布语义会翻倍复杂）；放弃了：slug 按语言本地化
   （/news/hello-world 与 /news/zh-hi 双 URL 体系，SEO 收益不确定）。
2. **新增裁定：cms_categories 加 `name_i18n` JSONB 多语言名（000155）**。
   `name` 保留为默认/回退；语言覆盖值缺失、或覆盖值为空串时回退 `name`。
   放弃了：分类名独立 i18n 表（字典表加一列即可，独立表过重）。
3. 前端配套：编辑页加语言下拉；MarkdownEditor 硬编码文案 i18n 化；
   公开端 NewsSection/详情按界面语言传 `lang`，分类标签优先用后端本地化
   `categoryName`（缺省回落 NEWS/ARTICLE 枚举标签）。
