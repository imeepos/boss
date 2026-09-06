# AAA LOID 凭据落库采用 AES-256-GCM 密钥外置(非单工哈希)

> 日期:2026-09-06 ｜ 域:AAA(A1 认证凭证与防爆破,G1/G2) ｜ 状态:已采纳

## 决策

`lo_accounts.password_credential` 落库 `v1$gcm$<nonce-b64>$<ct-b64>`(AES-256-GCM,
每次随机 nonce,同口令不同密文);密钥外置(env `BOSS_AAA_CRED_KEY` / 文件挂载,
空则从 `BOSS_AAA_SECRET` 派生并打 `[aaa] CRED KEY ALERT`,生产必须显式配置)。
编解码与校验收敛在 `internal/domain/aaa/credential`。

## why

A1 需求同时要求:凭据不落明文(需求 1)+ CHAP 按 RFC 1994 校验(需求 2)。
CHAP 响应 = MD5(ident || password || challenge),challenge 每请求随机,服务端必须
能还原口令才能算出同一摘要——单工哈希(bcrypt/SHA-256)数学上不可实现 CHAP;
FreeRADIUS 生态同样要求 CHAP 场景提供 Cleartext-Password。故"哈希存储"与
"CHAP 校验"不可兼得,取"不落明文"的安全目标:库内只有密文,密钥在 DB 之外,
拖库不直接泄漏口令(密钥与库分离才可解)。

## 放弃了什么(被否决项)

- 单工哈希(bcrypt/argon2/salted SHA-256):PAP 可验,CHAP 永远失败,违背需求 2;
- 明文落库:CHAP 可验但直接违背需求 1,否决;
- 双列(hash + 可逆):可逆列存在使哈希列失去安全意义,纯增攻击面;
- MS-CHAPv2(NT-hash 单工可验):需求明确指定 RFC 1994 CHAP,且 NAS 客户端兼容面不同。

## 风险与缓冲

- 口令可还原意味着内部可解密:密钥权限收敛到 aaa 进程;派生键仅为开发兜底,
  上线清单必须显式替换;
- 迁移缓冲:存量未设密账号默认 Reject;`BOSS_AAA_ALLOW_NO_CRED=true` 可临时放行,
  默认关;
- 防爆破:同 LOID 连续失败默认 5 次锁 15 分钟(`BOSS_AAA_LOCK_THRESHOLD`/
  `BOSS_AAA_LOCK_WINDOW` 可配),锁定表 lo_auth_lockouts(000194)。

## 关联

- 迁移 000194_aaa_credential_lockout;PAP/CHAP 校验与锁定:cred_authorizer.go/lockout_pg.go;
- docs/research/aaa-maturity-gap-analysis.md(G1/G2);terms.md auth_log.fail_reason 枚举。