-- 渠道基础反作弊参数：可通过 biz_params 热更，默认值与当前规则一致。
BEGIN;
INSERT INTO biz_params(key, value, description) VALUES
 ('fraud.partnerDailyOrderCap', '100', '伙伴企业每日渠道订单上限'),
 ('fraud.partnerCustomerCooldownHours', '24', '伙伴同一客户重复下单冷却小时数')
ON CONFLICT (key) DO NOTHING;
COMMIT;
