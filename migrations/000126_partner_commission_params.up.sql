-- 渠道佣金自动计提默认比例：订单进入 DONE 后仅对渠道企业订单计提。
BEGIN;
INSERT INTO biz_params(key, value, description) VALUES
 ('commission.partnerDefaultRate', '0.10', '伙伴渠道订单默认佣金比例')
ON CONFLICT (key) DO NOTHING;
COMMIT;
