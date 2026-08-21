-- 用户端实名认证分步流程(designs/realname-flow-states-v1):核验记录存驳回原因与证件附件。
-- reject_reason 仅 FAIL 有值;id_card_front/back_id 软引用 attachments.id(与 subject_id 同规,不加 FK)。
BEGIN;

ALTER TABLE verifications
    ADD COLUMN reject_reason    VARCHAR(255) NOT NULL DEFAULT '',
    ADD COLUMN id_card_front_id BIGINT       NOT NULL DEFAULT 0,
    ADD COLUMN id_card_back_id  BIGINT       NOT NULL DEFAULT 0;

COMMIT;
