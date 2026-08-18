# 上线交接清单（W12）

> 适用:BOSS 模块化单体(W1–W12 完成态)。权威契约:docs/contract/{terms,domain-map,fields}.md。

## 一、部署物

| 组件 | 镜像/入口 | 副本 | 说明 |
|---|---|---|---|
| server | `boss-server`(HTTP :8080 + gRPC :9090) | 2–6(HPA cpu70%) | /healthz 探针;/metrics 供 Prometheus |
| collector | `boss-collector` | 1 | SNMP 采集(BOSS_SNMP_TARGETS=code@host) |
| provisioner | `boss-provisioner` | 1 | Telnet 下发(BOSS_PROVISION_OLT_*) |
| gis | `boss-gis` | 1 | 消费 boss-order-events 环节12 同步 |
| report | `boss-report` | 1 | 周期报告(BOSS_REPORT_PERIOD) |

Helm:`deployments/helm/boss`(`helm lint`/`helm template` 通过后 `helm upgrade --install`)。
DSN 等敏感项外部注入,不进 values。

## 二、上线前门禁(全部必须绿)

1. `make check`(单测+race+vet+gofmt+build)。
2. 真实 PG 集成:`BOSS_PG_TEST_DSN=... make check`(全量含 e2e)。
3. 压测:`BOSS_DATABASE_DSN=... make load`(读 p95<500ms、错误率<1%;本仓实测 p95=8.55ms)。
4. 迁移前置检查:`migrate -path migrations -database $DSN up` 幂等可重放。

## 三、回滚预案(已演练)

- 应用层:镜像 tag 回退,`helm rollback`(Deployment 滚动)。
- 迁移层:逐版本 `migrate ... down 1`;000034(report_snapshots)已演练 down→up 全绿,仅快照留痕,业务零影响。
- 数据层:PG 快照恢复;Kafka 事件可重放(consumer group 断点续传)。

## 四、可观测

- 指标:/metrics(RED:boss_http_requests_total / boss_http_request_duration_seconds)。
- 部署:deployments/observability(prometheus/loki/promtail/alertmanager)。
- 链路:应用侧 trace-id 中间件贯穿(响应头 X-Trace-Id);OTel→Jaeger 导出器属部署层接线,接入点已留在 server 装配层。

## 五、已知边界(如实)

- OLT 协议为通用适配层(Telnet 行协议 + SNMP OID 可配),厂商 profile 随环境配置。
- OLAP(StarRocks/Doris)DSN 预留(config.OLAP),当前分析域直查 PG 派生聚合。
- GIS 瓦片/三维底座(Cesium 自托管)属前端工程,后端八级下钻/详情/热力数据已就绪。
