# 用户端 · 日常处理模拟清单（客户自助门户 · 50 轮）

> 目标：以"客户"角色为起点，逐轮模拟宽带用户**日常自助处理**的真实事件（每轮一个不同事件），
> 每个事件走**完整生命周期**，在用户端逐一找到对应功能点；凡生命周期中出现而页面/入口未覆盖的触达点，
> 即为需补全的页面/入口。本文档（含 `nav.js` 与 `docs/user/*.html`）即为补全《用户端》的推导依据与最终落地。
> 依据：`docs/BOSS综合业务支撑平台/BOSS综合业务支撑平台-需求全案.md` §2.3.17 / §3.2.17 客户自助门户（PORT-001~008）
> 与 §2.3.18 消息通知（NOT）、§SYS-001 密码策略、§CS-005 满意度回访。

## 〇、用户画像基线

| 维度 | 取值 |
|---|---|
| 角色 | 客户（个人/家庭宽带用户） |
| 入口 | 用户端 H5（`docs/user/`，底部 4 Tab：首页/产品/订单/我的） |
| 核心诉求 | 查余额/套餐/账单/用量、在线缴费/充值、报障追进度、改套餐/迁址/加购/退订、收通知、开发票、评价 |
| 需求锚点 | PORT-001 账户与余额、PORT-002 在线充值、PORT-003 报障与工单进度、PORT-004 服务变更、PORT-005 消息中心、PORT-007 自助注册 |

## 一、模拟总览（50 轮 · 每轮一个不同事件）

> 编号 U##；"现状"以 `docs/user/` 现有页面为权威，"有"= 已有页面/入口，"缺"= 需补全。

| 轮 | 事件（场景） | 现状 |
|---|---|---|
| U01 | 新用户自助注册开户 | 缺 `register` |
| U02 | 手机号+密码登录 | login 仅验证码，缺密码登录 |
| U03 | 忘记密码→短信重置 | 缺 `forgot` |
| U04 | 修改登录密码 | 缺 `security` |
| U05 | 更换绑定手机号 | 缺 `security` |
| U06 | 退出登录 | profile 有 |
| U07 | 查看套餐余额/有效期 | home 有，缺套餐详情入口 |
| U08 | 查看网络用量 | 缺 `usage` |
| U09 | 查看消息通知 | 缺 `messages` |
| U10 | 查看服务/故障公告 | 缺（并入 `messages`） |
| U11 | 浏览套餐列表 | products 有 |
| U12 | 查看套餐详情 | 缺 `product` |
| U13 | 在线下单新装 | 有（order.html 复用） |
| U14 | 加购增值服务 | 缺 `addon` |
| U15 | 升级/改套餐 | 缺 `change` |
| U16 | 迁址/移机 | 缺 `move` |
| U17 | 拆机/退订 | 缺 `cancel` |
| U18 | 静态 IP 申请 | 缺（并入 `change`） |
| U19 | 订单列表筛选 | orders 有 |
| U20 | 订单进度跟踪 | order 有 |
| U21 | 取消订单 | order 有"取消"，缺确认流程 |
| U22 | 催单反馈 | order 有"催单"，缺结果反馈 |
| U23 | 联系师傅 | order 有按钮，缺拨号/留言 |
| U24 | 工单完成满意度评价 | 缺 `rate` |
| U25 | 账单列表 | bills 有 |
| U26 | 账单明细 | 缺 `bill` |
| U27 | 在线缴费 | pay 有 |
| U28 | 支付结果 | 缺 `payresult` |
| U29 | 缴费凭证/收据 | pay 有"凭证"，缺 `receipt` |
| U30 | 电子发票 | 缺 `invoice` |
| U31 | 余额充值 | 缺 `topup` |
| U32 | 自动缴费/代扣签约 | 缺（并入 `bill`） |
| U33 | 欠费停机/复机状态 | 缺（并入 `myplan`） |
| U34 | 自助排障 | 缺 `diy` |
| U35 | 故障报修 | fault 有 |
| U36 | 报修进度跟踪 | 缺 `faultdetail` |
| U37 | 投诉/建议 | 缺 `complaint` |
| U38 | 在线客服 | 缺 `service` |
| U39 | 常见问题/帮助 | 缺 `help` |
| U40 | 实名认证/补登 | profile 有展示，缺核验流程 |
| U41 | 家庭地址管理 | profile 有"管理"，缺 `address` |
| U42 | 我的套餐/服务管理 | 缺 `myplan` |
| U43 | 通知订阅设置 | 缺 `notify` |
| U44 | 开票信息/发票抬头 | 缺（并入 `invoice`） |
| U45 | 用户协议/隐私政策 | login 有链接，缺 `agreement` |
| U46 | 消息已读/清空 | 缺（并入 `messages`） |
| U47 | 优惠券/营销活动 | 缺 `coupon` |
| U48 | 多语言切换（中/英/菲） | 缺 |
| U49 | 套餐对比 | 缺（并入 `product`） |
| U50 | 邀请好友/推荐 | 缺（并入 `coupon`） |

