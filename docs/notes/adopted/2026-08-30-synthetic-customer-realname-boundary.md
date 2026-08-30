# 合成客户实名核验边界裁定（审核 PASS 放行 + 证件照上传放行）

日期:2026-08-30
关联:fix 70672238(合成客户实名链路两处断点)、adopted/2026-09-03-synthetic-customer-recharge-boundary.md(充值边界)、docs/contract/fields.md §7.6/§7.7

## 1. 现象与根因

合成客户(App 自注册,`portal.syntheticID` 负数段隔离空间 ID,无 `customers` 主档)
走实名旅程时两处断点(102 实测复现,注册 cid=-9):

1. **证件照上传 50000**:`attachment.Service.Upload` 的 `UploaderID <= 0` 一刀切
   把负数 ID 拒为 `ErrInvalidUploader`。App 实名第一步即卡死,报「内部错误」;
   102 日志 `httpx: unmapped error (code=50000): attachment: invalid uploader type`。
2. **审核中心 PASS 报 40400 资源不存在**:`guardRealNameIdentity` 以
   `FROM customers c WHERE c.id=$1` 查一致性基准,合成客户无主档行 →
   `pgx.ErrNoRows` → `ErrCustomerNotFound` → 40400。PENDING 单在
   `/base/realname-review` 永久无法通过,FAIL 路径(不走门禁)反而不受影响。

## 2. 裁定

**(A) 审核放行(核验结论只落 verifications)** —— **采用**

- `guardRealNameIdentity` 查无主档(ErrNoRows)→ 放行:合成客户无主档,
  一致性门禁与 id_no 回填均无施力点;
- PASS 后 `UPDATE customers SET real_name_status='VERIFIED'` 本就是 0 行
  no-op,无主档可写;
- 用户端结论回显走 `profile_handlers.portalVerifyStatusPayload` 既有合成客户
  回退(以最新核验单为准),App 端展示已认证,旅程闭环;
- 与充值边界(42201 拒绝)不冲突:充值要写真实账务流水(FK 硬挂 customers,
  隔离空间无真实业务场景);实名核验只写 verifications 表,无 FK 阻碍,且
  实名提交本身是 App 演示旅程的一等能力(6 条关键路径之一),提交允许、
  审核拒绝会造出永久卡死的半开状态。

**(B) 合成客户转正(PASS 时建 customers 主档)** —— 否决

- 动隔离空间设计根基(负数段与正数段物理隔离),影响面远超 bug 修复;
- 注册侧建主档时机是独立产品决策,不该由审核动作隐式触发。

**(C) 审核侧拒绝合成客户(42201)** —— 否决

- 同 (B) 的反面:把半开状态合法化,App 端「审核中」永远等不到结论。

**(D) 附件上传仅拒 UploaderID==0** —— **采用**

- 负数合成客户是合法上传者(`requireCustomer` 注释明确"合成客户 ID 为负数
  (隔离空间),也是合法客户");`attachments.uploader_id` 无 FK,负数登记
  无副作用;账号/worker 恒为正数段,行为不变。

## 3. 实现落点

- `internal/domain/attachment/minio.go::Service.Upload`(校验从 `<=0` 收窄为 `==0`);
- `internal/domain/customer/pg_onboarding.go::guardRealNameIdentity`(ErrNoRows → nil 放行);
- 回归:`TestUploadSyntheticCustomerID`、`TestPGStore_Verify_SyntheticCustomerPasses`
  (替代原 MissingCustomerReturnsNotFound 断言,语义反转)。

## 4. 验证口径(修复部署 102 后)

注册新合成客户 → 上传两张证件照(200)→ `POST /auth/verify` 落 PENDING →
admin `POST /verifications/customer/-N/verify {"result":"PASS"}` 返回 code=0
(修复前 40400)→ `GET /auth/verify` 状态经核验单回退显示已认证。
