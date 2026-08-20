# 认证配置 secret 存储:复用 biz_params + AES-256-GCM 密文,不建独立表

日期:2026-08-21

## 决策

1. **存储复用 `biz_params`**(key 前缀 `auth.*`,如 `auth.cn.appSecret`),不建独立 auth_config 表:
   配置量级 <20 行、无关联查询、热更链路(ListParams/UpdateParam+审计)现成,独立表是过度设计。
2. **secret 可逆加密 AES-256-GCM**(`internal/pkg/secretbox`),密文格式 `enc:v1:<base64(nonce|ct)>`;
   密钥 env `BOSS_AUTH_SECRET_KEY` 优先,回退 `BOSS_JWT_SECRET`(sha256 派生 32 字节)。
   选可逆而非 hash:认证消费端(App 一键登录回调校验)需还原凭据调供应商 API,hash 不可行。
3. **掩码协议**:GET /auth-config 对 secret 只回 `{value:"",hasValue}`;PUT 空串=不修改。
   任何接口/页面状态不回显明文(审计 detail 只记 secretUpdated+length)。
4. **非 enc:v1: 前缀的存量值按明文兼容读取**,便于未来导入或应急手工修库。
5. **自检接口 v1 = 配置完整性校验**(POST /auth-config/{group}/test),不做真实外呼:
   供应商凭据未到(见 2026-08-20-sms-payment-channel §4 不提前 vendor),外呼自检待凭据到位后
   以薄 HTTP adapter 补上,届时 Amended 本 note。

## 放弃了什么

- 独立 auth_config 表 + 列级加密:多一张表和迁移,换来零查询收益。
- 用 hash 存 secret:无法还原,消费端用不了。
- 真实连通性外呼(JVerification/OpenGateway):凭据未到,先不写假外呼。