## 二、逐轮生命周期 ↔ 功能点 ↔ 现状

> 格式：`生命周期环节 → 应触达功能点 → 现状(有/缺)`；"✅有"=页面已承载，"✘缺"=需补全（已在落地产物补全）。

### 账号域（U01~U06）
- U01 自助注册：输手机号→验证码→设密码→实名→建档→可下单 → `register.html` ✘
- U02 密码登录：输手机号+密码→校验→失败锁定→登录 → `login.html` 增密码登录 ✘
- U03 忘记密码：验证码→设新密码→强制重登 → `forgot.html` ✘
- U04/U05 账号安全：改密/换绑→二次验证→生效 → `security.html` ✘
- U06 退出登录：清会话→回登录页 → `profile.html` ✅

### 首页状态域（U07~U10）
- U07 余额/有效期：状态卡→套餐详情下钻 → `home.html` ✅，`myplan.html` ✘
- U08 用量：周期用量/剩余流量 → `usage.html` ✘
- U09/U10 消息/公告：收余额/到期/停机/故障/优惠→已读/清空 → `messages.html` ✘

### 产品办理域（U11~U18）
- U11 套餐列表：浏览→筛选 → `products.html` ✅（seg 分组需可用）
- U12 套餐详情/对比：详情→加购/下单/对比 → `product.html` ✘
- U13 下单新装：选套餐→填地址→提交→生成订单 → 复用 `order.html` ✅
- U14 加购增值：选增值→订购→生效 → `addon.html` ✘
- U15 改套餐：选新套餐→生效方式(即时/预约)→差价结算 → `change.html` ✘
- U16 迁址：选新地址→资源核查→生成迁址工单 → `move.html` ✘
- U17 退订拆机：申请→扫码解绑(强制)→端口释放→账户停用 → `cancel.html` ✘
- U18 静态IP：申请→审批→配置下发 → 并入 `change.html` ✘

### 订单工单域（U19~U24）
- U19 订单筛选：全部/进行中/已完成/已取消 → `orders.html` ✅
- U20 订单进度：12 环节时间轴 → `order.html` ✅
- U21 取消订单：取消→端口释放→订单关闭 → `order.html` 增确认流程 ✘
- U22 催单：催单→反馈受理 → `order.html` 增反馈 ✘
- U23 联系师傅：拨号/留言 → `order.html` 增拨号 ✘
- U24 满意度评价：工单关闭→评分→<3 升级复核 → `rate.html` ✘

### 账务支付域（U25~U33）
- U25 账单列表：近N期/已缴/未缴 → `bills.html` ✅
- U26 账单明细：账期→费用项→合计 → `bill.html` ✘
- U27 在线缴费：选支付方式→支付 → `pay.html` ✅
- U28 支付结果：成功/失败/处理中 → `payresult.html` ✘
- U29 缴费凭证：凭证→电子收据 → `receipt.html` ✘
- U30 电子发票：开票→抬头→下载 → `invoice.html` ✘
- U31 余额充值：面额/自定义→GCash/Maya/卡/券 → `topup.html` ✘
- U32 自动缴费：签约→代扣→解约 → 并入 `bill.html` ✘
- U33 欠费停复机：欠费→停机→缴费→复机状态 → 并入 `myplan.html` ✘

