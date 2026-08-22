# SLO/SLI 定义（稳定版 1.0）

> 版本 V1.0｜环境：102 `http://192.168.0.102:28080`｜度量源：服务端 `/metrics`（Prometheus，`internal/pkg/server/metrics.go`）+ 数据库聚合。
> 原则：只定义有真实采集手段的指标；采集不到的标"待埋点"，不虚报达标。

## 1. 度量基础设施现状（已验证）

- 服务端暴露 `/metrics`（RED 模型），指标：
  - `boss_http_requests_total{method,path,code}` — 请求计数
  - `boss_http_request_duration_seconds{method,path}` — 时延直方图（bucket 5ms…2.5s）
- `/healthz` 存活探针。
- 102 环境实测：两个端点均 200 可采（2026-08 验证）。
- 2026-08-26 修复:Prometheus 与 boss-server 跨 compose 网络互不解析,抓取目标改为宿主机 `192.168.0.102:28080`,实测 up=1、流量入库、SLO 告警规则生效(BossServerDown/S1/S2)。

## 2. SLI 与 SLO 总表

| # | 能力 | SLI（怎么算） | SLO 目标 | 采集源 | 状态 |
|:-:|:-----|:--------------|:---------|:-------|:----:|
| S1 | API 可用性 | 非 5xx 响应占比（30d 窗口） | ≥ 99.5% | `boss_http_requests_total` 按 code 聚合 | 可采 |
| S2 | API 时延 | 核心读接口 P95 < 500ms（30d） | ≥ 99% 请求达标 | `boss_http_request_duration_seconds` | 可采 |
| S3 | 认证成功率 | 登录+API key 认证 2xx 占比 | ≥ 99.5% | `/auth/login`、`X-API-Key` 路径计数 | 可采 |
| S4 | 订单完成率 | 终态=完成 的订单 / 应完成订单（周窗口） | ≥ 95% | DB 聚合 orders.status | 需报表查询 |
| S5 | 消息延迟 | 通知入队→送达 P95 | ≤ 5min | 通知表 created/delivered 时间差 | 需 DB 聚合脚本 |
| S6 | 话单完整率 | 话单流水 / 应产话单（对账口径） | ≥ 99.9% | Q3 台账对账脚本 | 部分可采 |
| S7 | 报表时效 | 日报表产出时间点 | 每日 08:00 前 | 报表任务日志/时间戳 | 待埋点 |

## 3. 指标→SLO 推导（PromQL 速查）

```promql
# S1 API 可用性（30d）
sum(rate(boss_http_requests_total{code!~"5.."}[30d]))
/ sum(rate(boss_http_requests_total[30d]))

# S2 时延达标率（核心 GET 接口 P95 口径近似：≤0.5s 桶占比）
sum(rate(boss_http_request_duration_seconds_bucket{le="0.5"}[30d]))
/ sum(rate(boss_http_request_duration_seconds_bucket{le="+Inf"}[30d]))

# S3 认证成功率
sum(rate(boss_http_requests_total{path=~".*/auth/login",code="200"}[30d]))
/ sum(rate(boss_http_requests_total{path=~".*/auth/login"}[30d]))
```

## 4. 错误预算与响应动作

- 月度错误预算 = 1 − SLO（S1/S3 为 0.5%，S2 为 1%）。
- 预算耗尽 50%：发布列车降速（只允许修复类变更，见 `release-train.md` §灰度门禁）。
- 预算耗尽 100%：冻结功能发布，只留 P1 修复；触发复盘（`docs/postmortem/`）。

## 5. 待办（进入容量/演练阶段前补齐）

- [x] S5 消息延迟 DB 聚合脚本（`scripts/ops/slo-collect.sh`，含 S4/S5/S7，基线见 `slo-baseline-102.md`）
- [ ] S6 话单对账结果定时落表，可查询历史完整率
- [x] S7 报表产出时间采集（并入 `slo-collect.sh`：窗口结束 8h 内为按时）
- [x] Prometheus 抓取配置（102 已修复跨网络抓取并验证 up=1,见 prometheus.yml 注释）+ SLO 告警规则 slo-rules.yml
- [ ] Grafana 看板 JSON（导入即用,S1-S3 曲线）
