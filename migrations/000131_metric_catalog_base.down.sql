-- 指标目录与数据质量规则基础回滚
DROP INDEX IF EXISTS idx_metric_catalog_status;
DROP INDEX IF EXISTS idx_metric_catalog_owner;
DROP INDEX IF EXISTS idx_metric_catalog_key;
DROP TABLE IF EXISTS metric_quality_rules;
DROP TABLE IF EXISTS metric_catalog;