### 故障支持域（U34~U39）
- U34 自助排障：选症状→引导→仍不行转报修 → `diy.html` ✘
- U35 故障报修：类型/地址/描述→提交 → `fault.html` ✅
- U36 报修进度：报修单→处理环节→完成 → `faultdetail.html` ✘
- U37 投诉建议：类型→描述→受理 → `complaint.html` ✘
- U38 在线客服：会话→转人工 → `service.html` ✘
- U39 帮助中心：FAQ→自助 → `help.html` ✘

### 个人与营销域（U40~U50）
- U40 实名核验：填信息→证件上传→人像比对→审核→已实名 → `verify.html` ✘（流程页），`security.html` 承载入口 ✅
- U41 地址管理：新增/设默认/删除 → `address.html` ✘
- U42 我的套餐：当前套餐/变更/加购/退订/停复机 → `myplan.html` ✘
- U43 通知订阅：退订营销/保留业务 → `notify.html` ✘
- U44 开票信息：抬头/税号 → 并入 `invoice.html` ✘
- U45 协议：用户协议/隐私 → `agreement.html` ✘
- U46 消息已读/清空 → 并入 `messages.html` ✘
- U47 优惠券：领券/可用/过期 → `coupon.html` ✘
- U48 多语言：中/英/菲切换 → `nav.js` 增语言切换（页内）✘
- U49 套餐对比 → 并入 `product.html` ✘
- U50 邀请好友 → 并入 `coupon.html` ✘

## 三、补全结论（缺失页面/入口）

| # | 缺失能力 | 关联轮次 | 落地页面 |
|---|---|---|---|
| 1 | 自助注册 | U01 | `register.html` |
| 2 | 密码登录 | U02 | `login.html`（增） |
| 3 | 忘记密码 | U03 | `forgot.html` |
| 4 | 账号安全（改密/换绑） | U04/U05 | `security.html` |
| 4b | 实名认证核验流程 | U40 | `verify.html` |
| 5 | 消息中心 | U09/U10/U46 | `messages.html` |
| 6 | 网络用量 | U08 | `usage.html` |
| 7 | 套餐详情/对比 | U12/U49 | `product.html` |
| 8 | 加购增值 | U14 | `addon.html` |
| 9 | 改套餐/静态IP | U15/U18 | `change.html` |
| 10 | 迁址移机 | U16 | `move.html` |
| 11 | 拆机退订 | U17 | `cancel.html` |
| 12 | 满意度评价 | U24 | `rate.html` |
| 13 | 账单明细/自动缴费 | U26/U32 | `bill.html` |
| 14 | 支付结果 | U28 | `payresult.html` |
| 15 | 缴费凭证 | U29 | `receipt.html` |
| 16 | 电子发票/开票信息 | U30/U44 | `invoice.html` |
| 17 | 余额充值 | U31 | `topup.html` |
| 18 | 自助排障 | U34 | `diy.html` |
| 19 | 报修进度 | U36 | `faultdetail.html` |
| 20 | 投诉建议 | U37 | `complaint.html` |
| 21 | 在线客服 | U38 | `service.html` |
| 22 | 帮助中心 | U39 | `help.html` |
| 23 | 地址管理 | U41 | `address.html` |
| 24 | 我的套餐/停复机 | U42/U33 | `myplan.html` |
| 25 | 通知订阅设置 | U43 | `notify.html` |
| 26 | 用户协议 | U45 | `agreement.html` |
| 27 | 优惠券/邀请 | U47/U50 | `coupon.html` |
| 28 | 多语言切换 | U48 | `nav.js`（增） |
| 29 | 订单取消/催单/联系师傅交互 | U21/U22/U23 | `order.html`（增） |

> 页内级缺口（不新增页面，回填到已有页面）：`products.html` seg 分组可切换、`home.html` 增消息/用量/套餐入口、`pay.html` 增充值/结果/发票入口、`fault.html` 增进度/排障/客服入口、`profile.html` 增我的套餐/地址/通知/发票/协议入口。

