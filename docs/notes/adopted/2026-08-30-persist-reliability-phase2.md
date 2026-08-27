# 业务持久化可靠性整改(阶段2):并发领取、留痕事务化与静默降级显性化

日期:2026-08-30

## 决策

承接 2026-08-30-audit-sync-persistence(审计同步落库),本阶段把已确认会破坏业务留痕/任务可靠性的缺陷修完:

1. **订单环节推进原子化**:`orders.stage` 计数器前移与 `order_stages` 日志写入合并为单事务
   (复用 submitPartnerAtomic 的"复制 store 换 tx"模式)。崩溃或第二条语句失败不再产生
   "计数器到 N 而日志缺 N"的分叉单,该分叉此前会让重试恒撞 ErrIllegalTransition、订单永久卡死。
   `order.stage.done` 广播仍在提交后尽力发出(开放平台 outbox 自身幂等)。
2. **Webhook 投递原子领取**:`ListDue` 从普通 SELECT 改为单语句
   `UPDATE ... FOR UPDATE SKIP LOCKED`(沿用 provision ClaimTask 先例),领取即把
   `next_attempt_at` 推进 120s 租约窗口。多实例/重叠循环批次互不相交;投递器崩溃后行在
   租约到期自动重回待投——不丢失、不重复失控、不永久滞留。
3. **审计写入失败处理**:RecordAudit 与请求上下文解耦(WithoutCancel+5s 超时),瞬时失败重试一次,
   最终失败输出 `[audit] WRITE FAILED` 告警并附全量载荷供人工补记。承认的局限:PG 整体不可用期间
   审计仍无法持久(数据库就是事实源,不引入第二存储);载荷日志是最后防线而非替代。
4. **配置密文解密失败显性化**:五个通道(minio/sms/stripe/push/realid)统一走 decryptConfigSecret,
   解密失败输出 `[config-secrets] DECRYPT FAILED`(每字段热重载周期只报一次),返回空串保持原
   env 兜底语义;secretbox 兜底内置默认密钥时启动告警。

同时修复顺带发现的断链缺陷:fresh 投递行 http_status/last_error 为 NULL,ListDue/ListDeliveries
以 *int/*string 扫描必然报错——一旦环境开始使用 webhook 订阅,投递循环将每轮失败、任务永久滞留待投。
因当前环境 subscriptions=0 才未爆发。

## 边界裁定(验证结论,不整改项)

- **开放平台限流**:RPM 令牌桶为进程内短窗保护,重启清零可接受;日配额权威数据在 `open_usage_day`
  (PG upsert,跨实例一致)。部署约束单实例(compose 无 replicas),扩容前必须换共享限流存储。
- **对账渠道注册**:`ChannelSourceRegistry` 为启动期装配(stripe source 每次 boot 确定性重建),
  注册信息无运行期变更语义;批次与流水本身持久于 PG 且建批幂等——重启恢复无缺口。
- **内存态分类(确认后明确不整改)**:
  | 位置 | 性质 | 理由 |
  |---|---|---|
  | OpenRateLimiter.buckets | 保护性限流窗口 | 上方边界裁定;重启代价=短窗突发,日配额兜底 |
  | etlScanner.latest | 观察性缓存 | 下轮扫描重建,丢失无业务影响 |
  | domain/*/memory.go(order/aaa/customer 等) | 测试替身 | grep 确认无非测试引用,wiring 全部走 PGStore |
  | 前端主题/语言偏好 | localStorage | 不涉及服务端内存态 |

## 放弃了什么(被否决项)

- **给 open_webhook_deliveries 加"delivering"状态列做租约**:需迁移+管理面适配;现有列即可表达,
  零 schema 变更达到同等互斥语义。
- **advance 单语句 CTE(UPDATE...RETURNING)+INSERT**:省一次往返但可读性差、守卫逻辑分散;
  且 pgxmock 序列改动量与事务方案相同。
- **audit 失败写本地文件 WAL 再重放**:引入文件生命周期与容器卷管理,且 PG 不可用时业务请求
  本身也在失败,追补通道收益低于运维成本(见上方局限声明)。
- **admin pushconfig 解密失败的展示文案修正**:仅影响展示路径的边角(显示原始值),非本阶段
  高优先级,登记为遗留观察项不动 UI。

## 关联

- docs/notes/adopted/2026-08-30-audit-sync-persistence.md(阶段1)
- internal/domain/order/pg_advance.go / internal/domain/openplat/webhook_pg.go /
  internal/app/config_secrets.go / internal/pkg/httpx/httpx.go
