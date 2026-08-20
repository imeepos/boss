# 2026-08-24 实名二要素自动核验:阿里云实人认证 Id2MetaVerify,提交即核验,凭据未配保持人工

## 裁定

客户实名(用户端 `POST /auth/verify` 与 admin `POST /customers/:id/real-name`)提交后,
若配置了阿里云实人认证凭据(`BOSS_REALID_ALIYUN_AK_ID/SECRET`),即时调
`cloudauth.aliyuncs.com Id2MetaVerify`(身份二要素:姓名+身份证号, POP V1 签名零 SDK 直调):
BizCode 1→PASS、2/3→FAIL,结论直接落 verifications(PASS 同步 `customers.real_name_status=VERIFIED`),
operator_name 快照「阿里云二要素」(method 归「第三方」枚举,operator_account_id=0)。
凭据为空(通道 nil)或通道调用失败/落库失败 → 保持 PENDING 走既有后台人工核验,不阻塞提交。

## 为什么

- 实名是开户前置(terms.md real_name_status),纯人工核验拖长 onboarding 闭环;
  二要素是监管底线核验,阿里云权威数据源,无需用户拍照即可判定。
- 与短信(2026-08-22)/支付(2026-08-23)同构:凭据未到不接渠道、不 vendor SDK,
  env 兜底、nil=禁用降级,行为可在无凭据环境完整回归。

## 放弃了什么

- **云市场第三方二要素 API(AppCode 头)**:供应商锁定+账号多一套,阿里云主账号 AK 已有
  (与短信同源),cloudauth 官方产品长期更稳。
- **人脸/证件 OCR 多要素(Cloudauth Id2MetaVerifyWithOCR /实人认证 H5)**:需要端上拍照链路,
  移动端 UI 与活体流程成本高;二要素先行,人像比对留后续按需接(接口签名同构)。
- **提交改同步长阻塞**:通道 10s 超时,失败即 PENDING 人工兜底,不引入异步任务/重试队列。
- **worker 师傅实名同轮接入**:复用同一通道很容易,但按"一次改一件事"纪律单独成笔。

## 落点

- `internal/pkg/realid/`:Verifier 接口 + aliyun cloudauth 实现(签名与 sms/aliyun_intl 同构,不跨包抽公共层)。
- `internal/app/realid.go` AutoVerifyRealName:编排核验→落库→同步主档,PENDING 兜底。
- 接线:用户端 portalVerifySubmit、admin customer real-name 提交端点。
