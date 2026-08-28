# M3 渠道 CH 放量 · 验收证据与卡点发现

> 日期：2026-08-28｜对应 `docs/plan/q4-launch-growth-plan.md` 方向三 3.1 / 里程碑 M3（W7–W8 首段）
> 环境：102 `http://192.168.0.102:28080`（真实环境验证 + 真实 PG 集成测试，无 mock）

## 1. CH 域能力清点（已就绪，非重复建设）

| 能力 | 契约/实现 | 状态 |
|---|---|---|
| 入驻申请/审核/驳回 | `api/openapi/admin/partner.yaml` /applications + approve/reject | 已建 |
| 企业档案（me） | /partner/me（法人/企业名/联系人/审核时间） | 已建 |
| 员工管理 | /partner/staff + /staff/{id}/status | 已建 |
| 区域权限 | /partner/region-scope | 已建 |
| 渠道下单（共用直营 12 环节状态机） | /partner/orders（partner-order.yaml；PartnerOrder=true → submitPartnerAtomic 事务+风控+限额） | 已建 |
| 佣金台账 + 结算 | 000120 partner_commission_ledger；/partner/commissions + /{id}/settle（ACCRUED→SETTLED） | 已建 |
| 首单风控（FMS 前哨） | PartnerOrderRisk：DAILY_ORDER_LIMIT(100/法人/日) + 客户 24h 冷却 | 已建 |
| 审计 | partner_order.submit / blocked 留痕 | 已建 |
| 域测试 | pg_apply / pg_commission 集成测试（真实 PG，自清理） | **本批复跑全绿** |

## 2. 102 现状与真实链路验证（2026-08-28）

- partner_applications：7 单（5 APPROVED：Smoke/CDP Verify/**杭州米波网络科技**/CDP 步骤冒烟/E2E Partner 9832；2 REJECTED）。
- **佣金台账 partner_commission_ledger 为空**（尚无任何渠道订单达到终态计提）。
- API 冒烟：
  - `POST /api-keys`（account 主体 422，E2E Partner，name=m3-verify-partner-order）→ 签发成功（三主体绑定契约通）。
  - `GET /partner/me` → code:0，企业法人 15（E2E Partner 983299873000）鉴权链路通。
  - `POST /partner/orders`{customer 213, offer 101, address 3988, channel 104} → **42200**
    被法人归属校验拦截：`checkPartnerOwnership` 要求客户与产品必须在渠道法人(15)下，
    而 102 全部客户/产品属法人 1。

## 3. 卡点发现：渠道企业无自有目录，实际无法独立下单（放量红线）

- 现状：5 家 APPROVED 渠道企业的法人（7/8/9/15/21）下 **0 客户、0 产品、0 区域地址**；
  平台目录全部挂在法人 1。
- 实现语义：`checkPartnerOwnership`（pg_partner_fraud.go）强制渠道订单的 customer.offer
  属于渠道法人——即**渠道自营目录**设计。
- 后果：按当前实现，任何已批准渠道企业都无法通过 /partner/orders 下单，
  **M3 验收「渠道可独立下单跟踪」在 102 上当前不成立**，佣金台账将永远是空。
- 无设计中立 note 记载"渠道自营目录"为有意裁定（检索 notes 无命中）→ 属待裁定业务口径。

### 候选处置（交业务拍板，不阻塞其它里程碑）

| 选项 | 语义 | 放量影响 | 工量 |
|---|---|---|---|
| A 渠道自营目录（维持现状） | 渠道企业自带客户/产品/区域 | 需建目录初始化流程（引导/批量），起步慢 | 中（建数工具+引导） |
| B 纯平台共享 | 渠道只卖平台产品，订单法人落平台，channelId 记渠道 | 放量最快，但渠道"自有品牌/客群"无承载，风控边界松 | 小（改归属校验+对账口径） |
| C 混合（推荐） | 渠道可售平台产品（订单法人平台、渠道标识+佣金照计）；渠道自有目录作为扩展 | 放量与自营双轨，佣金/对账语义清晰 | 中（归属校验放宽+台账补充渠道法人字段） |

> 依据三年路线图 2028 Q1「渠道与直营订单共用 12 环节语义，数据域隔离有效，佣金可对账」——
> 数据域隔离与放量并不互斥，C 的"平台代售 + 渠道标识"是常见可落地形态。

## 4. M3 结论（本轮）

- CH 域能力与测试实测全绿（真实 PG 集成 2 项 PASS）；鉴权/风控/审计链路真实可走。
- 发现唯一真实卡点：渠道企业无自有目录 → 无法独立下单（§3），给出 A/B/C 三案与推荐（C）。
- 决策落地后（含启用渠道目录初始化或归属口径调整）再补"渠道下单 → 佣金计提 → settle"全链真实证据。

遗留：① 渠道目录归属口径待拍板（§3）；② 本批签发 account-422 测试 API key
（m3-verify-partner-order）留待续测，合入后可吊销。