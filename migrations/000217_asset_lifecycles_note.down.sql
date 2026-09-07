-- 000217 down:删 note 列(修复痕迹随之消失;down 前无需业务前置)。

ALTER TABLE asset_lifecycles DROP COLUMN IF EXISTS note;