## 四、落地校验（验收证据）

> 新增页面 27 个，用户端共 36 个 HTML 页；`nav.js` 4 Tab（首页/产品/订单/我的）不变，
> 新增页面均通过首页宫格、我的-服务/设置、列表"更多"、账单/故障/订单详情等入口可达。

### 4.1 新增页面 ↔ 轮次 ↔ 需求锚点对账

| 页面 | 轮次 | 需求锚点 |
|---|---|---|
| register.html | U01 | PORT-007 自助注册 |
| forgot.html | U03 | SYS-001 密码找回 |
| security.html | U04/U05 | SYS-001 改密/换绑 |
| verify.html | U40 | 实名认证核验流程 |
| agreement.html | U45 | 用户协议/隐私 |
| usage.html | U08 | PORT-001 用量 |
| messages.html | U09/U10/U46 | PORT-005 消息中心 |
| product.html | U12/U49 | 套餐详情/对比 |
| addon.html | U14 | PORT-004 加购 |
| change.html | U15/U18 | PORT-004 改套餐/静态IP |
| move.html | U16 | PORT-004 迁址 |
| cancel.html | U17 | PORT-004 退订拆机 |
| rate.html | U24 | CS-005 满意度 |
| bill.html | U26/U32 | PORT-001 账单明细/自动缴费 |
| payresult.html | U28 | PORT-002 支付结果 |
| receipt.html | U29 | 缴费凭证 |
| invoice.html | U30/U44 | 电子发票/开票信息 |
| topup.html | U31 | PORT-002 在线充值 |
| diy.html | U34 | 自助排障 |
| faultdetail.html | U36 | PORT-003 报修进度 |
| complaint.html | U37 | 投诉建议 |
| service.html | U38 | 在线客服 |
| help.html | U39 | 帮助中心 |
| address.html | U41 | 家庭地址管理 |
| myplan.html | U42/U33 | PORT-004 我的套餐/停复机 |
| notify.html | U43 | NOT 通知订阅 |
| coupon.html | U47/U50 | 优惠券/邀请 |

### 4.2 已有页面回填（页内级缺口）

| 页面 | 回填内容 |
|---|---|
| login.html | 密码登录 tab、注册入口、忘记密码、协议链接 |
| home.html | 消息铃铛、套餐/账单/余额下钻、8 宫格（充值/用量/消息/客服） |
| products.html | 套餐详情/加购入口 |
| orders.html | 已完成订单"去评价" |
| order.html | 取消确认、联系师傅拨号、催单反馈、变更地址 |
| bills.html | 账单明细下钻、开发票、余额充值 |
| pay.html | 支付结果、凭证、开发票 |
| fault.html | 自助排障/在线客服、报修进度下钻 |
| profile.html | 我的套餐、消息、优惠券、发票、账号安全、通知、投诉、帮助、协议、语言切换 |

### 4.3 链接完整性

- 全量 `href` 与 `location.href` 目标页均存在于 `docs/user/`，无死链；
- `nav.js` 4 Tab 目标页（home/products/orders/profile）均存在。

### 4.4 页面 ↔ 需求锚点对账

| 需求 | 落地页面 |
|---|---|
| PORT-001 账户余额/账单/缴费/用量 | `home` / `bills` / `bill` / `usage` / `topup` / `pay` |
| PORT-002 在线充值 | `topup` / `payresult` |
| PORT-003 报障与工单进度 | `fault` / `faultdetail` / `diy` |
| PORT-004 服务变更 | `change` / `move` / `addon` / `cancel` |
| PORT-005 消息中心 | `messages` / `notify` |
| PORT-006 多语言 | `nav.js` 语言切换 |
| PORT-007 自助注册 | `register` |
| SYS-001 密码找回 | `forgot` / `security` |
| CS-005 满意度评价 | `rate` |

<hr>

> 本清单随 `docs/user/*.html` 与 `nav.js` 同步维护。
