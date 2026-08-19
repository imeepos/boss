# 发票税务属地裁定:多属地网关并存,系统发票无独立法定效力

日期：2026-08-18

## 决策

发票法定效力层做成**多属地税局网关**，CN（数电票，经电子发票服务平台/乐企）与 PH（BIR eIS）并存，按 `tax_jurisdiction` 选择通道；本期两条通道都未接外部接口（资质/密钥未就绪），落地为 `manual` 人工通道：运营者在对应税局平台开具后回填税局票号（`tax_no`）。`invoices` 表内生成的发票一律定义为"开票意图 + 内部留痕"，`invoice_no`（原 ARN）为**内部流水号**；法定票号以税局回执回填为准。

## why

- 自建系统开具的发票无税控签名/税局回执即无法律效力，普票同样；这是法定要求，不是工程选型。
- 平台为多品牌/多子公司架构（legal_entities），各主体可能注册在不同税管辖区；全局单属地会破坏既有 Philippine 场景数据，属破坏性更新，被否决。
- 网关接口先冻结、适配器后落地：乐企/BIR eIS 都需开票主体资质、密钥与协议对接，凭空编码只产生死代码；manual 通道保证业务闭环今天可用，后续接任一网关零迁移。

## 放弃了什么

- 全局硬编码中国税局单属地：与多品牌/跨属地架构冲突（用户裁定否决）。
- 系统发票直接对外（无税局回执）开具：无效票据。
- 本期引入乐企/BIR eIS SDK/签名实现：无资质密钥，先留接口与状态机。

## 关联

- docs/contract/terms.md 第 4 节 invoice.tax_status
- docs/contract/fields.md §3.4（tax_jurisdiction/tax_channel/tax_status/tax_no 列）
- migrations/000049_tax_gateway.up.sql
- adopted/2026-08-18-tax-invoice-arn-numbering.md（Amended：ARN 语义降格为内部流水号）
