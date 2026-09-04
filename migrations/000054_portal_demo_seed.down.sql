-- 回滚 000054:清除门户演示种子(幂等;演示客户按手机号 13900001234 寻址)。
-- 限制:
--  1) user_usages 的用量行按演示客户整行清除(up 仅写执行当月,down 无法还原该月份边界);
--  2) portal_wallets 仅在余额仍等于种子值 88.00 时删除,期间有真实充值则保留不动;
--  3) addons 仅在无任何订阅引用时删除,channels 的 ONLINE/AGENT 为目录行直接删除(无 FK 挂接)。
BEGIN;

DELETE FROM coupons WHERE coupon_id = 'C-DEMO-001';

DELETE FROM addon_subscriptions s
USING customers c
WHERE s.customer_id = c.id AND c.phone = '13900001234'
  AND s.addon_id = 'iptv' AND s.action = 'subscribe';

DELETE FROM addons a
WHERE a.addon_id IN ('iptv', 'wifi6', 'cloudcam')
  AND NOT EXISTS (SELECT 1 FROM addon_subscriptions s WHERE s.addon_id = a.addon_id);

DELETE FROM user_usages u USING customers c
WHERE u.customer_id = c.id AND c.phone = '13900001234';

DELETE FROM user_plans p USING customers c
WHERE p.customer_id = c.id AND c.phone = '13900001234' AND p.product_id = 101;

DELETE FROM user_addresses a USING customers c
WHERE a.customer_id = c.id AND c.phone = '13900001234' AND a.addr_code = 'HOME';

DELETE FROM portal_wallets w USING customers c
WHERE w.customer_id = c.id AND c.phone = '13900001234' AND w.balance = 88.00;

DELETE FROM topup_denominations
WHERE denom_id IN ('denom-50', 'denom-100', 'denom-200');

DELETE FROM channels WHERE code IN ('ONLINE', 'AGENT');

COMMIT;
