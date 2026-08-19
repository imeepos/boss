# 师傅端门户 workerJWT:同密钥、独立 claims、issuer 隔离(2026-08-19)

## 决策
师傅端门户 /api/worker/v1 不复用 auth.Manager.Sign 签发账号 claims(aid/usr/role)token,
而是在 internal/app/http_worker_portal.go 内实现 workerJWT:

- 密钥与 TTL 复用同一配置源 config.JWT(BOSS_JWT_SECRET / BOSS_JWT_TTL),不引入第二套密钥。
- claims 独立结构 {wid, wname, sub=worker},issuer 固定 "boss-worker"。
- admin Manager.Verify 只接受 issuer=boss,workerJWT 校验只接受 issuer=boss-worker,
  双向互不可冒充:师傅 token 打不进 /api/v1 admin 面,admin token 打不进师傅端。

## 为什么
auth.Manager.Claims 无 subject 维度;若直接 Sign(workerID, staffNo, "technician"),
admin Authn 会把该 token 当作 accounts 账号接受(aid=workerID),形成横向越权。
middleware 的 worker Subject 机制(API key 路径)覆盖的是免登录 key,不是 JWT 登录。

## 放弃了什么
- 放弃"一个 Manager 签所有主体":短期多写 ~60 行签发/校验,换取主体隔离;
  未来若统一 token 服务,应先给 auth.Claims 加 subject 再合并(届时本文 Amended)。
- 放弃 password 模式即时可用:worker 域未暴露密码校验服务(workers.password_hash
  不经 app 层读),登录暂只放行 sms 模式(验证码存储表也缺,见门户 DB 商议清单)。
