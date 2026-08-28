# 决策记录（Agent Notes）

> 制度依据 dsh-codebase-wisdom 01-foundation/09 与 07-design/02：每个不可逆决策一篇 dated note，记 why 与放弃了什么。

## 规则

1. 任何不可逆/难逆决策（密钥入库、双模型并存、域分合、序列/ID 方案、内置数据方式），落地当天写一篇 `yyyy-mm-dd-topic-title.md` 进对应 lifecycle 目录。
2. note 结构：`# 标题` / `日期` / `决策` / `why` / `放弃了什么（被否决项）` / `关联`。不写实现细节，实现看代码。
3. 决策被推翻时**不删除**原文，在文中加 `> Amended yyyy-mm-dd: ...` 指向新 note（决策代谢：冻结 + cross-link + amend，不销毁）。
4. ADR（docs/ADR-*.md）仍保留给架构级大决策；note 记"日常但不可逆"的裁定。两者互链。
5. 评审 finding 的编号沿用现有体系：架构评审 `发现 N.M`（architecture-review.md），契约对账 `D/A/B/E/G#`（contract/alignment-audit.md），事后复盘 `docs/postmortem/000N-*`。

## 目录

- `adopted/` 已采纳并生效的决策
- `rejected/` 调研后否决的方案（记否决理由，防止重提）

## 索引

> 排序：日期升序，日期相同按文件名排序；Amended/被取代项在原位标注，不下沉。
> 行格式：`日期 ｜ 决策摘要 ｜ 文件路径`。

