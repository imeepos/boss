# AAA 域成熟度差距分析（对照成熟 3A 管理系统）

> 2026-09-06 ｜ 负责人轮产出 ｜ 基准:FreeRADIUS 3.x/rlm 生态、运营商级商用 AAA(Alepo/Ray 一类)公开能力面、RFC 2865/2866/5176
> 事实来源:internal/domain/aaa、internal/domain/aaa/radius、internal/domain/aaa/billing、cmd/aaa、internal/app/grpc_aaa.go、migrations 000014/000030/000103、internal/httpapi/admin/aaa*.go 全量代码走读
> 结论先行:**认证环节零凭证校验是 P0 安全缺口**;会话管理与强制下线整体缺失是 P0 功能缺口。

## 1. 我方现状(代码事实)

- 认证:cmd/aaa 起 RADIUS 1812/1813,Access-Request 仅按 User-Name(LOID)查 lo_accounts 状态做 Accept/Reject。**User-Password/CHAP 属性完全不校验**;lo_accounts 无凭据列,aaa.Profile.Password 为无落库来源的死字段。
- 授权:PGAuthorizer 直查 PG(JOIN product_offers/qos_templates),ACTIVE 下发带宽串(FramedPool 承载)+固定 SessionTimeout 3600s;停复机仅改 DB 状态(Suspend/Resume + gRPC),**只影响下一次认证**,注释宣称的 Redis TTL 缓存未实现。
- 计账:Acct Start/Stop/Interim 全量转 CDR,PG 落库+Kafka 双写,kafka 状态回写+补偿循环(cdr_compensation_loop)已有;无唯一会话去重、无在线会话表、无僵尸会话清理。
- 管理面:admin 仅 4 只读路由(summary/lo-accounts/cdrs/auth-logs)+无凭据管理;NAS 侧仅全局共享密钥(config 一项),无 per-NAS 注册表。
- 可观测:auth_logs 仅 SUCCESS/FAILED 两列;话单 Kafka 失败有状态标记;认证日志写失败为静默降级(违反留痕红线)。

## 2. 差距矩阵(六类,按优先级)

### P0(安全与核心闭环,本波次落地)

| # | 成熟 AAA 基线能力 | 我方状态 | 缺口影响 |
|---|---|---|---|
| G1 | PAP 密码校验+凭据安全存储;CHAP(RFC 1994)/MS-CHAPv2 | 完全缺失,任何 ACTIVE LOID 均放行 | 任意用户可冒用他人 LOID 入网,计费错挂 |
| G2 | 认证失败限速与账号锁定(防爆破) | 无任何限制 | 可无限次撞库 |
| G3 | 在线会话表(radacct 模式):Start 建/Stop 关/Interim 更新,在线用户可查 | 无会话表,只有追加式 cdrs | 无法回答"谁在线",无法支撑下线与并发控制 |
| G4 | 并发会话数限制(simultaneous-use) | 无 | 一号多拨、计费争议 |
| G5 | 强制下线与停复机即时生效(CoA/Disconnect, RFC 5176) | 无;停复机靠 SessionTimeout 自然过期,最长延迟 1h | 欠费停机后用户仍可继续用满 1 小时 |
| G6 | 僵尸会话清理(NAS 重启/超时未更新判定掉线) | 无 | 在线数虚高、话单缺口无告警 |

### P1(运营完备,下一波次)

| # | 基线能力 | 我方状态 |
|---|---|---|
| G7 | per-NAS 注册表(设备+独立密钥+启停) | 仅全局密钥;设备域有 OLT 台账但未与 AAA 关联 |
| G8 | 厂商 VSA 下发真实限速(华为/中兴 rate-limit 等) | 带宽仅以字符串塞 FramedPool,设备端未必生效 |
| G9 | 话单与会话一致性对账+缺口告警+导出 | 有 Kafka 补偿,无对账报表与导出 |
| G10 | 认证成功率/在线数/NAS 健康监控与告警 | 仅 admin summary 静态页 |
| G11 | 凭据生命周期(重置密码/批量导入导出) | 无 |

### P2(远期对齐,登记不排期)

Radsec(RADIUS over TLS)/Diameter/EAP 系列、代理转发(realm)、多实例共享缓存(Redis 授权缓存,注释已预留)、时段策略与 FUP 超量降速、自助门户改密。

## 3. P0 波次计划(已派发两个专属会话)

- **A1 认证凭证与防爆破(G1+G2)**:lo_accounts 增加凭据(哈希存储);RADIUS 侧实现 PAP 校验+CHAP;连续失败锁定(计数+TTL 自动解锁);认证日志写失败必须留痕(对齐留痕红线)。
- **A2 在线会话与强制下线(G3~G6)**:新增在线会话表(Start 建/Stop 关/Interim 累加,唯一会话 ID 去重);并发会话上限(默认 1,账号级可配);停复机/强制下线经 CoA Disconnect 下发 NAS(不可达时显式标记+重试);僵尸会话后台清理(NAS 重启时间戳/超时阈值);admin 在线用户查询+强制下线路由。

派发纪律:任务单只含需求与验收规则,不含源码;各会话 worktree 开发,迁移号先 fetch+merge main 再定;合并走 worktree 协议(先 merge main 解冲突跑门禁,再回主树 ff-only)。
