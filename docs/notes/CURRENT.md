# 现行有效决策单页(CURRENT.md)

> 人工精选维护:AI 会话与新人开工前**只读这一页**即可拿到现行裁定,不必通读 83 篇 note。
> 依据 README.md 索引精选;被 Amended/取代的决策不在此列,文末有代谢清单。
> 新决策落地当天由作者把"一句话规则"追加到对应分组(与 README 索引同步)。

## 契约与域边界

- 三端 API 前缀分离:admin/user/worker 各自 /api/{face}/v1,门户状态落库,不跨端混用(2026-08-19, api-three-portal-prefix)
- 订单归属公司由安装地址判定,客户与法人无直接归属关系(2026-08-20, order-legal-entity-by-address)
- geo(国际地理基础数据)与 gis(联动/实体同步)分立两域(2026-08-18, geo-vs-gis-split)
- 附件删除一律软删(attachments.deleted_at),引用面不可穷举禁物理删(2026-08-25, attachment-soft-delete)
- API 文档 = 契约运行时聚合 openapidoc + Swagger UI(/base/apidocs),契约 YAML 即权威(2026-09-06, api-docs-openapidoc)
- ODN 物理层升级业务基础:地址覆盖关联→端口占用态→逻辑物理绑定三阶段推进;编码命名空间仍独立,桥接走关系表(2026-09-06, odn-business-linkage)
- 存量开户数据:线路 VLAN 四元组挂 ports(不挂账号);非 ODN 规范编码的光缆层级以 ports.legacy_path 单列承接;存量月数挂 lo_accounts.contract_months;不伪造历史订单(2026-09-07, legacy-vlan-on-ports)
- ODN 设备字典十类(SNW/OLT/ODF/OCC/ODB/OBD/SDB/SBD/PRT/TBP):OBD 归 ODB、SBD 归 SDB 为箱内部件扩展;导入域箱体设备可无城市分域唯一;资源链导入留空不猜填、规划态一律 PLANNED(2026-09-07, odn-box-types-import-chain)
- 投资口径维度归属:容量(分光户数)按设备维度只进城市/全网视图不摊网格;项目级总额(预算/材料成本)按明细金额占比分摊,零归属显式未登记;分光容量经 backfill-split 幂等回写 odn_device_split_capacity(000221,W5,split-capacity-investment-depth)

## 订单与状态机

- order_no 用 DB 序列生成(行锁计数表只用于发票 ARN)(2026-08-17, order-no-db-sequence)
- 下单幂等:可选 requestId 客户维度唯一,重放返回已有订单(2026-08-23, order-submit-idempotency)
- 订单推进计数器与环节日志同事务;webhook 投递 SKIP LOCKED 原子领取+租约(2026-08-30, persist-reliability-phase2)
- 端口预占:installing/inflight 占位语义见 adopted note,RESERVED 超时回滚(2026-08-28, port-reserved-installing-inflight)
- 业务时区 Asia/Manila:DB/容器存 UTC,展示归客户端,自然日界走 clock 包(2026-08-21, business-timezone)

## 数据与钱

- 充值余额是预存(只进不出);未来接消费须补 wallet 流水表+CHECK>=0(2026-09-03, portal-wallet-balance-semantic)
- 合成客户(负数 ID 隔离空间)拒绝真实充值,走 42200(2026-09-03, synthetic-customer-recharge-boundary)
- 支付链路:充值原子落账(RecordTopup 同事务)、webhook 隧道自愈、账单流水强制双挂(2026-08-26, payment-hardening)
- 套餐付费模式挂客户订购关系 lo_accounts,预付费办业务当时即收(2026-08-22, prepaid-postpaid-billing-mode)
- 客户可 1:N 链路(一客户多地址);quad_links 四码唯一约束以 000086 非空唯一为准(2026-08-20, db-dualtrack-convergence)
- LOY 积分跨域兑换用补偿模式(先扣后发失败回补),不做跨域同事务(2026-08-26, loy-minimal)
- 券模板/兑换/核销独立 promotion 域,billing 经 CouponDeductor 同事务核销(2026-08-26, promotion-domain)

## 渠道与密钥

