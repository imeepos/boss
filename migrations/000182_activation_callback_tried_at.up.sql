-- 激活回调尝试时刻(2026-09-04 任务A-d):师傅端 GET activation lastTry 可见。
-- activation_callbacks 同订单唯一行(000154 幂等 upsert),tried_at 随每次尝试刷新;
-- 建列默认 now() 即完成存量回填,无需数据迁移。
ALTER TABLE activation_callbacks ADD COLUMN tried_at TIMESTAMPTZ NOT NULL DEFAULT now();
