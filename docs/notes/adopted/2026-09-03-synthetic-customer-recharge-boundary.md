# 隔离空间合成客户(负数 ID)充值边界裁定

日期:2026-09-03

## 1. 现状边界

合成客户(`portal.NextSyntheticCustomerID` 返回负数 customer_id,
隔离空间,无 `customers` 主档)的充值路径:

- `POST /api/user/v1/topups`(用户端门户)→ `RecordTopup`:
  - 入参 `Payment.CustomerID` = 负数时跳过 FK 检查(`pg_topup.go:31 if p.CustomerID > 0`),
  - 但 `INSERT INTO payments(...) VALUES(..., $2, ...)` 的 `customer_id` 列是
    `000068 起的 BIGINT REFERENCES customers(id)` 硬 FK(不可空时由迁移 000148
    确认全员双挂),负数 ID 在 `customers` 表无对应行 → 23503 FK 违反 → 充值失败;
  - 同事务的 `INSERT INTO portal_wallets(...)` 走 `ON CONFLICT (customer_id) DO UPDATE`,
    主键冲突由 PG 自动 upsert,余额写入不受 FK 拖累——但流水无法落库,余额也就
    永远不会进入。

## 2. 裁定(三个选项及理由)

**(A) 拒绝合成客户充值** —— **采用**

- `topupRecordPayment` 入口先校验 `cid > 0`,负数直接回 42201(参数非法:合成客户不支持充值);
- UI 在用户端 /topups 页面按 `cid > 0` 隐藏充值档位(已在 docs/user 改造范围内);
- admin 端 `POST /admin/v1/topups` 同样门禁——合成客户不在收费场景中;
- 余额查询 `GET /api/user/v1/topups` 仍可读(合成客户的 portal_wallets 由注册时 portal 域
  零值初始化),只是不能写入。

**(B) 允许合成客户充值(改 FK)** —— 否决

- 改 `payments.customer_id` FK 为可空或拆出硬 FK——会破 `000068/000148 双挂语义`,
  真实客户账单流水的 customer_id 也会被松绑,与既有裁定冲突;
- 合成客户充值不属于真实业务(隔离空间=演示/测试),放开既无场景也无审计需要。

**(C) 演示模式专属旁路** —— 否决

- 给合成客户加一个 `kind=SYNTHETIC` 的伪充值接口(只写 portal_wallets 不写 payments),
  后续维护成本高于价值,无真实演示诉求;
- 已通过的 docs/user 收口证明真实 user API 已能跑通演示旅程,无需旁路。

## 3. 实现落点

文件:`internal/httpapi/user/billing_handlers.go::topupRecordPayment`
- 在调 `RecordTopup` 之前加 `if cid <= 0 { respond c 42201 错误; return }`;
- 同步在 `portalBalanceGet` 读取侧也加注释:合成客户的 portal_wallets 余额始终 0
  (无充值路径),查询返回合法零值。

测试:`internal/httpapi/user/payment_list_test.go` 追加 `TestPortal_TopupSyntheticReject`
- 合成客户调 `POST /topups` 期望 42201;真实客户调成功。
- 端到端覆盖(`102 实测`):合成客户 jwt 登录 → `POST /topups` → 42201;
  真实客户同样路径 → 200。

## 4. 关联

- `internal/domain/portal/portal.go::Service.NextSyntheticCustomerID`
  + `pg.go::syntheticID`(负数段定义);
- `migrations/000068_payment_nullable_bill.up.sql`(payments.customer_id FK 加挂);
- `internal/domain/billing/pg_topup.go::RecordTopup`(落账逻辑);
- `docs/contract/alignment-audit.md D2`(隔离空间合成 ID 号段裁定);
- `docs/notes/adopted/2026-08-20-db-dualtrack-convergence.md §2`(portal_accounts 隔离空间)。

## 5. 放弃了什么

- 不改 FK(不动 000068):双挂语义是收口硬化成果,留合成客户旁路会重新打开口子;
- 不建 payments 旁路表:合成客户无真实充值业务,徒增表面积;
- 不动 docs/user 余额 UI(已在前批 P1-1 修过):合成客户的余额始终 0,UI 显式展示「—
  未充值」即可,无需特殊提示;
- 不动 admin-web 后台充值入口:admin 端仅服务真实客户,合成客户连 admin 都进不去。