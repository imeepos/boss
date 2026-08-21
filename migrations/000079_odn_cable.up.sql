-- ODN 光缆段落与纤芯(《Suniway ODN 地理空间编码规范》第 5 章,E8)。
-- 段落存储为两端设备编码列(a_code/b_code),"--"仅设计图纸不入库(规范 5.1);
-- A/B 方向由设备优先级决定(规范 5.2):SNW/机房>ODF>OCC>CLS>ODB>P>MH,TW/TBX 最低档兜底。
BEGIN;

CREATE TABLE odn_cable_segment (
    id         BIGSERIAL PRIMARY KEY,
    a_code     VARCHAR(16) NOT NULL, -- A 端(优先级较高设备),现有设施码或未来核心设备码
    b_code     VARCHAR(16) NOT NULL, -- B 端(优先级较低设备)
    name       VARCHAR(64) NOT NULL DEFAULT '',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_odn_segment UNIQUE (a_code, b_code),
    CONSTRAINT chk_odn_segment_ab CHECK (a_code <> b_code),
    CONSTRAINT chk_odn_segment_code CHECK (a_code ~ '^(P|MH|TW|CLS|TBX|SNW|ODF|OCC|ODB|SDB|PRT|TBP|OLT)[0-9]{3,5}$'
                                       AND b_code ~ '^(P|MH|TW|CLS|TBX|SNW|ODF|OCC|ODB|SDB|PRT|TBP|OLT)[0-9]{3,5}$')
);

CREATE TABLE odn_fiber (
    segment_id BIGINT      NOT NULL REFERENCES odn_cable_segment(id) ON DELETE CASCADE,
    g_no       SMALLINT    NOT NULL CHECK (g_no BETWEEN 1 AND 99), -- G01~G99 同段落光缆顺序(规范 5.3)
    kind       VARCHAR(32) NOT NULL DEFAULT '', -- 光缆型号描述
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (segment_id, g_no)
);

COMMIT;
