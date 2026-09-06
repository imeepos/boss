# AAA P0/P1 执行计划（A3~A6）

> 2026-09-06 晚 ｜ 负责人轮 ｜ 输入:docs/research/aaa-maturity-gap-analysis.md 复盘清单
> 社区基准:FreeRADIUS rlm_sql radacct 惯例（Interim-Update 为累计口径,服务端直接覆盖 UPDATE,从做累加）;per-NAS 客户端管理惯例（clients.conf/nas 表按来源 IP 独立密钥,未注册拒绝）;华为/中兴各有限速 VSA 集（属性名随厂商配置化）
> 纪律:每项独立 worktree;完成即测试提交;合并序由负责人统一调度;合并成功后清理分支。

## A4 Interim 语义修正（最先合并,阻塞 A3 部署）

- 依据 RFC 2866:Interim-Update 的 Acct-Input/Output-Octets 是会话累计值。在线会话维护从"增量累加"改为"覆盖最新累计值";新值小于已存值不回写并输出 [aaa] 告警（计数器回退异常）;cdrs 话单逐包记录保持不变。
- gRPC EmitCDR 旁路口径与 RADIUS 主链路对齐,假设与验证方式在实现说明留痕。
- 验收:make check 全绿;用例覆盖累计覆盖/回退护栏/在线流量值断言;机理写入提交信息。

## A3 部署与数据收口（A4 合并后执行）

- 顺序硬约束:①102 实库应用迁移 000194/000195（先校验后执行,SSH 直查证据）→ ②部署 main 构建并携带 BOSS_AAA_ALLOW_NO_CRED=true 灰度 → ③存量 LOID 账号批量重置凭据（随机口令走既有密文体系,输出数量报告）→ ④关闭开关重启 → ⑤负例验证未设密被拒。
- 全程留痕,失败必告警;部署说明落 docs/ops;验收含抽样 3 账号 PAP 认证 Accept 证据与 admin /aaa/sessions 200 证据。

## A5 per-NAS 注册表 + VSA 限速（G7+G8）

- NAS 客户端实体（IP/密钥密文/厂商/启停/CoA 端口）+ admin CRUD + 迁移;RADIUS 按来源 IP 校验 per-NAS 密钥,未注册拒绝并留痕;CoA 用该 NAS 自己的密钥与端口;全局密钥降级为默认关闭的兼容开关。
- 带宽模板→厂商限速 VSA 映射（华为/中兴,属性名可配）,Access-Accept 按账号所属厂商下发,无匹配走现状兜底;不引入新依赖。
- 验收:未注册拒绝+留痕/per-NAS 密钥放行/CoA 错密钥失败留痕/双厂商 VSA 下发断言/兜底路径;契约同步独立小提交。

## A6 admin 前端补齐

- aaalog 页失败原因列（枚举文案,空=成功）;账号管理加重置密码（二次确认,一次性随机密码弹层可复制）;新增在线会话页（过滤+分页+强制下线,异步受理提示）;menu.def.ts/App.tsx/i18n types 中央登记独立小提交;禁原生 select,双主题验证。
- 验收:typecheck+test+build 全绿;cdp-capture 双主题截图断言四个交互点;对接 102 真实接口。

## 合并调度

A4 → A3（部署依赖 A4 的语义修正）;A5、A6 与 A4 并行,完成即合并;全部合并后负责人收口:ledger 提交、分支清理、会话归档。
