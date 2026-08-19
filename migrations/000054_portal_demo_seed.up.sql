-- 门户端到端演示种子(幂等):渠道目录 + 演示客户(手机号 13900001234)的门户钱包/地址/套餐/增值/券/用量。
-- 目标:Android 端全部页面在真实服务上首次登录即有可读数据;重复执行不产生重复行。
-- 幂等策略:有唯一约束的表用 ON CONFLICT;无唯一约束的表用 NOT EXISTS 守卫。
BEGIN;

-- 渠道目录补齐(仅 HALL 已在 000027 人工录入;ONLINE/AGENT 为下单必选项)。
INSERT INTO channels (code, name, status) VALUES
  ('ONLINE', '线上', 'ACTIVE'),
  ('AGENT',  '代理商', 'ACTIVE')
ON CONFLICT (code) DO NOTHING;

-- 演示客户按手机号寻址(已存在则挂载,不存在则不打扰)。
INSERT INTO portal_wallets (customer_id, balance)
SELECT id, 88.00 FROM customers WHERE phone = '13900001234'
ON CONFLICT (customer_id) DO NOTHING;

INSERT INTO user_addresses (customer_id, addr_code, contact, phone, detail, is_default)
SELECT id, 'HOME', '王先生', phone, 'Sunrise Village, Building 8, Unit 502', TRUE
FROM customers c
WHERE c.phone = '13900001234' AND NOT EXISTS (
  SELECT 1 FROM user_addresses a WHERE a.customer_id = c.id AND a.addr_code = 'HOME'
);

INSERT INTO user_plans (customer_id, product_id, plan_name, status, effective_at)
SELECT c.id, 101, '家庭宽带100M', 'ACTIVE', now()
FROM customers c
WHERE c.phone = '13900001234' AND NOT EXISTS (
  SELECT 1 FROM user_plans p WHERE p.customer_id = c.id AND p.product_id = 101
);

INSERT INTO addons (addon_id, name, price, status, subscriber_count) VALUES
  ('iptv', 'IPTV 高清电视', 1000, 'on', 0),
  ('wifi6', 'WiFi6 千兆路由', 500, 'on', 0),
  ('cloudcam', '云监控回看', 2000, 'on', 0)
ON CONFLICT (addon_id) DO NOTHING;

INSERT INTO addon_subscriptions (customer_id, addon_id, action)
SELECT c.id, 'iptv', 'subscribe' FROM customers c
WHERE c.phone = '13900001234' AND NOT EXISTS (
  SELECT 1 FROM addon_subscriptions s
  WHERE s.customer_id = c.id AND s.addon_id = 'iptv' AND s.action = 'subscribe'
);

INSERT INTO coupons (coupon_id, customer_id, name, amount, status, expire_at)
SELECT 'C-DEMO-001', c.id, '缴费满 100 减 20', 2000, 'active', now() + interval '90 days'
FROM customers c
WHERE c.phone = '13900001234' AND NOT EXISTS (
  SELECT 1 FROM coupons cp WHERE cp.coupon_id = 'C-DEMO-001'
);

INSERT INTO user_usages (customer_id, month, upload_gb, download_gb, total_gb)
SELECT id, to_char(now(), 'YYYY-MM'), 56.6, 300.2, 356.8
FROM customers WHERE phone = '13900001234'
ON CONFLICT (customer_id, month) DO NOTHING;

INSERT INTO topup_denominations (denom_id, amount, bonus, active) VALUES
  ('denom-50', 5000, 0, TRUE),
  ('denom-100', 10000, 500, TRUE),
  ('denom-200', 20000, 1500, TRUE)
ON CONFLICT (denom_id) DO NOTHING;

COMMIT;