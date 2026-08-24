# 2026-08-23 阿里云国际短信 API 实测修正:dysmsapiintl 端点已失效

## 决策

对接真实测试账号(LTAI 开头 AK)实测发现,2026-08-22 选型 note 依据的旧 API 形态已整体失效,按实证修正 `internal/pkg/sms/aliyun_intl.go`:

- **端点**:`dysmsapiintl.aliyuncs.com` 全球 NXDOMAIN(本机/102/223.5.5.5/8.8.8.8 均无 A 记录),真实端点为 `dysmsapi.ap-southeast-1.aliyuncs.com`。
- **参数**:旧 `To/From/Message/MessageTag` 自由文案形态不被接受;实测必填为 `PhoneNumbers`(纯数字,带 `+` 报 MOBILE_NUMBER_ILLEGAL)+ `ContentCode`(控制台报备模板编号)。验证码走 `Type=OTP` + `VerificationCode`。
- **响应**:成功/失败键为 `ResultCode`/`ResultMessage`(非 ResponseCode/ResponseDescription)。
- **配置键**:`sms.from`(SenderID,API 已不接受)与 `sms.template.cn/my`(自由文案,已不支持)移除,改为 `sms.contentCode.cn/my`(报备模板编号);`BOSS_SMS_ALIYUN_FROM` env 同步移除。
- 签名算法(POP V1 HMAC-SHA1)不变,实测通过(错误均为参数级,无 SignatureDoesNotMatch)。

## 放弃了什么

- **自由文案模板**:"文案模板"配置卡片语义改为"报备模板 ContentCode",文案本身由阿里云控制台模板定义,系统侧不再渲染 `{code}` 占位。
- **SenderID 配置**:新 API 不接受 From 参数,签名在控制台模板侧管理。

## 后续偿还

- 测试账号控制台尚无报备模板(任意 ContentCode 报 SMS_CONTENT_CODE_ILLEGAL),真实试发被阻塞;模板报备通过后用 `POST /sms-config/channel/test` 带 phone 复验端到端。
- `Type=OTP/VerificationCode` 参数形态基于旧版国际 API 词汇推断,真实模板报备后如报参数错误需按实测再修。
