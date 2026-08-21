# 2026-08-25 附件删除采用软删除（attachments.deleted_at，迁移 000093）

## 裁定

admin 端删除附件 = 软删除（`deleted_at` 置时间戳），列表/查询默认过滤；MinIO 对象保留，不物理删除、不从桶移除。

## Why

- `attachments` 被 `verifications.id_card_front_id/back_id`（000070）等业务列引用，物理删除会产生悬空 id（0=未传 与 已删无法区分）。
- 上传者三主体（account/worker/customer）多态软引用，无法穷举反向引用做删除前检查（方案"物理删 + 引用检查"被放弃：引用面随业务扩张，检查清单必然过时）。
- 审计要求：删除动作可追溯（attachment.delete 记审计日志），对象保留支撑事后取证。

## 放弃了什么

- 物理删除 + 引用检查：见上，检查面不可维护。
- MinIO 生命周期策略联动：留待对象存储成本真正成为问题再做，当前数据量不值得。
- 恢复接口：暂无需求，软删数据 DBA 可手工恢复。

## 影响

- Store 全部读路径（Get/ListByUploader/List/GetByIDs）统一 `deleted_at IS NULL` 过滤。
- 契约同步：docs/contract/fields.md §1.6.4。
