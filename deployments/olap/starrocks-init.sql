/* StarRocks OLAP 初始化(阶段9 宽表;口径与 internal/domain/analytics/pg.go 一致)。
   数据由 PG 侧 ETL 定时写入(宽表只存派生聚合,不存业务基表)。
   执行: mysql -h <fe> -P 29030 -uroot < deployments/olap/starrocks-init.sql */

CREATE DATABASE IF NOT EXISTS boss_olap;

/* 区域投资一张图:收入(payments SUCCESS)/投资(扩容端口×单价) */
CREATE TABLE IF NOT EXISTS boss_olap.wide_region_roi (
  region_id   BIGINT NOT NULL COMMENT '区域ID',
  region_name VARCHAR(64) COMMENT '区域名',
  revenue     DOUBLE COMMENT '已收款',
  investment  DOUBLE COMMENT '扩容投资'
) PRIMARY KEY (region_id) DISTRIBUTED BY HASH (region_id);

/* 端口热力宽表:小区(4)/楼栋(5)端口利用率 */
CREATE TABLE IF NOT EXISTS boss_olap.wide_port_heat (
  address_id  BIGINT NOT NULL COMMENT '地址ID',
  name        VARCHAR(64) COMMENT '地址名',
  level       TINYINT COMMENT '4 小区 / 5 楼栋',
  ports_total BIGINT COMMENT '非 DISABLED 端口',
  ports_used  BIGINT COMMENT 'USED/RESERVED 端口'
) PRIMARY KEY (address_id) DISTRIBUTED BY HASH (address_id);

/* 设备健康宽表:维护一张表 + 资产健康度分子 */
CREATE TABLE IF NOT EXISTS boss_olap.wide_device_health (
  device_no    VARCHAR(64) NOT NULL COMMENT '设备号',
  device_type  VARCHAR(32) COMMENT '设备类型',
  health_score SMALLINT COMMENT '健康评分 0-100',
  fault_count  INT COMMENT '累计故障次数',
  age_years    DOUBLE COMMENT '投运年限',
  reason       VARCHAR(255) COMMENT '处置理由',
  priority     VARCHAR(16) COMMENT 'MUST_REPLACE/SUGGEST/WATCH'
) PRIMARY KEY (device_no) DISTRIBUTED BY HASH (device_no);

/* 订单转化宽表:单行汇总(DONE/有效订单/在服客户) */
CREATE TABLE IF NOT EXISTS boss_olap.wide_order_flow (
  id               TINYINT NOT NULL COMMENT '固定 1',
  orders_done      BIGINT COMMENT 'DONE 订单数',
  orders_effective BIGINT COMMENT '非 CANCELLED 订单数',
  active_customers BIGINT COMMENT '在服客户数'
) PRIMARY KEY (id) DISTRIBUTED BY HASH (id);
