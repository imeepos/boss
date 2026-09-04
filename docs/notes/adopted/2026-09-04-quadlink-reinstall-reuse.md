# 四码同客户同地址重装:刷新复用既有 quad_link 行

- 日期:2026-09-04
- 状态:Adopted
- 关联:任务A(装维链路缺陷修复)取证一;terms.md §5「四码合一」Amended;000110(活跃链路唯一索引)

## 事实

102 装机链路三角色模拟:师傅端 POST /tickets/{no}/scan-bind 对 EPC-0002 稳定 50000,
日志 `quadlink: scan link: duplicate key uq_quad_links_address (23505)`。
DB 实况:地址 288 存在历史 quad_links 行 id=379(LINKED,同客户 213,来自旧装机
ORD-20260819-000311 DONE)。同客户同地址重装时:applyTag 营新 UNLINKED 预绑定行
(不触索引),scan-bind 置 LINKED 时与历史 LINKED 行撞部分唯一索引。

## 裁定

VerifyScan(环节9)置 LINKED 前查同地址活跃行(LINKED/CONFLICT):

1. **同客户 → 刷新复用既有行**:UPDATE 既有行 port_id/asset_id/status='LINKED'
   (旧端口由新单端口顶替,资产以现场扫码为准),删除本单临时预绑定行(同一订单
   数秒前的中间产物,非审计对象;审计由 scan_logs + `[quadlink] LINK REUSE`
   grep 级日志承载),四码身份延续,一地址至多一条活跃链路不变。
2. **跨客户 → 拒绝**:`quadlink.ErrAddressConflict` → 40920 族(reason 透传),
   禁止把 A 客户的地址链路改绑给 B 客户。

## 为什么

- 重装(同客户同地址)是正常业务流,终身一行的唯一索引与它天然冲突;000110 已把
  唯一性收窄到"活跃链路",但 DONE 装机的 LINKED 行本就该保持 LINKED(客户在网),
  唯一可行路径是复用而非新增。
- 复用保住四码身份连续性:用户地址码/客户码不变,端口/资产随新单刷新,
  GetByAddress(取最新行)不会读到过期 UNLINKED。

## 放弃了什么

- **旧行置 UNLINKED + 新行置 LINKED**:地址生命周期内留下两行,GetByAddress
  按 id 倒序取最新,旧行占坑且语义混乱;对账 Conflict 判定面扩大。
- **应用层先 DELETE 后 INSERT**:四码行 id 换代,历史引用(如按 link id 的排查)断链。
- **DB 触发器 upsert**:逻辑藏进触发器,应用层不可测、审计不可 grep,违反平台
  「失败/关键路径必须可观测」红线。
- **迁移回填历史 DONE 行**:不动存量数据;Reconcile 的孤儿清理 + 本裁定已覆盖
  增量场景,历史多行由对账任务按既有规则收敛。

## 影响面

- quad_links 每地址活跃行 ≤1 的既有不变式保持;scan-bind 请求/响应契约不变。
- 40920 族新增语义:地址活跃链路跨客户冲突(此前仅扫码不一致)。
