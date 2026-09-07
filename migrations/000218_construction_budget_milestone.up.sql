-- 000218(P-INFRA-1 W6,G2 剩余): 施工项目预算金额与里程碑清单。
-- 预算=项目可挂预算金额(可空=未登记);执行进度=已结算金额(SETTLED 结算单合计,只读派生)/预算。
-- 里程碑=名称/计划完成日/状态;编辑口径(terms.md §4):预算与里程碑清单(增删/改名/计划日)
-- 仅项目 PENDING 可改;里程碑状态可标记至项目 ACCEPTED 前;ACCEPTED 后全部锁定。
BEGIN;

ALTER TABLE construction_projects
    ADD COLUMN budget_amount NUMERIC(14,2) CHECK (budget_amount IS NULL OR budget_amount >= 0);

CREATE TABLE construction_milestones (
    id           BIGSERIAL PRIMARY KEY,
    project_id   BIGINT NOT NULL REFERENCES construction_projects(id),
    name         VARCHAR(128) NOT NULL,
    planned_date DATE,
    status       VARCHAR(16) NOT NULL DEFAULT 'PENDING'
                 CHECK (status IN ('PENDING','DONE')),
    done_at      TIMESTAMPTZ,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_construction_milestones_project ON construction_milestones(project_id);

COMMENT ON COLUMN construction_projects.budget_amount IS '工程预算金额,NULL=未登记(000218);执行进度=SETTLED 结算合计/预算,只读派生';
COMMENT ON TABLE  construction_milestones             IS '施工项目里程碑清单(000218);编辑锁口径见 terms.md §4';
COMMENT ON COLUMN construction_milestones.planned_date IS '计划完成日(DATE 可空)';
COMMENT ON COLUMN construction_milestones.status       IS 'PENDING 未完成 / DONE 已完成;完成标记至项目 ACCEPTED 前';

COMMIT;
