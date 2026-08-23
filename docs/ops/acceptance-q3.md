# Q3 财务、税务与交付标准化 · 季度验收报告

日期:2026-08-23
核验环境:102 部署环境(http://192.168.0.102:28080,migrations ≤000116)
结论:**三条核心验收全部通过**。

## 验收① 账实差异可定位

- `GET /billing/ledger-recon?period=2026-08`:按账单粒度返回 应收/实收/开票/退款 四金额 +
  `diffKind` 分类(MATCH/PARTIAL/REFUNDED/OVERPAID/…),汇总含 byKind 与三总额。
- 实测:账单 BILL-E2E-TAXJUR-001 → 999/999/1118.88/1998,MATCH,附 invoiceNo+taxStatus。
- 渠道侧对账(正交):`/reconciliations` 批次+行级 items 差异定位,Stripe 源自动拉单已接。

## 验收② 发票状态与税局状态正交、可追踪

- 正交实测:INV-00000001 `status=ISSUED`(业务)与 `taxStatus=PENDING`(税务)并存,
  属地快照 `juris=CN`(000109 法人配置→开票快照链)。
- 迁移实测:`POST /invoices/:id/tax-backfill` → taxStatus PENDING→ISSUED,法定票号回填
  24122000000012345678,业务状态不变。
- 追踪实测:`GET /invoices/:id/tax-events` 轨迹行 BACKFILL(taxStatusAfter/taxNo/操作人/时间)。
- 全链:开票(占号连续)→网关提交/人工回填→作废/重开(编号保留)→轨迹留痕(000112 invoice_tax_events)。

## 验收③ 新环境可按清单完成部署和初始化

清单:`docs/deploy/new-env-init-checklist.md`(九步,逐步验证命令+通过标准+签核表)。
在 102 复跑关键步骤:

| 步骤 | 结果 |
|---|---|
| 3 迁移核验 | schema_migrations 117 行 vs 仓库 116 个 up 文件(差 1 为历史撞号存量 000044 例外,已登记) |
| 4 超管+权限码 | role=sysadmin,permissionCodes=59,menu:* 种子齐 |
| 5 基础数据层序 | regions=11 / legal_entities=9 / product_offers=22,无孤儿 |
| 7 冒烟三接口 | aaa/summary、billing/ledger-recon、reconciliations 均 200 |

## 四个重点完成对照

| 重点 | 落地 |
|---|---|
| 发票/税局/法定票号/作废/重试/属地 | ARN 连续发号、void/reissue、tax-submit 重试、backfill、法人属地配置(000109)、轨迹(000112) |
| 预付费/后付费/退款/欠费/停复机/收款渠道 | billing_mode(000103)、全额退款(000113)、dunning 自动停机、缴费自动复机、Stripe 源自动对账 |
| 租户/法人/区域隔离模板 | 清单 §6 数据隔离验证步骤 + region_scope SQL 层裁剪(AAA/账实核对已接) |
| Admin 设计系统统一 | page-patterns.md 四类页面模式 + 模式件全量覆盖,12 模块采用率 100%,门禁 80% 目标位 |

## 已知边界(不阻塞验收)

- CN 乐企 / PH BIR eIS 网关适配器待外部资质与密钥,当前走 manual 人工回填通道(设计内路径)。
- 渠道对账的微信/支付宝自动拉单未接(Stripe 已接),manual 渠道建批+手工录入。
