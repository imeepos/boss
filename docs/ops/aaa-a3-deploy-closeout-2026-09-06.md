# AAA-A3 部署与数据收口记录(102)

> 2026-09-06 ｜ 执行:AAA-A3 会话 ｜ 依据:docs/plan/aaa-p0p1-execution-plan.md A3 节
> 上线顺序硬约束:①迁移 → ②灰度开关(true)部署 → ③存量 LOID 批量重置凭据 → ④关开关(false)重启 → ⑤负例验证。全程留痕,明文口令零落盘。

## 1. 环境勘察结论(动手前)

- 102 部署形态:gitea CI(`.gitea/workflows/deploy-102.yml`)push main 自动构建全二进制镜像(GIT_SHA ldflags 注入)→ 私有仓库 `192.168.0.102:5000/boss/server` → `docker-compose -f deployments/docker-compose.102.app.yml -p boss-app up -d --force-recreate`;compose 配置内联 environment,来自仓库文件。
- 应用启动自动跑迁移(`database.Migrate`,internal/app.New);`scripts/ops/migrate-102.sh` 为手工执行/回滚工具。
- PG:`boss-infra-postgres-1`,库 boss,用户 boss(25432 对外)。
- boss-aaa 容器同镜像 entrypoint `boss-aaa`,UDP 1812/1813;服务器为多人共享环境,动手前已核查无他人部署进行中。

## 2. 迁移(000194/000195/000196,按序)

102 实库由当日 CI 部署的应用启动自迁移应用(早于本任务执行窗口),本任务以直查留证并复核对象完整性:

```
schema_migrations: 000194_aaa_credential_lockout applied_at=2026-09-06 13:40:40+00
                    000195_aaa_online_sessions applied_at=2026-09-06 13:50:29+00
                    000196_aaa_nas_clients       applied_at=2026-09-06(A5 随 069dafc 部署自迁移,本任务复核)
lo_accounts.password_credential TEXT NULL          # 存在
auth_logs.fail_reason VARCHAR(32) DEFAULT ''       # 存在
lo_auth_lockouts(loid,fail_count,locked_until,updated_at)   # 存在
aaa_online_sessions 全列 + uq_aaa_online_sessions_loid_session + 2 索引   # 存在
cdrs.close_reason VARCHAR(32) NULL                 # 存在
aaa_nas_clients(id,name,nas_ip,secret_enc,vendor,coa_port,enabled,时间戳)   # 存在,0 行
```

回滚:`scripts/ops/migrate-102.sh down 000193_oss_inventory_audit`(down 前自动 backup-102.sh)。

### 2.1 A5 per-NAS/VSA 配置面(负责人通知后补记,2026-09-06)

- NAS 注册:`aaa_nas_clients` 当前 **0 行**;未注册 NAS 来源的 RADIUS 包被丢弃并留痕(`[aaa] NAS REJECT UNREGISTERED ip=...` + `nas not registered`)。真实 NAS 接入前必须先登记,或由负责人裁定开启兼容开关。
- 兼容开关:`BOSS_AAA_GLOBAL_SECRET_COMPAT=true` 时未注册 NAS 回退全局密钥(compose 未设=关,默认严格)。
- VSA 属性名 env(可后配,未设即用内置默认):`BOSS_AAA_VSA_HUAWEI`(默认 input-average-rate,output-average-rate → 华为 VID 2011 类型码 78/80 平均速率)、`BOSS_AAA_VSA_ZTE`(覆盖 ZTE VID 3902 属性对,默认语义名映射 84/86,以目标设备规范为准,fields.md §8J)。后续需要调整时把对应 env 追加进 `deployments/docker-compose.102.app.yml` 的 aaa 服务即可,随下次部署生效。

## 3. 凭据密钥 BOSS_AAA_CRED_KEY(值不入仓库)

- 生成:102 宿主机 `openssl rand -hex 32`(32 字节熵,64 hex 字符),一次写入。
- 存放:`/etc/boss-aaa/cred.key`,0600,属主 **1000:1000(容器 app 用户)**;bind mount 只读进 server/aaa 两容器 `/run/aaa-cred/cred.key`,经 `BOSS_AAA_CRED_KEY_FILE` 读取(readEnvOrFile 支持文件注入)。值不进仓库/env 明文/process argv/任何文档。
- 指纹(sha256 前 16 位,仅供核对):`e3c9b46d405cf7f6`;server 与 aaa 容器内指纹一致已实测。
- 踩坑记录:镜像以 app(1000:1000)运行,root 属主 0600 文件容器内 Permission denied → 启动打 `CRED KEY ALERT` 退回 Secret 派生密钥;chown 1000:1000 并重启后 ALERT 消失。
- 影响面(2026-09-06 起):除 LOID 凭据外,A5 per-NAS 注册表的 NAS 共享密钥密文复用同一 codec/密钥;密钥丢失 = 全部凭据不可解,需重置所有 LOID 口令并重录 NAS 密钥。建议按密钥托管流程异地备份一份(本任务未代管)。

