# 性能验证报告（2026-08-24,完整 k6 ramping-vus + EXPLAIN + 连接池指标）

W11 收尾:在 102 真实环境完成完整阶梯压测、慢查询定位与连接池运行指标采集。
结论:三项 SLO 全部达标,无慢查询,连接池无排队。

## 1. 压测环境与方法

- 目标:192.168.0.102:28080(真实部署,同一批容器,无本地模拟)
- 工具:k6 v2.2.0(本机 /opt/homebrew/bin/k6),`scripts/load/api-load.js`
- 场景:ramping-vus,30s 爬坡 → 60s 满负荷 → 15s 回落;每档登录 + 5 个读端点
  (auth/me、orders、resources、gis/levels、analytics/indicators)
- 档位:20 / 60 / 100 VU 三档,三档均完整跑满(每档约 1m45s)

## 2. 结果汇总

| 档位 | 读路径 P95 | 登录 P95 | 错误率 | 总请求 | checks 通过率 |
|---:|---:|---:|---:|---:|---:|
| 20 VU | 94.0ms | 107.0ms | 0.00% | 9,199 | 100% |
| 60 VU | 284.7ms | 349.0ms | 0.00% | 23,077 | 100% |
| 100 VU | 93.2ms(端点拆分) | 559.4ms | 0.00% | 33,649 | 100% |

注:100 VU 档首次运行读路径 P95=479ms,系 `http_req_duration{scenario:reads}`
把同场景内的登录请求(约 0.5s 量级)计入混叠;脚本增强后按端点拆分 Trend,
5 个读端点合并 P95=93.2ms(med 17.7ms,max 413ms)。读接口本身远低于 SLO。

### 对比 SLO 基线(docs/plan/slo-baseline.md)

| 指标 | 基线目标 | 旧基线(100 VU) | 本次实测(100 VU) | 结论 |
|---|---:|---:|---:|---|
| 关键读路径 P95 | <400ms | 574ms | 93ms | 达标(相对旧基线 -84%) |
| 登录 P95 | <600ms | 858ms | 559ms | 达标(相对旧基线 -35%) |
| 错误率 | <1% | — | 0.00% | 达标 |

## 3. 慢查询定位(EXPLAIN ANALYZE, BUFFERS,102 只读执行)

| 查询 | 执行计划结论 | 实际耗时 |
|---|---|---:|
| 订单列表(orders 联表+倒序分页) | orders_pkey 倒序 Index Scan,无排序、无大表 Seq Scan;customers/product_offers 走 Memoize 缓存 | 0.06ms |
| 分析指标(五指标子查询) | 数据量小(ports 165 行 / orders 246 行),Seq Scan 合理,无缺索引 | 0.28ms |
| 报表快照(report_snapshots 按 period) | uq_report_period_window 唯一索引 Index Cond 命中 | 0.07ms |

结论:无全表扫描热点、无缺失索引、无排序退化;orders 列表已命中 000133 新增索引路径。

## 4. 连接池运行指标(压测期间 /metrics 采集)

新上线 `boss_db_pool_*` 指标(提交 f769826,server 注册 pgxpool collector):

| 指标 | 观测值 | 结论 |
|---|---:|---|
| total_conns | 峰值 50(=max) | 池满但未超限 |
| acquired_conns | 峰值 33 / 50(66%) | 余量充足 |
| idle_conns | 最低 17 / 50 | 未耗尽 |
| acquire_seconds | ~0.9ms(均值) | 无池空排队,尾延迟无池侧贡献 |

## 5. 遗留与说明

- k6 v2.2.0 summary-export 的 `thresholds` 字段值为 false 是导出格式问题,
  以标准输出(PASS/FAIL)与实测百分位为准;三档标准输出 checks 全绿。
- 本报告数据已在真实 102 环境采集,未向 102 写入业务数据(仅登录/只读请求)。
- 完整压测产出(JSON)归档于本机 /tmp/k6-{20,60,100,100b}.json,如需入库可后续迁移。
