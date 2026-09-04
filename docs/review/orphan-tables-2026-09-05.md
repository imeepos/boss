# 留存未用表评估（9 张）——2026-09-05 机械对账

> 来源：2026-09-04 表数量对账纪要行动项 2（10 张停用孤儿表评估出清或标注）。
> 本文为评估与标注，不含 DDL；出清（DROP 迁移）需产品确认后另行立项。

## 对账方法（可复跑）

1. 库存：102 `boss-infra-postgres-1` boss 库 `information_schema.tables`，public BASE TABLE 共 199；
   剔 `audit_logs_*` 6 张分区子表、`schema_migrations`、`spatial_ref_sys` = 净业务表 191。
2. 代码：对每张表在 `internal`+`cmd` 生产 Go（排除 *_test.go）grep SQL 关键字上下文
   `(from|join|into|update|table) + 表名`，零命中即「留存未用」。
3. 迁移链：196 个唯一 CREATE TABLE − 3 张链内已 DROP（real_name 三兄弟，000059）= 193 留存，与库存逐表吻合，无缺失。

## 纪要口径修正（重要）

- 纪要中「10 张真孤儿 = 6 user_* + real_name 三兄弟 + service_metric_snapshots」有误：real_name 三兄弟
  （real_name_verifications / worker_real_name_verifications / customer_real_name_verifications）已在 000059
  链内 DROP（即 192−3 的那 3 张），不在留存集。真实留存未用为 **9 张**（下表）。
- 000181 的 monthly_* 4 表在 Go 生产代码有读写（internal/domain/monthly），非孤儿。
- 对外报数净口径以 data-relations V1.5 为准：**191**。

## 9 张留存未用清单

| 表 | 来源迁移 | 停用证据 | 处置建议 |
|---|---|---|---|
| user_balances | 000046 | 裁定 D1：余额权威态在 portal_wallets（pg_actions.go Deprecated 注释） | 标注保留；随 userdata 域出清迁移一并 DROP |
| user_notify_settings | 000046 | 裁定 D1：走 portal_prefs.notify | 同上 |
| user_messages | 000046 | 裁定 D1：走 portal_messages | 同上 |
| user_invoices | 000046 | 裁定 D1：开票走 billing 域，停写 | 同上 |
| user_complaints | 000046 | 裁定 D1：走 complaints | 同上 |
| user_verify_records | 000046 | 000059 归一：无写入方，读走 verifications（pg_lists/pg_aggregate 注释） | 同上 |
| service_metric_snapshots | 000118 | 生产 Go 零 SQL 引用 | 确认无 BI 消费后随出清迁移 DROP |
| odn_region_code | 000075 | 生产 Go 零读写，行数据仅迁移种子供给（odn_grid 等经复合 FK 引用其行，表本身不被代码触碰） | 保留（ODN 主数据，种子依赖） |
| odn_city_code | 000075 | 同上 | 保留（ODN 主数据，种子依赖） |

## 出清前置条件（将来时，不在本次执行）

1. 六张 user_*：新独立迁移 `DROP TABLE IF EXISTS ...`，先核对无视图/物化视图/外部报表依赖；
2. service_metric_snapshots：与 QA/BI 确认 000118 看板不再读后执行；
3. odn 两表因被同域复合 FK 链引用，随 ODN 域改造一并评估，不单独 DROP。