- Stripe 卡通道:发起建意图/回调验签落账,pay_no 幂等;凭据仅存 biz_params 密文,env 兜底已移除(2026-08-23/2026-09-05, stripe-card-channel, stripe-config-backend-and-worker-charge)
- method='card' 双语义对账口径:柜面 POS(三要素非空)不进 Stripe 渠道对账范围(2026-09-07, payment-method-card-dual-semantics)
- 验证码短信:阿里云国际单通道起步,区号路由留国内注入口(2026-08-22, sms-channel-aliyun-intl)
- 实名二要素:阿里云 Id2MetaVerify 提交即核验,凭据未配保持人工(2026-08-24, realid-channel-aliyun-cloudauth)
- 开放平台密钥:AppId+Secret HMAC,Secret 原文落库(验签必需)(2026-08-22, open-platform-secret)
- 认证配置 secret:复用 biz_params + AES-256-GCM;解密失败必须显性报错(2026-08-21, auth-config-secret-storage)
- AAA LOID 凭据:lo_accounts.password_credential 密文落库(AES-256-GCM 密钥外置,CHAP 需可还原口令故单工哈希被否);未设密默认 Reject,BOSS_AAA_ALLOW_NO_CRED 迁移缓冲默认关;连续失败 5 次锁 15 分钟可配(2026-09-06, aaa-credential-storage)
- 菲律宾行政区划 PSGC 以 migration 全量内置,PSGC 事实优先于甲方规范笔误(2026-08-18, psgc-builtin-migration)

## 端与发布

- 移动端 Android = Kotlin+Compose 双 app 独立 Gradle 工程(2026-08-19, mobile-android-init)
- 师傅端前端 = web/desktop Vite MPA(早期 worker-web 裁定已被取代)(2026-08-19, desktop-worker-web-vite-mpa)
- 管理端桌面 = Tauri 2 壳内嵌 admin dist(2026-08-19, desktop-tauri-wrap-admin)
- apprelease 发布域:双门槛升级判定+确定性灰度分桶,APK 入 MinIO(2026-08-28, app-release-domain)
- user Android 发布签名:独立 keystore 不入库,apksigner 指纹验签门禁(2026-08-27, user-android-release-signing)
- MCP 接入 = 本地 stdio server(cmd/bossmcp),3 工具覆盖 user+worker 全端点(2026-09-06, mcp-server-user-worker)
- 三端全量支持 API key 鉴权,主体边界不跨端(2026-08-25, api-key-three-portal)

## 流程与协作

- worktree 合并协议:冲突 feature 侧消化、收尾四步、中央文件 append-only;连续 3 次 re-sync 未合入即止损(2026-08-22/28, worktree-merge-protocol, worktree-merge-circuit-breaker)
- 自定义角色:内置 7 角色只读,后端只收权限码全集全量替换(2026-08-22, custom-roles)
- 审计写入改同步落库(PGWriter),崩溃不丢关键留痕;写失败 [audit] WRITE FAILED 告警(2026-08-30, audit-sync-persistence)
- 数据核查红线:先查库再接口复核;验收造数 acc_ 不过夜+孤儿巡检三通道(2026-08-29, audit-closeout-rulings)
- 开户工作台聚合页 /bss/onboarding:建档→实名→下单→派单一页完成(2026-08-29, onboarding-workbench)
- 新建 worktree .env 自动接入:post-checkout 钩子补齐(主 .env 优先,example 兜底,绝不覆盖);clone 后一次性引导 sh scripts/env-hooks-init.sh(2026-09-04, worktree-env-auto-provision)
- 前端体验打磨:有现成组件一律复用(pickers/Dropdown/ConfirmDialog/StatusTag/DataTable/Card 等),禁止原生 select;逐页打磨每条验收须真实走查留证;选择器族改动只动 components/pickers 且对外 API 只加不改(2026-09-08, ux-polish-component-adoption)

## 已代谢(勿再引用原文裁定)

- server-ts 已移除,职责归 Go 实体(2026-08-17 立项→2026-08-21 移除)
- 师傅端 worker-web 方案被 desktop 收编(2026-08-19 同日 Amended)
- 数字孪生可视化 v1 Canvas 树 → 现行为 v4(PGIS 真地图+实时视域+trend 曲线)
- uq_quad_links_customer 唯一索引已被 1:N 裁定撤销(000088/000097)
