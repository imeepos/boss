# M4 营销活动 + 首单风控 · 验收证据

> 日期：2026-08-28｜对应 `docs/plan/q4-launch-growth-plan.md` 方向三 3.2/3.3 / 里程碑 M4（W9–W10 首段）
> 环境：102 `http://192.168.0.102:28080`（全部真实 API + 真实数据，无 mock）
> 测试主体：客户 213（13900001234，customer 主体 API key）；管理端 ops-main key。

## 1. 活动上线（3.2 验收：至少 2 个活动）

| 活动 | 配置 | 模板 id | 状态 |
|---|---|---|---|
| 首单立减 50 元 | scopeType=FIRST_ORDER, CASH 5000 分, 每人限 1, 30 天, 限量 100 | 7 | ENABLED |
| 积分兑换 30 元券 | pointsPrice=1000, CASH 3000 分, 限量 50, 30 天 | 8 | ENABLED |

LOY 配套（此前为空，本批配置）：等级 白银(500)/黄金(2000)、任务 完善资料(200/ONE_TIME)+每日登录(10/DAILY)。

## 2. 全链真实演练（发放→兑换→核销→对账）

1. **发放**：admin `POST /coupon-templates/7/issue`{customerIds:[213]} → issued:1。
2. **积分种子**：admin `POST /points/213/adjust` +1000（reason 留痕）→ balance 1000。
3. **客户视角**：`GET /user/v1/points` 流水可追（entryId/afterBalance/reason）；`GET /points/exchange-offers` 正确返回 T2。
4. **兑换**：`POST /user/v1/points/exchange`{templateId:8} → cost:1000, couponId=CPN-f14e71f1a04c；
   余额 1000→0，流水 EXCHANGE -1000（先扣后发，失败补偿回补契约在域测试覆盖）。
5. **核销**：`POST /user/v1/payments`{billNo:BILL-E2E-TAXJUR-001, amount:999, cash, couponId} →
   **实收 969，抵扣 3000 分**，PAY-30 SUCCESS（核销与落账同事务）。
6. **对账双端点全 MATCH**：
   - /coupon-recon：T1 ISSUED 1；T2 **USED 1 / redeemed 1 次 3000 分**；diffKind=MATCH
   - /loy/points-recon：客户 213 balance=0=entriesSum，byReason 可读，MATCH
7. **等级匹配**：`GET /points/tier` → 白银会员（lifetimeEarn 1000 ≥ 500）正确命中。

## 3. 3.2 验收结论

- 券/积分发放-兑换-核销-补偿-对账全链真实闭环，两个活动上线 → **3.2 达成**。
- 官网落地页/转化埋点（3.4）与老带新推荐码未含在本批（见 §5 遗留）。

## 4. 3.3 首单风控现状

- 渠道链路风控已在跑（M3 证据）：PartnerOrderRisk 日限额(100/法人/日)+客户 24h 冷却+审计留痕。
- 直营下单风控（同号多单/异常地址）尚未接直营 Submit 路径——列入下批代码项：
  规则引擎挂 order.SubmitRegular（biz_params 可配阈值 + audit 拦截留痕），与 3.3 验收对齐。

## 5. 遗留（下轮）

1. 3.3 直营首单风控规则 v1 落码（同号多单/异常地址，biz_params 配置 + 审计）。
2. 3.4 官网获客闭环：CMS 活动 → 注册转化追踪（utm/来源字段）。
3. 老带新推荐码活动（需推荐关系建模，排 M4 后段）。
4. M3 渠道目录归属口径 A/B/C 仍待拍板（`m3-channel-evidence.md` §3）。