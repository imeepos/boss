# M1 真实支付/缴费闭环加固 · 验收证据

> 日期：2026-08-28｜对应 `docs/plan/q4-launch-growth-plan.md` 里程碑 M1（W3–W4 首段）
> 环境：102 `http://192.168.0.102:28080`（全部真实环境证据，无 mock）
> 内容：M0 行动项闭环（对账口径裁定 + 僵尸单处置）、支付退款全链路演练、话单轧账机制确认、实名/税务评估。

## 1. 对账口径裁定（M0 行动项①，代码+决策双落地）

- 裁定：端口在环节 12（UpdateMap）才消费（`pg_workflow.go` 实证 `UPDATE ports SET status='USED'`；
  102 对照 DONE 单端口 6/6 全 USED）。INSTALLING 单端口保持 RESERVED 属合法在途。
- 代码：`pg_recon_counts.go` — orphanReservedPorts 在途集合扩为 `RESERVED+INSTALLING`；
  新增 `installingStuck`（INSTALLING 超 7 天），把"装维卡死"信号从误报孤儿里拆出。
- 测试：recon 形状测试 10→11 项，断言按 Name 定位防索引错位。
- 决策记录：`docs/notes/adopted/2026-08-28-port-reserved-installing-inflight.md`；
  8-25 审计"是(314)"按代谢规则加 Amended。
- 部署：合入 main 后 102 自动部署（boss-server 重启 6 分钟前实录）。

## 2. 僵尸单处置（M0 行动项②）

- 7 张 INSTALLING 僵尸单全属测试客户 13900001234（采购经理·王，E2E 残留，8-20/21 卡死 8 天）。
- 处置：走正式 `POST /orders/{orderNo}/cancel`（7 单全 CANCELLED，审计留痕，禁 SQL 直改）。
- 结果 SQL 直查：旧口径孤儿 58→0、新口径孤儿 0、installingStuck 0。
- 历史 daily_recon URGENT（id 34/41）为审计留痕保留；后续 03:00 对账无新异常（当日已全 0）。

## 3. 支付退款全链路演练（M1 1.2）

- 对象：测试客户 213 现金流水 PAY-E2E-RESUME-003（999.00，SUCCESS，挂 BILL-E2E-TAXJUR-001）。
- 动作：admin `POST /payments/5/refund`（reason=M1 支付退款全链路演练-测试数据）。
- 核验（SQL 直查 + API 回显）：
  - 支付 SUCCESS → REFUNDED，refunded_at + refund_reason 落库；
  - 账单 PAID → UNPAID（全额退款回账）；
  - 审计留痕 `payment.refund | payment | PAY-E2E-RESUME-003`（含 reason）；
  - `ledger-recon?period=2026-08` 回显 paidAmount=0，无 DIFF_PENDING 残留。

## 4. 话单轧账机制确认（M1 1.3，空态正常）

- 现状：102 无 collector/provisioner 容器、无真实 OLT，AAA 容器在跑但 cdrs=0（无真实 RADIUS 流量）。
- 机制：slo-cruise s6 每日采集话单（阈值 SLO_UNBILLED_ALERT=100 防空态误报）+ daily_recon 账单侧；
  真实话单接入（设备+采集器）属 P3（`remaining-gaps-priorities.md` §4.2 依赖真实设备数据）。
- 结论：轧账日报机制就位并每日运行；有效输出随真实设备接入而来，空态不误报。

## 5. 真实实名/税务评估（M1 1.4）

- 真实实名（阿里云 CloudAuth）：102 `app.env` 无 REALID 凭据 → 试跑阻塞于真实资质 KEY；
  通道（realname 域 + realid channel，2026-08-24 接线）已就绪，拿到凭据即可联调。
- 税务：invoices 实务 2 张（INV-00000001/06，系统票 ISSUED）；无税局资质/回执通道配置 →
  维持 2026-08-18"系统发票无独立法定效力 + 多属地网关并存"姿态；乐企/BIR eIS 资质到位后接
  invoice_tax_events 回执链路（000112 已备表）。
- 结论：两项均阻塞于外部资质，不阻塞主线；M3 渠道、M2 装维不受影响。

## 6. M1 验收

- 对账口径修正 + 僵尸单清零 + 支付退款全链路真实演练通过（差异=0 全程可回放）。
- 遗留 1.1 Stripe 守卫连续观察（cron */3 在跑、URL 稳定）；1.3/1.4 机制就位、资质阻塞如实记录。
- 下一里程碑 M2：worker Android 收口首发（见 `docs/plan/worker-android-scope.md`）。