## 4. 代码修复(部署前置缺口)

- `2b62501e fix(aaa)`:boss-server 的 app.Aaa 装配漏注凭据编解码器,admin 重置接口 `POST /lo-accounts/{loid}/reset-password` 必返 `aaa: credential codec not configured`(102 日志 15:59 已两例)。抽 `credential.NewResolved` 共享装配,server 与 cmd/aaa 同口径;commit 含单测,make check 全绿。
- `4da9b826 fix(contract-sync)`:并行 A6 会话并入的 aaasession 菜单键使 E 项门禁红,按检查器既定豁免通道登记 baseline(复用 menu:loaccount,同 capacity 先例)。

## 5. 部署与证据

- 部署 1(灰度):main=4da9b826,healthz 自报 commit=4da9b82,5 容器 healthy,aaa 无 CRED ALERT,双侧指纹一致。
- R2 证据:`GET /api/admin/v1/aaa/sessions`(X-API-Key)= **HTTP 200** `{"code":0,"data":{"items":[],"total":0,...}}`;UDP 1812/1813 监听(ss 实查)。
- 部署 2(关开关):main=a9e7cc24,healthz=a9e7cc2,`BOSS_AAA_ALLOW_NO_CRED=false` 实查(printenv)。

## 6. 数据收口(R3)

- 批量重置:3/3 ACTIVE LOID 账号走 admin 重置接口(密钥读 102 本地克隆文件,响应明文仅存 shell 变量,即用即弃,零落盘零日志):LOID-C-00000214 / LOID-C-00000216 / LOID-E2E-RESUME-001,接口 code=0。
- 抽样认证:3/3 PAP 探针(RFC 2865 §5.2 User-Password 隐藏编码,临时探针脚本 /tmp/pap_probe.py 用后即删)→ **Access-Accept ×3**;auth_logs 同刻 3 行 SUCCESS(fail_reason 空)。
- 库内对账:`lo_accounts` total=3,v1$gcm$ 密文 3/3(67 字符);`lo_auth_lockouts` 0 行。

## 7. 关开关与负例(R4)

- 终态:`BOSS_AAA_ALLOW_NO_CRED=false`(compose 已固化,重启生效,healthz=a9e7cc2)。
- 负例:新建无凭据账号(SQL 受控造数,克隆既有行仅换 loid,验后即删)RADIUS 认证请求被**拒**——A5 并入后未注册 NAS 来源先行丢弃:`[aaa] NAS REJECT UNREGISTERED ip=172.28.0.1` + `nas not registered`(见 §9 风险 1);凭据层拒绝语义由既有实证背书:同探针 16:30 对不存在 loid 得 Access-Reject 且 auth_logs=FAILED|NOT_FOUND;cred_authorizer 未设密+开关关→BAD_CREDENTIAL 计数锁定逻辑未变。
- 造数清理:负例账号已删,lo_accounts 回到 3 行(造数不过夜)。

## 8. 回滚路径

1. 镜像:私有仓库按 sha tag 回退——`docker tag 192.168.0.102:5000/boss/server:<旧sha> ...:latest && docker compose -f deployments/docker-compose.102.app.yml -p boss-app up -d --force-recreate`(0efeeb6c/7458039f/db8b20b3 等历史 tag 都在)。
2. 开关:compose `BOSS_AAA_ALLOW_NO_CRED` 改回 true 重启(仅迁移缓冲用,重置已完成不应回开)。
3. 库:`migrate-102.sh down 000193_...`(自动先备份);仅清凭据:`UPDATE lo_accounts SET password_credential=NULL`。
4. 密钥轮换:生成新文件→chown 1000:1000→重启→**必须**重置全部 LOID 口令并重录 NAS 密钥(旧密文全部不可解)。

## 9. 遗留风险与移交项

1. **A5 per-NAS 严格门已随并行会话上线 102**(069dafc 起):未注册 NAS 来源的 RADIUS 包一律丢弃(默认全局密钥兼容开关关)。任何真实 NMS/oltsim 演练/探针需先在 A5 注册表登记 NAS,或由负责人裁定开启兼容开关(compose aaa 服务加 `BOSS_AAA_GLOBAL_SECRET_COMPAT: "true"`);oltsim 相关 E2E 脚本可能受影响,属 A5 移交事项。
2. 密钥异地备份未落(见 §3),建议纳入密钥托管流程。
3. LOID-E2E-RESUME-001 为 E2E 遗留账号,已随批量重置获得凭据;其口令无人持有,如需复用请重置。
4. 102 为共享环境,当日并行会话(A4/A5/A6)多次推进 main;本任务三次合并均按 worktree 协议 rebase 后 ff-only 完成。
