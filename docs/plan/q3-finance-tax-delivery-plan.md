# Q3 执行计划:财务、税务与交付标准化(2027 Q2)

> 来源:docs/plan/three-year-roadmap.md §2027 Q2。目标:消除账务、税务和环境交付中的人工差异,具备可复制部署能力。
> 验收:① 账实核对差异可定位 ② 发票状态与税局状态正交且可追踪 ③ 新环境按清单完成部署和初始化。

## 一、现状盘点(2026-08 审计)

| 能力 | 现状 | 缺口 |
|---|---|---|
| 发票+ARN | INV-/OR- 连续发号,作废保留编号(000048) | 无 |
| 税局正交状态 | invoice.status 与 tax_status 正交(000049) | 无 |
| 税局网关 | CN/PH 属地注册表,tax-submit/FAILED 可重试/tax-backfill 回执回填 | 缺税局事件轨迹(submit/receipt/void 只有终态) |
| 渠道对账 | 批次+行级明细,四种 diffKind(000035) | 无 |
| 账实核对 | **缺失** —— 本轮已补(见二) | — |
| 预付/后付 | billing_mode 快照挂 lo_accounts/orders(000103) | 退款、账单日切边界测试集未成体系 |
| 欠费停复机 | 任务留痕+LO 账号原子迁移+失败重试 | 无 |
| 数据隔离 | legal_entity_id 快照+DataScope(公司/区域)SQL 裁剪(AAA 已收口) | 交付模板/初始化种子/校验清单未文档化 |
| Admin 设计系统 | shadcn 风格组件 + PageHead/Toolbar/Table 基座 | 无统一规范文档,页面仍有内联样式分叉 |
| 环境交付 | docker-compose.102 + Helm chart + CI | 缺"新环境从零初始化"清单 |

## 二、本轮已交付(P1)

**账实核对 Ledger Reconciliation**(验收①落地):

- `GET /billing/ledger-recon?period=YYYY-MM&legalEntityId=`(permCode menu:paycheck)
- 应收(bills) vs 实收(SUCCESS payments−REFUNDED) vs 开票(ISSUED invoices)三角比对
- 逐账单 diffKind:UNPAID / PARTIAL / OVERPAID / REFUNDED / PAID_NO_INVOICE / MATCH
- 汇总:总应收/总实收/总开票 + byKind 计数;差异行优先排序;分页
- DataScope(法人+区域子树)SQL 裁剪;设计:docs/design/q3-ledger-recon.md
- 测试:ClassifyLedgerRow 表驱动 + pgxmock(排序/汇总/分页)

## 三、后续轮次排期

| 优先 | 工作项 | 对应验收 | 说明 |
|---|---|---|---|
| P2 | tax_events 轨迹表 + 提交/回执/作废留痕查询 | ② | 现仅终态+fail_reason,审计不可回放 |
| P3 | Admin paycheck 页新增"账实核对"页签消费 ledger-recon | ① | UI 呈现差异行与汇总卡 |
| P4 | 账务边界测试集:预付/后付/退款/欠费/停复机/日切 | ① | 端到端账务口径回归 |
| P5 | 新环境初始化清单:部署→迁移→种子→权限模板→验收冒烟 | ③ | 复用 devseed + backup 恢复演练 |
| P6 | Admin 设计系统规范文档 + 页面样式收敛 | 支撑 | 列表/表单/详情/流程四类模板 |
| P7 | 属地配置产品化(tax gateway 参数入 biz_params 热更) | ② | 现为装配期注入 |

## 四、边界与不做

- 不新建统计表复制 bills/payments 数据;账实核对实时查询。
- 不改 000011/000048/000049 既有迁移语义;P2 轨迹表走增量迁移(先查两处迁移号防撞号)。
- 不用 mock 验收;102 真实环境复测(API 已上 CI 部署链)。
