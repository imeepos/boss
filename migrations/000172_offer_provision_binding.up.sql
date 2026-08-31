-- 产品/套餐 ↔ 下发模板显式绑定(方案B)。
-- 推翻 2026-08-30 preconfig-template-resolution 纯带宽匹配裁定:后台把套餐和下发模板绑死,
-- 环节7 优先走绑定,杜绝"买套餐开错模板/无绑定开不了"。
-- 裁决见 docs/notes/adopted/2026-09-01-offer-provision-binding.md。
BEGIN;

CREATE TABLE offer_provision_bindings (
    id              BIGSERIAL PRIMARY KEY,
    legal_entity_id BIGINT NOT NULL REFERENCES legal_entities(id),
    offer_id        BIGINT NOT NULL UNIQUE REFERENCES product_offers(id),
    template_id     BIGINT NOT NULL REFERENCES provision_templates(id),
    remark          VARCHAR(128),
    created_at      TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_offer_prov_bindings_entity ON offer_provision_bindings(legal_entity_id);
CREATE INDEX idx_offer_prov_bindings_template ON offer_provision_bindings(template_id);

COMMIT;
