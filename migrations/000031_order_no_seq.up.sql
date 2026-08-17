-- 订单号序号:进程内原子计数在重启后重复,违反 orders.order_no 唯一约束。
-- 改由数据库序列发号,跨进程/重启不重复。
BEGIN;
CREATE SEQUENCE IF NOT EXISTS order_no_seq;
COMMIT;
