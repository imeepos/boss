# 生产域名 boss.ymm.cn + ingress TLS(cert-manager/Let's Encrypt)终结

日期:2026-08-20

## 决策

1. **公开域名定为 `boss.ymm.cn`,三门户同域、路径前缀区分**:`/api/admin`、`/api/user`、`/api/worker`,
   与 2026-08-19 三端 API 前缀分离决策一致;Web/桌面/移动端统一基址
   `https://boss.ymm.cn/api/<portal>/v1`。
2. **TLS 在 k8s ingress 层终结**(nginx ingress + cert-manager `letsencrypt-prod` 自动签发续期),
   server 内部仍是明文 HTTP;证书不进应用容器,轮换零重启。
3. helm chart 新增 `templates/ingress.yaml` + `values.yaml ingress.*`(默认 enabled=false,
   生产置 true);`deployments/app.env.example` 增 `BOSS_PUBLIC_BASE_URL` 生产段注释。

## why

- 单域名单证书:一张 Let's Encrypt 证书覆盖全部门户,免跨子域 cookie/CORS/PIN 维护;
  移动端只内置一个生产基址(BuildConfig),发版后不变。
- ingress 终结 TLS 是 k8s 事实标准,证书自动续期;应用侧零证书逻辑,回滚/扩容无证书负担。

## 放弃了什么

- 放弃按门户拆子域(user.boss.ymm.cn 等):换来免多证书与跨域配置,代价是网关层无法按子域独立路由/限流
  (当前单副本单体无此需求;若未来按门户拆服务,届时本文 Amended)。
- 放弃 server 进程内 TLS(自管证书):运维成本高且滚动重启才能换证书。
- 放弃通配符证书 *.ymm.cn:ACME DNS-01 需 DNS 服务商凭据,单域 HTTP-01 更简。

## 关联

- `deployments/helm/boss/templates/ingress.yaml`、`values.yaml`
- `deployments/app.env.example`
- 2026-08-19-api-three-portal-prefix.md(路径前缀裁定)
