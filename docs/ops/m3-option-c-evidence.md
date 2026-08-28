# M3 渠道 CH 放量 · C 案落地验收证据

> 日期：2026-08-28｜对应 `docs/ops/m3-channel-evidence.md` §3 C 案（推荐案）落地
> 环境：102 真实 API + 迁移 000162 已部署

## 1. C 案实现（commits cad05bb8 + 8285b6cb）

| 改动 | 说明 |
|---|---|
| `checkPartnerOwnership` 放宽 | 客户/产品属渠道法人 **或** 平台法人(1) 均放行 |
| orders 新增 `partner_entity_id`（迁移 000162） | 渠道下单时由 handler 从 partner profile 落入 |
| `accruePartnerCommission` COALESCE | 佣金归 `partner_entity_id`（C 案代售平台产品时），直营仍用 `legal_entity_id` |
| `submitPartnerAtomic` 清零 LegalEntityID | 渠道法人只做佣金归属，不参与地址归属冲突校验 |
| 决策记录 | `docs/notes/adopted/2026-08-28-partner-catalog-option-c.md` |

## 2. 102 真实链路验证（2026-08-28）

| 验证项 | 操作 | 结果 |
|---|---|---|
| C 案下单 | `POST /partner/orders`（customer 213/offer 101 均属平台法人 1，channel 104，partner key 法人 15） | **code:0**，ORD-20260828-000575 创建 |
| 双法人归属 | SQL 直查 orders | `legal_entity_id=6`（地址推导）、`partner_entity_id=15`（渠道法人）|
| 审计 | `GET audit_logs WHERE action='partner_order.submit'` | `{"channelId":104,"legalEntityId":15}` 留痕 |
| 佣金归属 SQL | `COALESCE(partner_entity_id, legal_entity_id)` | 代售平台产品时佣金归 15（渠道），非 6（平台） |
| 集成测试 | `TestPartnerCommissionLedgerIntegration`（真实 PG） | PASS |
| 清理 | 演练单已 cancel（造数不过夜） | CANCELLED |

## 3. M3 验收结论（更新）

- ✅ **渠道可独立下单跟踪**：C 案下渠道零建目录即可通过 `/partner/orders` 下单（使用平台共享产品），
  佣金归属渠道法人，审计完整留痕。
- ✅ **佣金结算可对账**：`partner_commission_ledger` 表 + `/partner/commissions` + `/{id}/settle` 全就绪，
  集成测试 PASS；佣金归 `partner_entity_id`（非地址法人），语义清晰。
- 渠道目录归属口径 **C 案已落地**（决策 note 已过账），M3 卡点解除。

## 4. 遗留

- 渠道订单推到终态（环节12）触发佣金 ACCRUED → settle 全链闭环：需自举端口/分光器等资源
  （mainchain-acceptance.sh 已有自举逻辑），列入放量运营阶段验证。
- worker 真机走查、老带新推荐码、话单轧账——同 `m5-acceptance.md` §5。