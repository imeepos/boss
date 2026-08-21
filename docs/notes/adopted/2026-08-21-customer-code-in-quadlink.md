# customers 表新增 customer_code：四码 customerCode 字段落码（contract）

日期：2026-08-21

## 决策

在 `customers` 表新增 `customer_code VARCHAR(32) NOT NULL UNIQUE`,前缀 `C-`,后 8 位 = `id` 左零 8 位(如 `C-00000046`)。
新增 migration `000085_customer_code.up.sql` / `.down.sql`。

权威映射(进 `fields.md`):

| 概念 | 页面列 | API 字段 | 实体字段 | DB 列 |
|:-----|:-------|:---------|:---------|:------|
| 客户业务编码 | 用户码 | `customerCode` | `CustomerCode` | `customers.customer_code` |
| 客户权威标识(对账/扫码/外键) | 客户 ID | `customerId` | `CustomerID` | `customers.id` |

## why

1. **设计稿/前端三处已固化为"用户码"展示**:
   - `designs/worker-order-detail-v1.spec.md` §4.5 Card4 "四码校验矩阵"四格之一 `customerCode`
   - `docs/worker/order.html:136` "用户码"
   - `docs/worker/report.html:72` "用户码"
   - `docs/admin/quadlink.html:105` "用户码"

   师傅现场装维需要在不依赖手机/系统账号的前提下口头/书面核验客户,字符串码远比 BIGINT ID 实用。
2. **与现有四码字段同构**:`asset_code`(`A-`)、`port_code`(`P-`)、`user_addresses.addr_code`(用户端 `PH-MNL`)均已存在,
   `customer_code` 落码消除四码中"唯独客户没有字符串码"的不对称。
3. **历史错误的兜底修正**:`docs/contract/data-layers.md` 附录 A 第 8 条记录 mock 时期 `quad.customerCode` 错填 `lo_accounts.loid`,
   该口径已被 `fields.md §5.1` 裁定(`customer_id` ≠ `loid`)废止。本 note 给出该字段在新时代的正确语义,
   关闭历史 mock 残留。
4. **`customer_id` 仍是权威**:本决策仅为 customers 增加展示冗余,**不改四码合一逻辑**(`fields.md §5.1` 不变)。
   扫码/对账/外键全部以 `customer_id` 为准,`customer_code` 仅做展示与人工口头核验。
5. **发号复用现有模式**:`adopted/2026-08-18-tax-invoice-arn-numbering.md` 已为 `INV-` / `OR-` 建立 `arn_sequences`
   行锁计数表模式。本字段采用 `id` 派生(零号段消耗、单事务内 `INSERT..RETURNING id` 即得),无需新建发号器。

## 放弃了什么(被否决项)

- **B 案(退化展示为 id_no/phone 掩码)**:契约上把"用户码"语义混进另一个对象(`id_no`/`phone`),破坏 `fields.md §2.1`
  命名边界;且证件号已脱敏(`docs/contract/fields.md §2.1`),截尾展示体验差。
- **C 案(删除 customerCode,前端改 customerName)**:回退产品已固化的"四码校验矩阵"语义,
  `worker-order-detail-v1.spec.md §4.5` 需同步重设计,UI 形态从"码"退化到"姓名+ID"不便师傅现场核对。
- **D 案(契约上承认空串)**:契约没说清楚的话,任何后续 Agent 都会再问一次(本次会话即触发);
  且与设计稿 §4.5 长期不一致。
- **不沿用 `lo_accounts.loid`**:已被 `fields.md §5.1` 裁定废止(`customer_id` ≠ `loid`),即便历史 mock 这么干过。

## 落地任务清单(配套随主变更同提交)

1. ✅ migration `000085_customer_code.up.sql` / `.down.sql`(2026-08-21 落地):
   - 加列 `customer_code VARCHAR(32)`
   - UPDATE 回填 `C-` + 8 位左零
   - 建触发器 `trg_customers_set_code`(BEFORE INSERT,基于 `NEW.id` 派生)兜底新增行
   - SET NOT NULL + UNIQUE INDEX
2. ⚠️ `server-ts` `Customer` 实体:本仓库未发现 server-ts 实体目录(`grep -r 'class Customer\\b' --include='*.ts'` 无命中),
   fields.md §7.1 标注的"server-ts/src/entities/customer.ts"暂未落地。跳过。
3. ✅ Go 域 `Customer` 结构体加 `CustomerCode` 字段;`customerCols` 加 `customer_code`;
   `scanCustomer` 多扫一列。INSERT 不动 — 由 DB 触发器兜底生成。
4. ✅ `internal/httpapi/worker/scan.go:portalQuadH` 把 `customerCode: ""` 改为读
   `quadCustomerCode(a, c, q.CustomerID)`,新函数读 `a.Customer.Get(...).CustomerCode`。
5. ✅ `docs/contract/fields.md` §2.1 增 `CustomerCode` 行 + §5.1 加"展示冗余"说明。
6. ✅ `docs/contract/data-layers.md` 附录 A 第 8 条标 ✅ 已收敛(指本 note)。
7. ✅ 师傅端 `worker/order.html:136` / `worker/report.html:72` / `admin/quadlink.html:105`
   字段不变,由后端接口正常返回即可。
8. ✅ 单测:`pg_test.go` `TestPGStore_Get` / `TestPGStore_List` 更新 mock SELECT 列点,
   并新增 `CustomerCode` 字段断言;`TestPGStore_Get/未命中` 同步更新正则。
9. ⏳ e2e:补一条 `quad{ customerCode: "C-..." }` 期望值 — 待真实 102 环境 migration 跑完后补。

## 落地偏离说明

- 落地方案由"应用层发号"调整为"DB 触发器兜底":保留 note 落地点 #3 INSERT 列同步,但实际
  INSERT 不带 customer_code(由 BEFORE INSERT 触发器基于 `NEW.id` 派生)。优势:零应用层发号
  逻辑、INSERT 路径零改动(两处生产 INSERT + 一处 gis 集成测试全部无需触动);唯一代价:
  PG 端多一个 plpgsql 触发器函数 + 一条触发器,与 `arn_sequences` 行锁计数表同源
  (adopted 2026-08-18)。
- PG BEFORE INSERT 触发器读取 `NEW.id` 的可靠性:PG 10+ BIGSERIAL = `DEFAULT nextval(...)`,
  默认表达式在 BEFORE 触发器之前求值,`NEW.id` 已被 sequence 赋值(参见
  https://www.postgresql.org/docs/current/trigger-definition.html)。

## 关联

- `docs/contract/terms.md` §5 关键术语(冻结语义:`customer_code` 为展示冗余,不改对账)
- `docs/contract/fields.md` §2.1 customers / §5.1 quad_link
- `docs/contract/data-layers.md` 附录 A 第 8 条(本 note 关闭)
- `docs/contract/alignment-audit.md` §7 四码口径错误
- `docs/notes/adopted/2026-08-18-tax-invoice-arn-numbering.md`(发号模式参照)
- `designs/worker-order-detail-v1.spec.md` §4.5(四码校验矩阵 UI 不变)