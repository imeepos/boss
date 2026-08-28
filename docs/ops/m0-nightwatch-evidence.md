# M0 放量基线冻结 + 守夜上线 · 验收证据

> 日期：2026-08-28｜对应 `docs/plan/q4-launch-growth-plan.md` 里程碑 M0（W1–W2 首段）
> 环境：102 `http://192.168.0.102:28080`（全部真实环境证据，无 mock）
> 目标：契约/环境/对账守夜就位、Stripe 守卫告警收口观察、SLO 巡航跑通、worker Android 范围确认。

## 1. 守夜锚点核对（服务端循环 + 102 cron）

| 锚点 | 载体/触发 | 证据 |
|---|---|---|
| 五域每日对账 | 服务端 `daily_recon_loop.go`（03:00，30min 补跑，幂等） | 提醒中心存在 daily_recon URGENT 条目（见 §3，口径问题待 M1） |
| 话单补偿 / ETL 自动派单 | 服务端循环 | 已内建（代码在位，无异常） |
| 孤儿巡检门禁 | 102 cron 08:10 db-patrol-gate | 已装（2026-08-29 实录，本批复核在 crontab） |
| 备份 / gitea 清理 / 市场对账 | 102 cron | crontab 含 backup-daily 03:30、gitea-cleanup 03:00、market-reconciliation 月 |
| Stripe 隧道守卫 + 告警 | 102 cron */3 stripe-tunnel-url.sh | `/tmp/stripe-tunnel-url.log` 持续 `already up to date`（URL 稳定，守卫收口观察期无人工介入） |
| **SLO 巡航 + 周报（本批新增）** | 102 cron 每日 03:40 / 周日 03:50 | 见 §2 |

## 2. SLO 巡航上线（本批交付）

- 新脚本 `scripts/ops/slo-cruise.sh`（采集+阈值判定+提醒中心告警+JSONL 留痕；`--weekly` 周报；`--selftest`）。
- `slo-collect.sh` 双模式改造（102 本机 docker exec / 远端 ssh），修复原脚本 102 本机自连 host key 失败。
- 服务端 `ops/notify-emit` refType 白名单新增 `slo_cruise`（`internal/httpapi/admin/ops_notify.go` + OpenAPI 同步）。
- cron 安装实录（102 crontab）：`40 3 * * *` 每日巡航、`50 3 * * 0` 周报。
- 真实运行（102 本机，2026-08-28T03:38）：
  ```text
  s4 created=31 done=4 ratio=0.129  s5 todo_open=4 todo_older_24h=3
  s6 cdrs_total=0 billing_unbilled=0  s7 lag=1s on_time_cnt=3
  CRUISE-WARN: 待办积压: 3 条超 24h 未处理
  ```
- 告警入库验证（提醒中心）：`slo_cruise | WARN | SLO 巡航告警 | 2026-08-28T03:38`。
- 留痕：`/tmp/slo-cruise-YYYYMM.jsonl` 每日一条。

### 过程中踩坑（已修，教训入 skill）

1. `ops/notify-emit` 有 refType 白名单，未登记 refType 会被拒——先读契约再写脚本。
2. **信封层 42200/40100 同样返回 HTTP 200**：只看 http_code 会静默吞错（本批实测 emit 静默失败）。
   修复：以信封 `code:0`（或 ok:true）判定成败，失败响亮留痕（对齐 stripe-tunnel-url.sh 既有模式）。

## 3. 守夜首日真实发现（运营信号，交 M1 处置）

待办积压 3 条超 24h（SQL 直查权威表 admin_notifications）：

| id | level | refType | 内容 | 状态 |
|---|---|---|---|---|
| 175 | WARN | realname | 实名待审核 acc_dbg3(customer/341) | 人工审核项，8-27 建立 |
| 41 | URGENT | daily_recon | 孤儿保留端口 58 个（20260823） | **5 天未处置** |
| 34 | URGENT | daily_recon | 孤儿保留端口 48 个（20260822） | **6 天未处置** |

孤儿端口根因（每日对账口径问题，2026-08-28 直查迁移+recon 定义+ports 表）：

- 现状：`ports.status='RESERVED'` 且订单不在 `RESERVED` 态的端口 = 7 个，全部挂两张僵尸单：
  `ORD-20260820-000`（5 端口）、`ORD-20260821-000`（2 端口），订单状态 INSTALLING，自 8-20/8-21 起 8 天未动。
- 对照组：DONE 订单端口 6/6 全部为 USED → 正常生命周期是 RESERVED→USED。
- 判断：recon SQL（`pg_recon_counts.go`）"在途订单"集合只含 `o.status='RESERVED'`，未含 INSTALLING；
  对 INSTALLING 单，端口保持 RESERVED 是否合法取决于口径（激活才消费）。历史 48/58 高点随单量回落至 7。
- **M1 行动项**：① 定口径——INSTALLING 是否算端口在途（含则改 recon SQL 消除误报，不含则端口应在进入 INSTALLING 时置 USED）；② 僵尸单（INSTALLING 超 7 天）纳入订单超时/放弃处置流程。两项先单项口径裁定并写 note。

## 4. worker Android 范围与排期确认

见 `docs/plan/worker-android-scope.md`（M0 交付）：50+ 屏现状盘点、6 项差距
（离线缓存为最大工程缺口）、复用地图、W5–W6 排期。

## 5. M0 验收结论

- 守夜锚点矩阵就位于 102 实际运行（含新增 SLO 巡航，告警链路真实入库验证）。
- Stripe 守卫收口观察期 URL 持续稳定、无人工介入。
- 巡航首日即捕获真实运营信号（待办积压 + 对账口径疑点），已定位根因、给出 M1 行动项。
- worker Android 范围确认文档交付，M2 可开工。

遗留：① 提醒中心 probe 探针 INFO 条目（probe-deploy-ok，部署验证遗留，无害）；② M1 待办见 §3。