-- 订单预缴月数快照 + 赠送时长回填(促销体系:预付费环节4 按合约期月数收款)。
-- buy_months: 下单客户选定预缴月数;0=按月缴(环节4 只收 1 个月月费,维持原口径)。
-- gift_months: 环节4 收款时按 gift_duration_rules 阶梯命中的赠送月数,回填快照。

ALTER TABLE orders
    ADD COLUMN buy_months INT NOT NULL DEFAULT 0 CHECK (buy_months BETWEEN 0 AND 60),
    ADD COLUMN gift_months INT NOT NULL DEFAULT 0 CHECK (gift_months BETWEEN 0 AND 60);
