# 2026-08-22 短信验证码通道选型:阿里云国际短信单一通道起步,区号路由留扩展位

## 决策

验证码登录短信,中国大陆(+86)与马来西亚(+60)现阶段统一走**阿里云国际短信**(dysmsapiintl,`SendSMS` RPC, POP V1 HMAC-SHA1 签名,零 SDK 依赖直调)。区号路由抽象 `internal/pkg/sms.Router` 预留 `ByRegion["86"]` 注入位,中国国内报备通道(阿里云/腾讯云国内短信,需企业签名+模板报备)就绪后无缝切换。

配套裁定:

- 号码统一归一化为 E.164 存储/外发;裸 11 位 1 开头号码默认中国,显式 `+` 前缀必须命中支持区号(86/60),否则 422。
- 同 phone+scene 60 秒冷却(`portal_sms_codes.issued_at`),防刷;冷却命中返回 42300 CodeResourceBusy。
- 凭据(`BOSS_SMS_ALIYUN_AK_ID/AK_SECRET/FROM`)为空时降级 `LogSender`(日志打印验证码,仅开发联调)——本地/CI 无需真实凭据。
- 验证码文案按区号双语:86 中文 / 60 英文,内置默认可后续配置化。

## 放弃了什么

- **中国国内通道立即接入**:签名/模板报备周期长且需企业资质落定,阻塞登录主线;阿里云国际版可直达中国号码(到达率略低于国内三网,接受)。
- **Twilio/Infobip**:多一套供应商账号与结算,且与阿里云国内版未来收敛到同一控制台的机会;当前量级不值得。
- **memoryStore 发送钩子**:发送属 PG 真实链路,内存替身保持"固定码 123456"不动,测试不外呼。

## 后续偿还

- 企业资质落定后,接入国内通道并注入 Router `ByRegion["86"]`;同时评估国内到达率。
- 验证码文案模板从硬编码迁到配置/DB。

> Amended: 端点/参数形态已被实测推翻，见 `2026-08-23-sms-aliyun-intl-endpoint-fix.md`（dysmsapiintl 域名 NXDOMAIN，真实端点 dysmsapi ap-southeast-1，参数 PhoneNumbers+ContentCode）。
