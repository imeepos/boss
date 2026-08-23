-- 配置模板自定义：参数内容、版本、启停和编辑留痕。
BEGIN;
ALTER TABLE provision_templates ADD COLUMN IF NOT EXISTS content JSONB NOT NULL DEFAULT '{}'::jsonb;
ALTER TABLE provision_templates ADD COLUMN IF NOT EXISTS version INTEGER NOT NULL DEFAULT 1;
ALTER TABLE provision_templates ADD COLUMN IF NOT EXISTS status VARCHAR(16) NOT NULL DEFAULT 'ENABLED';
ALTER TABLE provision_templates ADD COLUMN IF NOT EXISTS updated_at TIMESTAMPTZ NOT NULL DEFAULT now();
CREATE INDEX IF NOT EXISTS idx_provision_templates_status ON provision_templates(status, legal_entity_id);
COMMIT;
