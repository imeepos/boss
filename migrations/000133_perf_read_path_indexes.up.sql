-- 读路径覆盖索引：订单列表、分析聚合和报表关联。
-- 仅增加可并发创建的索引，不改变业务数据。
CREATE INDEX idx_orders_status_id_desc
    ON orders (status, id DESC);
CREATE INDEX idx_orders_customer_id_desc
    ON orders (customer_id, id DESC);
CREATE INDEX idx_ports_status ON ports (status);
CREATE INDEX idx_payments_status_bill ON payments (status, bill_id);
CREATE INDEX idx_expansions_region ON expansions (region_id);
CREATE INDEX idx_device_maintenances_priority_health
    ON device_maintenances (priority, health_score, fault_count DESC, age_years DESC);
