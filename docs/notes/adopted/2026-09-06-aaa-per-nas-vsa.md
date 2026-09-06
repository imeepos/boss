# 决策:AAA per-NAS 注册表与厂商 VSA 限速(A5)

> 2026-09-06 ｜ AAA-A5 会话 ｜ 关联:G7+G8(docs/research/aaa-maturity-gap-analysis.md)

## 裁定

1. **per-NAS 密钥体系**:按请求来源 IP 注册 NAS 客户端(名称/IP 唯一/密钥密文/厂商/CoA 端口/启停,迁移 000196);
   RADIUS 服务端以 SecretSource 按来源 IP 取密钥校验,未注册/停用一律拒绝并留 [aaa] 告警;CoA/强制下线改用
   目标 NAS 自己的密钥与端口。
2. **密钥落库沿用 A1 凭据密文体系**(v1$gcm$,AES-256-GCM 密钥外置):RADIUS 报文签名与 CoA 组包在运行期需要
   密钥明文,单向哈希不可行;复用 codec 保证一套密钥材料(BOSS_AAA_CRED_KEY)管全部 AAA 密文。明文落库被否决。
3. **全局密钥降级为兼容开关**(BOSS_AAA_GLOBAL_SECRET_COMPAT,默认关):仅对未注册 NAS 回退全局密钥/端口;
   停用(enabled=false)NAS 不回退。迁移路径固化为三步(登记→灰度回退→关闭),权威描述 fields.md §8J.3。
4. **VSA 用 RFC 2865 通用编码**,不引入厂商字典依赖;华为 VID 2011 默认 78/80(平均速率,官方规范确定);
   中兴 VID 3902 默认 84/86(镜像华为布局偏移,**未经实机核对**),属性对经 BOSS_AAA_VSA_ZTE 可配覆盖,
   上线前必须对照目标设备 RADIUS 私有属性规范核对。厂商不匹配/无映射回退现状 FramedPool 带宽串,行为不回退。

## 放弃了什么

- 照抄 FreeRADIUS nas 表全量建模(NAS-IP-Address 网段/虚拟服务器/type/ports):当前自研单服务端场景过度设计,只取来源 IP+密钥+厂商+端口+启停。
- 单向哈希存密钥:RADIUS/CoA 均需明文,不可行(与 CHAP 同理,见 2026-09-06-aaa-credential-storage.md)。
- Access-Request 层错密钥主动识别:RFC 2865 下 Request Authenticator 为随机数,无 Message-Authenticator 时协议上不可校验;
  错密钥经计费验证子(RFC 2866 可校验)与 PAP 解密失败间接暴露,库层对可校验报文统一丢弃并留痕。
- VSA 编码引入厂商字典库:违反不新增依赖约束,通用编码(VID+VType+VLen+integer)足够。

## 遗留风险

- 中兴属性码 84/86 未实机核对(可配覆盖兜底)。
- Access-Accept 路径按来源 IP 查注册表增加一次 DB 查询(认证低频,可接受;必要时加缓存)。