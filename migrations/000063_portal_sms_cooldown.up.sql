-- 验证码发送冷却:同一 phone+scene 60 秒内只允许签发一次;issued_at 记录最近签发时间。
BEGIN;
ALTER TABLE portal_sms_codes ADD COLUMN issued_at TIMESTAMPTZ NOT NULL DEFAULT now();
COMMIT;
