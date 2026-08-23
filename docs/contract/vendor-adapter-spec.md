# 供应商适配器规范(_vendor-adapter-spec_)

> 权威源:本文是外部系统(支付/短信/实名/税务/地图/设备/AAA 等)接入 BOSS 的适配器规范。
> 与代码冲突时以本文为准;推翻本文需走 adopted note。
> 背景:三年路线图 2028 Q3「Q4 开放平台与互操作」交付物之一;
> 执行计划见 `docs/plan/q4-open-platform-plan.md` M3。

## 0. 核心原则(七条,全部来自已有裁定)

1. **事实源在内**:订单/账务/客户等核心事实只落 BOSS 自己的表;供应商回执是输入,
   不是账。外部系统(含开放平台集成方)**不得直接写入核心事实表**(季度红线)。
2. **薄 adapter,不 vendor SDK**:供应商侧只封装最少调用面(HTTP/SNMP/...),
   不引入长尾 SDK 依赖(2026-08-20 网关形态裁定 + 2026-08-23 Stripe 修正)。
3. **密钥引用不存值**:凭据一律环境变量或 `biz_params` 配置引用(热更);
   适配器启动时解析,永不明文落库(密钥制度见 adopted notes)。
4. **未配置即降级**:凭据缺失 → 适配器不注册,调用方走既有降级路径
   (如 realid 落 PENDING 人工核验、tax 走 manual 回填、stripe intent 400)。
   不允许"缺凭据就 panic/启动失败"。
5. **幂等在调用方兜底**:对外发起带幂等键(如 stripe intent 用 payNo);
   供应商重投/重试不得产生重复业务动作(pay_no 唯一约束、open_webhook_deliveries
   的 (subscription_id,event_id) 唯一约束,同一思想)。
6. **回调先验签**:一切 webhook/回调先验签(HMAC/签名头)再落账;
   验签失败拒绝,不进业务。
7. **沙箱一等公民**:每个适配器必须支持 sandbox 模式(见 §3),
   供开放平台沙箱环境(M4)与集成方自助验收使用。

## 1. 接口形状(Go 约定)

适配器 = 一个窄接口 + 一个厂商实现 + 注册处;**接口定义在域侧(internal/domain/...)**,
实现放 `internal/pkg/<vendor>`(纯协议)或域内(强业务语义时)。

```go
// 域侧定义能力接口(以 billing.TaxGateway 为范本)
type TaxGateway interface {
    Jurisdiction() string                                       // 路由键:CN / PH
    Channel() string                                            // 渠道:leqi / bir_eis
    Issue(ctx context.Context, inv Invoice) (TaxReceipt, error)
}

// 注册处:装配期注入,运行期只读;按路由键取,未注册返回 nil → 调用方降级。
type TaxGatewayRegistry struct{ ... }
```

约定:

- 接口方法 ≤4 个;参数用域内已有结构(Invoice),禁止把供应商 DTO 透传进域层。
- 返回 `(结果, error)`;**结果里必须携带供应商回执的原始标识**(税票号/支付意图 ID),
  供对账与审计回放。
- 网络重试/退避在**适配器内**处理(建议 3 次指数退避);业务幂等在**域层**兜底。
  Webhook 向外投递的重试由开放平台 outbox 统一负责(000125),适配器不重复造。

## 2. 错误分类

| 类别 | 判定 | 处理 |
|:-----|:-----|:-----|
| 配置缺失 | 凭据/endpoint 为空 | 不注册,调用方降级(§0.4) |
| 请求无效 | 4xx + 供应商明确拒绝 | 不重试,业务侧落 FAILED/人工 |
| 暂时失败 | 5xx / 超时 / 网络错误 | 适配器内退避重试,仍失败则报错由调用方决定(轮询补偿/死信) |
| 验签失败 | 回调签名不符 | 拒绝并记审计,不进业务 |

## 3. 沙箱模式

- 适配器构造参数带 `sandbox bool`(或配置项);sandbox 实现:
  不打真实供应商,按**契约样例**(fixture)返回确定结果,覆盖成功/失败/异步三条路径。
- fixture 与契约测试共用(见 M5 回放工具),放 `testdata/` 并在本文 §5 登记。
- 生产/沙箱路由:开放平台 `open_apps.sandbox=true` 的应用产生的调用,
  装配层必须路由到 sandbox 适配器实例。

## 4. 现有适配器对齐评估(2026-08-22 盘点)

| 系统 | 接口 | 实现 | 状态 | 与规范的差距 |
|:-----|:-----|:-----|:-----|:-------------|
| 支付(卡) | billing 域 + `internal/pkg/stripe` 薄 HTTP | Stripe | 已上线(000113/回调验签幂等) | 无 sandbox 模式 → M4 补 |
| 短信 | `sms.Sender`(E.164 区号路由) | 阿里云国际 + log 通道 + dynamic 热更 | 已上线 | log 通道可当 sandbox 用,需在 §3 语义下正名 |
| 实名 | `realid.Verifier`(PASS/FAIL) | 阿里云 CloudAuth + dynamic | 已上线 | 无 PASS/FAIL sandbox fixture → M4 补 |
| 税务 | `billing.TaxGateway`(属地路由 + registry) | manual/leqi(CN)/bir_eis(PH) | 接口就绪,网关按属地渐进 | leqi/bir_eis 实现方需遵守 §1/§2 |
| 设备采集 | `device.Poller` + snmpGetter 抽象 | gosnmp v2c(厂商 OID profile 可配) | 已上线 | OID profile 即"厂商差异隔离",符合规范 |
| 地图/GIS | 域内聚合(gis.go,无外部地图供应商) | 无 | 未对接 | 外部地图供应商(导航/地理编码)留待真实需求,届时按 §1 新增域侧接口 |
| AAA | `aaa.Authorizer` 等域内接口 | RADIUS/AAA 上游 | 域内 | 上游协议对接走 provision/quadlink 既有通道,不另立适配器 |

## 5. 新增供应商 checklist(评审门槛)

1. 域侧窄接口(方法 ≤4,域内结构出入参)评审通过。
2. 密钥:环境变量或 `biz_params` 引用;未配置降级路径明确。
3. 幂等键与回调验签方案写进接口注释。
4. sandbox 实现 + fixture(成功/失败/异步)与单测。
5. 错误按 §2 分类;重试策略注明(适配器内 or 调用方补偿)。
6. 在本文 §4 表登记一行(系统/接口/实现/状态)。
7. 契约联动:涉及新端点的同步 `api/openapi/` + `docs/contract/fields.md`。

## 6. 明确不做

- 不做支付聚合商/短信聚合商形态(2026-08-20 裁定)。
- 不为"将来可能对接"预建空适配器;地图等按真实需求立项。
- 适配器不做业务编排;编排归域服务/automation,事实归状态机 + PostgreSQL。
