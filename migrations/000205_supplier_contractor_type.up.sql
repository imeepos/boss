-- 000205(原预分配 000203,kaihu 线 000203_vlan_columns_integer 先行合入 main,按让号规则顺延): 供应商档案承建类型维度(P-INFRA-1 W1)。
-- MATERIAL 材料类(存量默认,既有语义不变)/ CONSTRUCTION 施工类(含资质信息 qualification;
-- 联系人复用既有 contact_name/contact_phone,不另立字段)。
ALTER TABLE procurement_suppliers
    ADD COLUMN contractor_type VARCHAR(16) NOT NULL DEFAULT 'MATERIAL'
        CHECK (contractor_type IN ('MATERIAL','CONSTRUCTION')),
    ADD COLUMN qualification VARCHAR(255);

COMMENT ON COLUMN procurement_suppliers.contractor_type IS '承建类型:MATERIAL 材料类 / CONSTRUCTION 施工类(000205)';
COMMENT ON COLUMN procurement_suppliers.qualification IS '施工类资质信息(等级/编号/有效期自由文本,可空,000205)';
