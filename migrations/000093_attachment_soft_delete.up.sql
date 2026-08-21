-- 附件软删除:attachments 被 verifications(id_card_front/back_id) 等业务引用,
-- 物理删除会产生悬空引用;删除=置 deleted_at,列表默认过滤,MinIO 对象保留可审计。
ALTER TABLE attachments ADD COLUMN deleted_at TIMESTAMPTZ NULL;
