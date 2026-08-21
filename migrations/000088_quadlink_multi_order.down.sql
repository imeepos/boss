BEGIN;

DROP INDEX IF EXISTS idx_quad_links_customer;

CREATE UNIQUE INDEX uq_quad_links_customer
    ON quad_links (customer_id) WHERE customer_id IS NOT NULL;

COMMIT;