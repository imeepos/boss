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
