-- 用户端产品分类(契约 api/openapi/user/product.yaml category 枚举 broadband/fusion/addon)。
-- 历史产品全部回退 broadband。
BEGIN;

ALTER TABLE product_offers
    ADD COLUMN category VARCHAR(16) NOT NULL DEFAULT 'broadband';

COMMIT;
