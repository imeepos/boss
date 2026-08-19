-- 订单服务评价落库(用户端 /orders/{orderNo}/rate)。取代 handler 进程内 map。
-- 一单一评(UNIQUE order_no);星级 1~5 与契约校验对齐。
BEGIN;

CREATE TABLE order_ratings (
    id          BIGSERIAL PRIMARY KEY,
    order_no    VARCHAR(32) NOT NULL UNIQUE,
    customer_id BIGINT      NOT NULL REFERENCES customers (id),
    stars       SMALLINT    NOT NULL CHECK (stars BETWEEN 1 AND 5),
    attitude    SMALLINT    NOT NULL CHECK (attitude BETWEEN 1 AND 5),
    quality     SMALLINT    NOT NULL CHECK (quality BETWEEN 1 AND 5),
    comment     TEXT        NOT NULL DEFAULT '',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT now()
);
CREATE INDEX idx_order_ratings_customer ON order_ratings(customer_id);

COMMIT;