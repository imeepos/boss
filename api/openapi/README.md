# REST OpenAPI 3

对外 REST 契约,APISIX 网关路由与 JWT 鉴权以此为准。

## 用户端(客户门户)

- `user.yaml` —— 聚合入口,`info/servers/tags` + 内联 `$ref` 指向 `user/` 下各域文件与 schemas。
- `user/` —— 按 7 类 tag 拆分的域文件,每文件控制在 200 行内:

| 文件 | tag | 接口范围 |
| --- | --- | --- |
| `user/auth.yaml` | Auth | 登录 / 注册 / 验证码 / 找回密码 / 实名认证 |
| `user/profile.yaml` | Profile | 我的 / 账号安全 / 通知订阅 / 语言 |
| `user/product.yaml` | Product | 产品套餐 / 套餐详情 / 增值服务 |
| `user/order.yaml` | Order | 订单(12 环节) / 退订 / 评价 |
| `user/billing.yaml` | Billing | 账单 / 缴费 / 充值 / 凭证 / 发票 |
| `user/customer-service.yaml` | CustomerService | 故障报修 / 投诉建议 / 智能客服 |
| `user/misc.yaml` | Misc | 消息 / 优惠券 / 用量 / 地址 / 自助排障 / 协议 |
| `user/schemas.yaml` | — | 全部共享 `components.schemas` 与 `securitySchemes` |

- `mock/` —— 按 `user.yaml` 契约的假数据 mock 服务,见 `api/mock/README.md`(mock 零依赖手工路由,不解析 yaml,拆分不影响)。

## 师傅端(装维门户)

- `worker.yaml` —— 聚合入口,servers 前缀 `/api/worker/v1`(与用户端 `/api/v1` 隔离)。
- `worker/` —— 按 tag 拆分的域文件:

| 文件 | tag | 接口范围 |
| --- | --- | --- |
| `worker/auth.yaml` | Auth | 登录 / 验证码 / 退出 |
| `worker/ticket.yaml` | Home/Ticket/Hall | 工作台 / 工单列表详情(12·6 环节) / 领取抢单 / 转单改约 / 签到导航 / 任务池 |
| `worker/scan.yaml` | ScanBind | 扫码绑定(环节 9) / 异常上报 / 拍摄取证 / 结果上报 / 激活(环节 10) / 电子签收 / 现场收款 |
| `worker/asset.yaml` | Asset | 拆机解绑 / 换件 / 旧件回收 / 领料借还 / 维护清单 / 测速资源 |
| `worker/profile.yaml` | Profile | 我的 / 绩效 / 排期 / 接单设置 / 满意度 |
| `worker/misc.yaml` | Misc | 消息 / 公告 / 排障手册 / 联系调度 / 安全上报 |
| `worker/schemas.yaml` | — | 全部共享 `components.schemas` 与 `securitySchemes` |

- `mock/worker/` —— 按 `worker.yaml` 契约的假数据 mock 服务(端口 8091),页面 `docs/worker/api.js`,对接约定 `docs/worker/API-INTEGRATION.md`。

## 拆分约定

- 域文件内对共享 schema 的引用统一写 `./schemas.yaml#/components/schemas/<Name>`,不写 `#/...`。
- `schemas.yaml` 内部 schema 间引用保持 `#/components/schemas/<Name>`(指向自身)。
- 新增接口只改对应域文件,不要回填 `user.yaml`;如新增 tag 再同步 `user.yaml` 的 `tags` 与聚合引用。
- 原 `user.yaml` 中同一 path 的 GET/POST 分块已合并为单一路径,multi-method 不再重复声明。

字段与状态枚举一律以 `docs/contract/{terms,fields,domain-map}.md` 为准。
