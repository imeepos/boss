-- 阶段6 演进:师傅自助注册(group_id/region_id 允许 NULL,后台审核时补正)。
-- adopted docs/notes/adopted/2026-XX-XX-worker-self-registration-loose-foreign-key.md
BEGIN;

ALTER TABLE worker_registrations
    ALTER COLUMN group_id  DROP NOT NULL,
    ALTER COLUMN region_id DROP NOT NULL;

ALTER TABLE worker_registrations
    DROP CONSTRAINT worker_registrations_group_id_fkey;

-- group_id 改为 nullable 之后旧 FK 自动弃用,新增一条允许 NULL 的可空约束(去 NOT NULL 即可)。
ALTER TABLE worker_registrations
    ADD CONSTRAINT worker_registrations_group_id_fk
        FOREIGN KEY (group_id) REFERENCES worker_groups(id);

COMMIT;
