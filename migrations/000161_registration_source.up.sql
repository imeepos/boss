-- 000161: customer_registrations 注册来源(官网获客闭环,2026-09 M4 3.4)。
-- 记录注册来源标识(如 landing-m4/app/web),用于官网到注册的转化追踪;空串=未携带。
ALTER TABLE customer_registrations ADD COLUMN source VARCHAR(32) NOT NULL DEFAULT '';
