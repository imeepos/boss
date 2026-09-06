# AAA 凭据密钥经宿主机文件注入 102(BOSS_AAA_CRED_KEY_FILE + bind mount)

> 日期:2026-09-06 ｜ 域:AAA(A3 部署与数据收口) ｜ 状态:已采纳

## 决策

102 生产凭据密钥不进仓库/compose/进程 argv:宿主机生成 `openssl rand -hex 32` 一次写入
`/etc/boss-aaa/cred.key`(0600,属主 1000:1000=镜像 app 用户),compose 双侧(server/aaa)
bind mount 只读至 `/run/aaa-cred/cred.key`,经 `BOSS_AAA_CRED_KEY_FILE` 读取;
值不写任何仓库文件/文档,部署文档只记指纹前 16 位(e3c9b46d405cf7f6)供核对。
A5 起该密钥材料同时保护 per-NAS 注册表密文,密钥丢失=全部凭据重置,属难逆决策。

## why

任务红线明确「密钥值不得写入任何仓库文件或文档」,而 102 的部署通道是仓库 compose
(CI 从仓库克隆读 compose,内联 environment 会把值带进 git);config 已支持
readEnvOrFile 文件注入,宿主文件 + bind mount 是零代码改动的最小通道,与既有
BOSS_HOSTCTL_HMAC_KEY_FILE(/etc/boss-host-ctl/hmac.key)先例同构。属主 1000:1000
为实测踩坑结论:镜像 USER app(非 root),root 0600 文件容器不可读会静默退回派生密钥。

## 放弃了什么(被否决项)

- compose 内联 `BOSS_AAA_CRED_KEY` 值:直接违背「值不进仓库」,否决;
- Secret 派生兜底(`boss-aaa-cred-key|BOSS_AAA_SECRET`):开发兜底可用,生产不可
  (密钥强度受 Secret 复用拖累且每启动打 ALERT),仅作缺配告警路径保留;
- 独立 secret manager(Vault 等):102 无该基础设施,引入新依赖违背最小改动,否决;
- chmod 604/644 放开读:世界可读密钥,否决;chown 容器 UID 保 0600 达成最小可读。

## 关联

- 落库形态与派生口径:adopted/2026-09-06-aaa-credential-storage.md;
- A5 per-NAS 密文复用:adopted/2026-09-06-aaa-per-nas-vsa.md;
- 部署实录与回滚:docs/ops/aaa-a3-deploy-closeout-2026-09-06.md。