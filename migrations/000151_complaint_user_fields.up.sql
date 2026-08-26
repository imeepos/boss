-- 000151: 用户端投诉与建议页改造 — 投诉与处理记录分页 + 详情。
-- 背景：原 complaints 表仅承载报障工单（装维诊断字段），用户端 POST /complaints
-- 只写 type 落库，description/relOrderNo/contact 在 handler 里被丢弃；列表与详情都
-- 无法区分。本次新增 description/contact/rel_order_no 三个可选列，专门承接用户端
-- 投诉与建议提交的内容；remote_diagnosis 仍归装维诊断使用，避免语义混淆。
--
-- 设计要点：
--  1) description TEXT NOT NULL DEFAULT '' 与装维 remote_diagnosis 同形，可空。
--  2) contact VARCHAR(32) NOT NULL DEFAULT '' 留长一些，兼容国内手机/座机。
--  3) rel_order_no VARCHAR(64) NOT NULL DEFAULT '' 与 ORD-20250817-001 工单号位数对齐。
--  4) created_at 已存在（000087），无需重复定义。
--  5) 全部 DEFAULT '' 兜底，老数据不会因 NOT NULL 失败；老 complaint 单无 description
--     渲染时空串，不阻塞历史分页。
BEGIN;

ALTER TABLE complaints ADD COLUMN IF NOT EXISTS description   TEXT        NOT NULL DEFAULT '';
ALTER TABLE complaints ADD COLUMN IF NOT EXISTS contact       VARCHAR(32) NOT NULL DEFAULT '';
ALTER TABLE complaints ADD COLUMN IF NOT EXISTS rel_order_no  VARCHAR(64) NOT NULL DEFAULT '';

-- 用户端列表分页走 customer_id 过滤 + created_at DESC 排序，加复合索引避免
-- 大量历史投诉时走全表 + sort。status 单独加索引，便于按 OPEN/PROCESSING 过滤。
CREATE INDEX IF NOT EXISTS idx_complaints_customer_created
    ON complaints(customer_id, created_at DESC);
CREATE INDEX IF NOT EXISTS idx_complaints_status
    ON complaints(customer_id, status);

COMMIT;