# 充值余额语义裁定:预存(只进不出,非"可用余额")

日期:2026-09-03

## 1. 余额的当前形态与含义

**裁定**:`portal_wallets.balance` 是**预存**(stored value),不是"可用余额"(available balance)。
- 当前口径:仅 SUCCESS 充值入账(RecordTopup 同事务 upsert 余额,adopted note 2026-08-26);
- 当前无任何消费路径(`AdjustBalance(ctx, cid, delta)` 接口签名虽允许负 delta,但
  `bills`/`payments`/`orders` 落账链路均未调它扣减余额——grep 实证);
- 用户端 `GET /api/user/v1/topups` 仅查余额 + 充值档位(50/100/200),无"消费/抵扣"入口;
- admin 端无任何扣减门户余额的后台按钮/接口。

## 2. 为什么不叫"可用余额"

"可用余额"语义要求:
- 预存可被消费(缴费/下单抵扣/购买附加包);
- 消费后余额减少,记账流水同步落;
- 有冻结/解冻/退款回滚等机制。

我们当前没有这套机制,即便叫"可用余额"也是名不副实。落到用户文案上,
"预存"更准确——钱进来了,要花得另外走缴费流程。

## 3. 未来接消费的口径(预防踩坑)

若未来要打通"余额可抵扣账单/下单":
- 必加 `portal_wallet_transactions`(customer_id, delta, ref_type, ref_id, balance_after, created_at),
  充值/消费/退款三类流水,负 delta 即消费;
- 扣减走 `AdjustBalance(cid, -amount)` 同事务 INSERT 一条 `kind=CONSUME` 流水;
- 退款走 `AdjustBalance(cid, +amount)` 同事务 INSERT `kind=REFUND` 流水(余额正向,
  但 payments.status=REFUNDED,术语严格区分"通道退款"与"余额回补");
- 数据库层 `balance >= 0` CHECK 约束,避免负余额透支;
- 余额不足以全额抵扣账单时,差额走普通账单缴费路径(部分抵扣,留痕)。

## 4. 关联

- 2026-08-26-payment-hardening.md §1(RecordTopup 原子入账);
- `internal/domain/portal/portal.go::Service.Balance/AdjustBalance`(接口契约);
- `migrations/000052_portal_state.up.sql::portal_wallets`(表结构)。

## 5. 放弃了什么

- **不改代码,只裁语义**:本期不实现余额消费,仅把"预存"含义落契约并留 adopted note。
  任何调用方在文案/接口文档里继续用"余额"也 OK——它是"预存"的简化俗称,
  但权威契约是"预存",新接口/新页面应优先使用"余额/预存"双标。
- **不动既有 AdjustBalance 接口签名**:虽然语义上"只进不出",但接口接受负 delta
  是预留扩展点(未来消费实现时不必改接口,只补流水表),且 `bills/payments`
  未扣余额 = 当前已满足"只进不出"现实行为。
- **不引入 wallet_transactions 表**:消费路径未立项,空表无意义;裁定点
  §3 描述了未来加表的结构,留给真正立项时建表 + 迁移。
- **不写 demo 数据修正**:既有 `portal_demo_seed` 余额是 demo 注入,
  102 上 0 行真实数据,口径调整无需迁移。