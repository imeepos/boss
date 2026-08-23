# 开放平台接入指南(open 端)

> 面向外部集成方(支付/短信/实名/税务/地图/设备及渠道系统)的自助接入文档。
> 契约权威源:`api/openapi/open.yaml`;管理面:`/api/admin/v1/openplat/*`(menu:openplat)。
> 执行计划:`docs/plan/q4-open-platform-plan.md`;供应商侧规范:`docs/contract/vendor-adapter-spec.md`。

## 1. 申请凭证

管理端「开放平台」创建应用,获得(仅返回一次,请妥善保存):

- `AppId`:公开标识,形如 `op_0123456789abcdef`
- `Secret`:签名密钥,形如 `ops_<32hex>`

应用级参数:每分钟限流(RPM,默认 60)、日配额(默认 10000)、沙箱标记。
凭证丢失只能停旧建新,不支持原位找回。

## 2. 请求签名(HMAC-SHA256)

每个请求带四个头:

```
X-BOSS-AppId:      op_xxx
X-BOSS-Timestamp:  1755859200        # Unix 秒,偏差 >5 分钟拒绝
X-BOSS-Nonce:      任意随机串
X-BOSS-Signature:  hex(HMAC-SHA256(Secret, canonical))
```

签名串(五处 `\n` 连接,body 为空时其哈希仍参与):

```
AppId \n METHOD \n /api/open/v1/path(不含 query) \n timestamp \n nonce \n hex(sha256(body))
```

错误码:缺头/验签失败/时间戳超窗 → 401;超 RPM → 429(`Retry-After: 1`);
超日配额 → 429(`X-BOSS-Quota-Exceeded: true`)。

## 3. 开放 API v1(端点清单见 open.yaml)

| 端点 | 说明 |
|:-----|:-----|
| `GET /api/open/v1/ping` | 连通性/验签自检,返回 appRowId 与沙箱标记 |
| `GET /api/open/v1/orders/{orderNo}` | 订单只读投影(orderNo/status/stage/offerId/createdAt) |
| `GET /api/open/v1/sandbox/samples` | 沙箱样例清单(样例订单号 + Webhook 回放样例) |

**数据边界**:开放面只读;外部系统不得直接写入订单、账务等核心事实表。

## 4. Webhook 订阅

1. 管理端为应用登记订阅(事件类型 + HTTPS 回调端点)。
2. 事件发生(如 `order.stage.done`,订单 12 环节每推进一格发一次,
   幂等键 `orderNo:stage:N`)后平台 POST 到回调端点,头:

```
X-BOSS-Event:      order.stage.done
X-BOSS-EventID:    ORD-20250817-001:stage:9
X-BOSS-Timestamp:  1755859200
X-BOSS-Signature:  t=1755859200,v1=hex(HMAC-Secret, timestamp+"\n"+hex(sha256(body)))
```

3. 接收方**必须验签**后再执行业务;同 EventID 重复投递直接幂等返回 2xx。
4. 非 2xx/超时按 30s×2ⁿ 退避重试(封顶 1h),6 次后进死信,
   可由管理端 requeue 或发测试事件 `openplat.test` 自检。

## 5. 沙箱验收(自助)

1. 申请 **sandbox=true** 的应用凭证(沙箱应用只可见 `SBX-*` 样例订单,
   生产订单双向不可见)。
2. 跑自助验收工具(零依赖,Node>=22):

```bash
node scripts/openplat-selftest.mjs \
  --base http://<环境地址> \
  --app op_xxx --secret ops_xxx
```

3. 工具覆盖五项:签名请求 ping、沙箱样例清单、样例订单查询、
   沙箱/生产隔离(404)、Webhook 验答回放(本地复算 v1 与样例比对)。
   全部 PASS 即完成沙箱验收;换生产凭证时加 `--skip-prod-check` 重跑前三项。

## 6. 版本策略

- 前缀 `/api/open/v1`;不兼容变更发 v2,旧版本至少保留两个季度。
- 兼容新增字段不升版本;集成方必须容忍未知字段。
