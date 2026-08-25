-- 装维队管理(000141):worker_groups 软删列。
-- 归属台账 worker_group_memberships 已由 000019 建立,此处不重复;
-- 软删语义见 fields.md §7.3 铁律 1(队伍删除被 FK 阻止,应软删)。
BEGIN;

ALTER TABLE worker_groups ADD COLUMN deleted_at TIMESTAMPTZ; -- null=在职装维队

COMMIT;
