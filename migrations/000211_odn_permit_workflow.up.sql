-- 000211(P-INFRA-1 W4): ROW 路权与 PECE 许可单工作流(审查 F3)。
-- 许可单承载 ROW(路权)/PECE 两类许可,可关联施工项目/设施/资源链,含证照档案要素
-- (批复号/管辖机构/有效期/附件引用);状态机登记 terms.md 4/fields.md 1.5.13;
-- 施工开工许可前置门控灰度 BOSS_ODN_PERMIT_GATE 默认关(与覆盖门控同模式);
-- 竣工 ACCEPTED 覆盖联动见 000211 域层(无 DDL,BOSS_ODN_ACCEPT_COVERAGE_LINK=off 可关)。
BEGIN;

CREATE TABLE odn_permits (
    id             BIGSERIAL PRIMARY KEY,
    permit_no      VARCHAR(32) NOT NULL UNIQUE,
    kind           VARCHAR(16) NOT NULL CHECK (kind IN ('ROW','PECE')),
    title          VARCHAR(128) NOT NULL DEFAULT '',
    approval_no    VARCHAR(128) NOT NULL DEFAULT '',
    authority      VARCHAR(128) NOT NULL DEFAULT '',
    valid_from     DATE,
    valid_until    DATE,
    status         VARCHAR(16) NOT NULL,
    project_id     BIGINT REFERENCES construction_projects(id),
    project_no     VARCHAR(32) NOT NULL DEFAULT '',
    facility_code  VARCHAR(16) REFERENCES odn_facility(code),
    chain_id       BIGINT,
    attachment_ids BIGINT[] NOT NULL DEFAULT '{}',
    note           VARCHAR(255) NOT NULL DEFAULT '',
    reject_reason  VARCHAR(255) NOT NULL DEFAULT '',
    created_by     BIGINT REFERENCES accounts(id),
    created_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT odn_permits_status_chk CHECK (
        (kind = 'ROW'  AND status IN ('NOT_STARTED','PENDING','APPROVED','EXPIRED','NA')) OR
        (kind = 'PECE' AND status IN ('PENDING_SIGN','SIGNED','STAMPED','NA'))
    )
);
CREATE INDEX idx_odn_permits_project ON odn_permits(project_id);
CREATE INDEX idx_odn_permits_kind_status ON odn_permits(kind, status);

COMMENT ON TABLE  odn_permits                IS 'ROW 路权与 PECE 许可单(F3,W4,000211);状态机 terms.md 4';
COMMENT ON COLUMN odn_permits.approval_no    IS '批复号(ROW 批准时必填)';
COMMENT ON COLUMN odn_permits.authority      IS '管辖机构';
COMMENT ON COLUMN odn_permits.valid_until    IS '有效期止(ROW 批准时必填);过期由开工门控判定并自动回写 EXPIRED';
COMMENT ON COLUMN odn_permits.project_id     IS '关联施工项目(可空);project_no 为关联时快照';
COMMENT ON COLUMN odn_permits.facility_code  IS '关联设施(odn_facility.code,可空)';
COMMENT ON COLUMN odn_permits.chain_id       IS '关联资源链 odn_resource_chain(id) 软引用(可空,无 FK)';
COMMENT ON COLUMN odn_permits.attachment_ids IS '证照附件引用 attachments(id) 数组';
COMMENT ON COLUMN odn_permits.reject_reason  IS '驳回/退回/作废原因(对应流转必填)';

COMMIT;
