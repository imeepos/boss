# 关键写操作幂等审计(Q1) 2026-08-23

> Q1 验收指标:关键写操作幂等覆盖率 100%。
> 审计范围 = 主链路 12 环节的全部写端点(admin 端;user/worker 端同链路复用同域服务)。
> 判定标准:同一请求重放 → ①等同成功响应且无重复副作用(严格幂等),
> 或 ②明确业务码拒绝且无副作用残留/资源泄漏(安全拒绝)。

## 结论表

| 环节 | 端点 | 重放行为 | 机理(代码证据) | 判定 |
|------|------|----------|----------------|------|
| 1 下单 | POST /orders | 无幂等键,重复=重复订单 | pg.go Submit 仅校验主体存在,无去重键 | **缺口,待裁定** |
| 2 核查 | POST /orders/:no/check-resource | 重复→重复返回清单,不二次推进 | pg_check.go:已核查(stage>=2)直接 return nil | 幂等 ✓ |
| 3 预占 | POST /orders/:no/reserve | 重复→状态机拒绝+端口补偿回滚 | handler 补偿调 ReleasePortByOrder,无 RESERVED 泄漏 | 安全拒绝 ✓ |
| 4 收费 | POST /orders/:no/charge | 重复→跳过收费直接重跑自动段 | handler: stage>=4 跳过 ChargeContract | 幂等 ✓ |
| 5-8 自动段 | Automation.run | 重复→已完成环节跳过 | automation.go: cur >= stage continue(幂等续推) | 幂等 ✓ |
| 8 指派 | POST /dispatch/pool/:no/assign | 重复→同师傅覆盖写,状态不变 | pg_dispatch.go: 纯 UPDATE 回填 | 幂等 ✓ |
| 9 扫码 | POST /tickets/:no/scan-bind | 修复前:重复写 scan_logs+推进报错;**本次修复**:LINKED 同资产重扫直接 MATCH,不写重复日志;环节9已完成的推进拒绝转 MATCH 响应 | pg_scan.go 幂等短路 + scan_handlers.go ErrIllegalTransition 转 MATCH | 幂等 ✓(本次) |
| 10-12 激活 | POST /tickets/:no/activate | 重复→自动段幂等续推跳过 | 同 Automation.run | 幂等 ✓ |
| 实名提交 | POST /customers/:id/real-name | 重复→新核验记录(留痕型,非重复副作用) | onboarding: 记录型 append,审核态以最新为准 | 可接受 ✓ |
| 缴费 | billing 收款 | pay_no 唯一幂等 | billing.go: 同事务 pay_no 唯一 | 幂等 ✓ |

## 缺口与裁定建议

**POST /orders 无幂等键**(客户/客服双击 → 重复订单,重复占端口/重复收费风险):
- 方案A:请求带 clientRequestId,orders 表唯一索引去重(契约变更,三端同步);
- 方案B:同 customer+address 存在非终态订单时拒绝(业务语义强,误伤合法二装)。
- 需 dated note 裁定后实施;实施前该端点记为未覆盖。

## 覆盖率

- 主链路写端点 10 项:幂等/安全拒绝 9 项,缺口 1 项(下单),**90%**;
  下单幂等化落地后达 100%。

## 回归测试

- internal/domain/quadlink/pg_scan_test.go:重扫 LINKED 同资产 → MATCH 无第二日志
- internal/httpapi/admin/scan_test.go:环节9已完成重扫 → MATCH 不报错