| 日期 | 决策 | 文件 |
|---|---|---|
| 2026-08-17 | order_no 改 DB 序列生成 | adopted/2026-08-17-order-no-db-sequence.md |
| 2026-08-17 | server-ts 已移除，职责迁移至 Go 实体 | adopted/2026-08-17-server-ts-entity-mirror.md |
| 2026-08-18 | app.env 固定密钥直接入库 | adopted/2026-08-18-app-env-in-repo.md |
| 2026-08-18 | geo 与 gis 分立两个域 | adopted/2026-08-18-geo-vs-gis-split.md |
| 2026-08-18 | 菲律宾行政区划以 migration 全量内置 | adopted/2026-08-18-psgc-builtin-migration.md |
| 2026-08-18 | 发票 ARN 发号采用行锁计数表（非 PG SEQUENCE） | adopted/2026-08-18-tax-invoice-arn-numbering.md |
| 2026-08-18 | 发票税务属地：多属地网关并存，系统发票无独立法定效力 | adopted/2026-08-18-tax-jurisdiction-china-shudian.md |
| 2026-08-18 | 用户端门户客户 JWT 复用 auth.Manager 签发（与 2026-08-19 三端前缀分离配套） | adopted/2026-08-18-user-portal-customer-jwt.md |
| 2026-08-19 | admin API=CORS 直连绝对地址，移除 Vite 代理与壳内反代 | adopted/2026-08-19-admin-api-direct-cors.md |
| 2026-08-19 | 三端 API 前缀分离(admin/user/worker) + 门户状态落库 + mock 移除 | adopted/2026-08-19-api-three-portal-prefix.md |
| 2026-08-19 | 管理端桌面客户端=Tauri 2 壳内嵌 admin dist，/api 壳内反代 | adopted/2026-08-19-desktop-tauri-wrap-admin.md |
| 2026-08-19 | 师傅端前端=web/worker 早期裁定（同日被 web/desktop 取代,见原文 Amended 段） | adopted/2026-08-19-worker-web-vite-mpa.md |
| 2026-08-19 | 师傅端前端=web/desktop Vite MPA 收编草稿，API 同源 /api/worker/v1 | adopted/2026-08-19-desktop-worker-web-vite-mpa.md |
| 2026-08-19 | 师傅端门户 worker JWT:同密钥、独立 claims、issuer=boss-worker 隔离 | adopted/2026-08-19-worker-portal-jwt.md |
| 2026-08-19 | 移动端 Android=Kotlin+Compose 双 app 独立 Gradle 工程 | adopted/2026-08-19-mobile-android-init.md |
| 2026-08-19 | 移动端门户缺失端点补齐:评价/地址映射/第三方网关姿态 + 演示种子 | adopted/2026-08-19-user-portal-mobile-endpoints.md |
| 2026-08-20 | 数据库双轨收敛:portal 唯一权威/合成 ID 负数段/四码部分唯一索引/实名统一 verifications | adopted/2026-08-20-db-dualtrack-convergence.md |
| 2026-08-20 | ODN 地理空间编码对齐:PRV/NodeCode 由 PSGC 派生映射不改权威数据,odn 无源物理层为待建域,端口码 P 前缀冲突待裁定 | adopted/2026-08-20-odn-geospatial-encoding-alignment.md |
| 2026-08-20 | 订单归属公司由安装地址判定,客户与公司无直接归属关系 | adopted/2026-08-20-order-legal-entity-by-address.md |
| 2026-08-20 | 生产域名 boss.ymm.cn + ingress TLS(cert-manager)终结 | adopted/2026-08-20-prod-domain-tls.md |
| 2026-08-20 | 真实短信/支付渠道:域内网关抽象,按属地选商,凭据未到不 vendor SDK | adopted/2026-08-20-sms-payment-channel.md |
| 2026-08-21 | 认证配置 secret 存储:复用 biz_params + AES-256-GCM 密文,自检 v1 只做完整性校验 | adopted/2026-08-21-auth-config-secret-storage.md |
| 2026-08-21 | 数据备份迁移(000095):本地磁盘 gzip JSONL 归档 + ON CONFLICT DO NOTHING 追加恢复,只补不删,进程内串行 | adopted/2026-08-21-backup-local-jsonl.md |
| 2026-08-21 | business-timezone 业务时区=Asia/Manila + DB/容器 UTC + 展示归客户端 + 自然日界走 clock 包 | adopted/2026-08-21-business-timezone.md |
| 2026-08-21 | customers 表新增 customer_code 字段:四码 customerCode 落码,前缀 C-,展示冗余不影响 customer_id 对账权威 | adopted/2026-08-21-customer-code-in-quadlink.md |
| 2026-08-21 | hostctl sidecar + MinIO 密钥热轮换 admin API(HMAC 鉴权) | adopted/2026-08-21-hostctl-sidecar-rotate.md |
| 2026-08-21 | MinIO root 密码改 bind-mount 文件注入(不动明文),bucket 统一策略+生命周期 | adopted/2026-08-21-minio-secret-file-mount.md |
| 2026-08-21 | 移除 server-ts 并更新相关文档 | adopted/2026-08-21-remove-server-ts.md |
| 2026-08-22 | 自定义角色(000100):内置 7 角色只读,模板复用放前端,后端只收权限码全集全量替换 | adopted/2026-08-22-custom-roles.md |
| 2026-08-22 | 数字孪生板块可视化升级 v2:PGIS 真地图(OL+OSM 瓦片起步)+ 新增 /gis/points 接口 + charts/ 共享 SVG 组件;v1 八级树形 Canvas 方案被推翻 | adopted/2026-08-22-intel-visualization-upgrade.md |
| 2026-08-22 | 数字孪生板块下一期 v3:tileserver-gl 自建 PMTiles + OL dark theme 双瓦片源(CartoDB Dark Matter)+ 经营分析去掉原表格页签;前端 maps/tile-source.ts 抽工厂 | adopted/2026-08-22-intel-viz-next-phase.md |
| 2026-08-22 | 数字孪生板块下一期 v4:GIS 视域点位实时计算 + 主题持久化 + 报告 trend 曲线(新增 /reports/history + LineTrend SVG 折线);通用 useLocalStorage hook | adopted/2026-08-22-intel-viz-v4-phase.md |
| 2026-08-22 | 开放平台应用密钥(000122):AppId+Secret HMAC 签名,Secret 原文落库(验签必需);只存哈希/应用层加密被否决 | adopted/2026-08-22-open-platform-secret.md |
| 2026-08-22 | 套餐付费模式挂客户订购关系(lo_accounts),预付费办业务当时即收(与环节4合流);套餐级模式/出账日批扣被否决 | adopted/2026-08-22-prepaid-postpaid-billing-mode.md |
| 2026-08-22 | 验证码短信落地:阿里云国际短信单通道起步(+86/+60),区号路由留国内通道注入口 | adopted/2026-08-22-sms-channel-aliyun-intl.md |
| 2026-08-22 | worktree 多分支合并协议:冲突在 feature 侧消化(合前反向同步+合完立刻同步其余),中央文件 append-only,收尾四步防丢码;配套 scripts/worktree-sync.sh | adopted/2026-08-22-worktree-merge-protocol.md |
| 2026-08-23 | 下单幂等键(000116):可选 requestId 客户维度唯一,重放返回已有订单;同客户同地址非终态拒单方案被否决 | adopted/2026-08-23-order-submit-idempotency.md |
| 2026-08-23 | Stripe 卡收单通道:发起建意图/回调验签落账,pay_no 幂等,密钥未配即降级 | adopted/2026-08-23-stripe-card-channel.md |
| 2026-08-24 | 实名二要素自动核验:阿里云实人认证 Id2MetaVerify,提交即核验,凭据未配保持人工 | adopted/2026-08-24-realid-channel-aliyun-cloudauth.md |
| 2026-08-25 | 三端全量支持 API key 鉴权:worker 端补 worker 主体密钥,主体边界不跨端,对接 AI 操作系统 | adopted/2026-08-25-api-key-three-portal.md |
| 2026-08-25 | 后台提醒中心:广播+读回执/30s 轮询/来源域 handler 层接入 | adopted/2026-08-25-admin-notify-broadcast-read-receipt.md |
| 2026-08-25 | 附件删除采用软删除(attachments.deleted_at,000093):引用面不可穷举禁物理删,MinIO 对象保留可审计 | adopted/2026-08-25-attachment-soft-delete.md |
| 2026-08-26 | 支付链路健壮性收口:充值原子落账(RecordTopup 流水+余额同事务)、webhook 隧道自愈(期望 URL+周期比对,UPDATE 保密钥/CREATE 换密钥落 biz_params)、账单流水强制双挂(000148) | adopted/2026-08-26-payment-hardening.md |
| 2026-08-26 | Stripe 测试环境接线:webhook endpoint 经 REST API 创建,隧道/凭据固定入 102 compose;充值页银行卡选项对齐 Android 移除 | adopted/2026-08-26-stripe-test-env-wiring.md |
| 2026-08-26 | Stripe 支付配置页化:沿用 biz_params+Dynamic 热更(60s),env 降级兜底;apiKey/whsec 分组密文存储,自检真实探活余额 | adopted/2026-08-26-stripe-config-page.md |
| 2026-08-26 | 最小 LOY 积分域(000104):账本+流水+积分换券;跨域兑换用补偿模式(先扣后发,失败回补),不做跨域同事务 | adopted/2026-08-26-loy-minimal.md |
| 2026-08-26 | 新建 promotion 营销促销域(000102):券模板/兑换/转赠/缴费抵扣核销从 userdata 独立;billing 经 CouponDeductor 注入实现同事务核销 | adopted/2026-08-26-promotion-domain.md |
| 2026-08-27 | 税务属地配置挂法人(000109):legal_entities.tax_jurisdiction/tax_channel 为配置源,开票经 bills.legal_entity_id 快照落票;区域定属地/独立配置表/手工指定被否决 | adopted/2026-08-27-tax-jurisdiction-on-legal-entity.md |
| 2026-08-27 | 设备更换单接入执行流:asset 域自治状态机(PENDING→assign→DOING→complete→DONE/FAILED),不生成派单工单(dispatch_tickets.order_id 1:1 订单强约束不动);师傅端经 /api/worker/v1/replacements 两端点接入,完成落既有 worker_replace_logs;放宽 order_id/换新套订单 12 环节/FAILED 回退被否决 | adopted/2026-08-27-replacement-ticket-flow.md |
| 2026-08-27 | 侧边栏菜单按实际内容重划 16 组(key/path/权限码不变,aaa 吸并 alarm,新增 worker/cms/channel/system 四组);顶栏分组主导航移除,分组导航只在侧栏;domain-map D 列维持原型分组不回填 | adopted/2026-08-27-sidebar-regroup.md |
| 2026-08-27 | user Android 发布签名与版号:独立 release keystore(不入库)+apksigner 指纹验签门禁;首发 versionCode=2/0.1.0(历史内测占 1);cleartext 白名单收敛;POST_NOTIFICATIONS 立项 | adopted/2026-08-27-user-android-release-signing.md |
| 2026-08-28 | 客户端版本发布域 apprelease(000137):双门槛升级判定+确定性灰度分桶;APK 入 MinIO;免登录检查挂各端 pub 组;不做差分/渠道/热修复 | adopted/2026-08-28-app-release-domain.md |
| 2026-08-28 | worktree 合并协议止损线:同一分支连续 re-sync 达 3 次未合入即停止竞速,推送保安全后与活跃会话错峰;无限重试(活锁)与中央合入队列被否决 | adopted/2026-08-28-worktree-merge-circuit-breaker.md |
| 2026-08-29 | 客户中心审计收尾裁定:实名三表 000059 归一为既定设计+PASS 一致性门禁;乱序 5 单补正+advance 前置守卫;address 288 双四码=0009 生命周期合法;投诉 orderId 维持可空;37 无评价单豁免;验收造数自清理+巡检门禁三通道;核查先查库再接口/行数磁盘余量/并行基线三项流程固化 | adopted/2026-08-29-audit-closeout-rulings.md |
| 2026-08-30 | 审计写入改同步落库:生产装配 PGWriter 替代进程内队列 AsyncWriter,关键留痕崩溃/队列满不丢 | adopted/2026-08-30-audit-sync-persistence.md |
| 2026-08-30 | 业务持久化可靠性阶段2:订单推进计数器与环节日志同事务、webhook 投递 SKIP LOCKED 原子领取+租约、审计写失败重试+全量载荷告警、配置密文解密失败显性化;限流/渠道注册边界与内存态分类登记 | adopted/2026-08-30-persist-reliability-phase2.md |
| 2026-09-03 | 导入权限边界:地址使用 menu:importer,Geo 使用 menu:geo,暂不追加地址双权限 | adopted/2026-09-03-importer-permission-scope.md |
| 2026-09-03 | 导入任务登记幂等键 clientKey 与下单幂等键 requestId 语义区分(upsert 写入 vs 查询返回,作用域/命名不统一) | adopted/2026-09-03-import-task-idempotency.md |
| 2026-09-03 | 充值余额是预存(只进不出)不是可用余额;未来接消费需补 portal_wallet_transactions 流水表+CHECK>=0 约束 | adopted/2026-09-03-portal-wallet-balance-semantic.md |
| 2026-09-03 | 合成客户(隔离空间负数 ID)充值边界:拒绝(走 42200);不动 FK、不建旁路;真实收费场景不覆盖合成客户 | adopted/2026-09-03-synthetic-customer-recharge-boundary.md |
| 2026-09-03 | 支付链路收口后续:隧道驻留 + 演示门户 + 缴费口径 + 余额/合成客户 6 项验收收尾汇总(commit 留证 + 102 实测留证) | adopted/2026-09-03-payment-followup-rollout.md |
| 2026-09-04 | 真实环境证据补录:102 隧道 URL 变化→Stripe endpoint 自愈、用户门户真实登录→4 条缴费记录、bossctl 运维端点冒烟 | adopted/2026-09-04-real-environment-evidence.md |
| 2026-09-05 | Stripe 配置后端化(去 env 兜底,凭据仅存 biz_params)+ 支付方式 App 端配置驱动(未配默认线下)+ 师傅端现场收款完整落账(payments 新增 offline 枚举) | adopted/2026-09-05-stripe-config-backend-and-worker-charge.md |

> 维护规则：新增 note 当天同步本表；同一日多条按文件名升序；Amended 项不移动原位（决策代谢可追溯）。