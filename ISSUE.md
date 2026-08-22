# ISSUE.md（上游/工具问题清单）

## 后端·order 12 环节全流程模拟(2026-08-22 bossctl CLI 演示发现)

- **已修复(2026-08-21, d397e40)｜行为缺口｜置备资产不回填 tag 双向绑定**:`POST /provision/assets`(internal/domain/asset/pg_write.go::CreateAsset)只写 `assets.tag_id`,不回填 `tags.bound_asset_id`/`tags.status`。而环节9 扫码核对(internal/domain/quadlink/pg_scan.go::VerifyScan)要求 `tags.bound_asset_id` 非空且等于 quad_link.asset_id,否则报 40920"扫码与预绑定不一致"。e2e 测试(internal/app/e2e_pg_integration_test.go:224)是建 tag 时手工传 `BoundAssetID + Status="BOUND"` 才绕过。→ 置备端点或 CreateAsset 应在 tag_id 非空时同步 `UPDATE tags SET bound_asset_id=$asset, status='BOUND'`,与 e2e 口径对齐。

- **已修复(2026-08-21, 756d21c)｜行为缺口｜自动化 applyTag 的四码不带资产**:`internal/domain/order/pg_workflow.go:125` 构造 `QuadLinkBindReq` 时 AssetID 恒为 0(自动化链路没有"选资产"步骤),而 VerifyScan 要求 `tags.bound_asset_id == link.asset_id` → **走自动化链路(POST /orders/:no/charge)的订单,扫码环节永远 MISMATCH**。只有 e2e 手工预绑定(CreateLink 带 AssetID)能通。→ 要么 applyTag 自动挑一个 IN_STOCK 未绑资产入库四码,要么 VerifyScan 对 asset_id=0 的 link 做兜底(扫码时回填而非比对失败)。这是自动化链路 vs 手工链路的真实分歧,阻塞所有真实订单走完 12 环节。

- **已修复(2026-08-21, a6c5d98)｜行为缺口｜worker/admin activate 不对称,段11/12 无独立推进入口**:worker 端 `POST /tickets/:no/activate`(internal/httpapi/worker/scan.go:224)只调 `Order.ActivateUser` 推段10;admin 端 `POST /tickets/:no/activate` 调 `Automation.AutoPostScan` 一次推段10-12,但 `advance` 状态机(internal/domain/order/pg_workflow.go:27)要求 `stage == step.stage-1` 严格顺序、非幂等 → worker 激活后订单卡在段10,admin 端 activate 再调报 42200(illegal transition),没有任何 HTTP 接口能把订单从段10 推到 12。激活回调重试接口 `POST /activation-callbacks/:id/retry` 只对已落库的回调日志有效,不推订单 stage。→ 修复方向:AutoPostScan 的 run() 对已 DONE 环节应跳过(幂等续推),或给段11/12 补独立端点。

- **已修复(2026-08-21, 2f0769f)｜信息不准｜admin 端 `POST /orders` 与三端分工口径冲突**:api/openapi/admin/order.yaml:34 描述为"客服代客下单",但项目裁定分工为"下单是 customer 自助(user 端),admin 只做建号/审批/调度/收费"(SKILL.md 基础原则)。admin 端该路由无调用方限定(仅 requirePerm menu:order),联调时极易误用 admin 批量代客下单(本次模拟即踩坑:11 单全用 admin 下,后全部 cancel 重来)。→ 建议 spec 描述补"仅限线下代客极少数场景"或直接下线路由,让 user 端 `/api/user/v1/orders` 成为唯一下单口径。

- **已修复(2026-08-21, 52612c4)｜信息缺失｜`POST /provision/channels` 渠道重复无专用错误码**:channels.code 唯一冲突时返回 50000 内部错误(internal/domain/order/pg_channel.go:44 的 SQLSTATE 23505 未映射),用户只看到"内部错误"无法判断是渠道已存在。→ 应映射为 40900 类业务码并注明"code 重复"。

- **已修复(2026-08-21, 52612c4)｜信息缺失｜渠道目录无查询接口**:channels 只有 POST /provision/channels 创建,没有 GET 列表。下单需要 channelId,联调时只能靠翻 DB 或从已有订单反查。→ 应补 GET /channels(或在下单预取接口里带出)。

- **行为怪象｜订单/工单号日期段按业务时区但列表展示按 UTC**:orderNo 是 ORD-20260822-xxx(业务时区 8/22),而 orders.createdAt 返回 2026-08-21T23:52:48Z(UTC 8/21)。pg.go:117 注释已说明发号按业务时区切日,但前端/admin 列表直接展示 UTC 时间戳,同一天的单出现"8/21 创建却 8/22 单号"的观感错位。→ 展示层应统一转业务时区,或 createdAt 序列化带时区标注。

## 后端·worker

- **信息缺失｜`GET /api/worker/v1/tickets/{ticketNo}` 字段不全**：`internal/httpapi/worker/ticket.go::workerTicketDetailHandler` 仅返回 `ticketNo/bizNo/status/statusLabel/stages/quad/riskCheck`，缺 `type/typeLabel/product/customerName/customerPhoneMasked/address/splitterPort/preBindTag/scheduleSlot/faultTypeLabel/reportedAt/slaLeftMinutes/remoteDiagnosis/finishedAt/distanceKm`（参 `api/openapi/worker/schemas.yaml::TicketDetail`）。移动端工单详情需按 `designs/worker-order-detail-v1.spec.md` §4.2 做 4 屏分支（A 安装/装维中、B 报障/紧急、C 待领取、D 已完成），缺字段前端只能 fallback 占位（`type` 由 `stages.length` 推断 INSTALL=12/REPAIR=6，其余字段缺失则隐藏区块）。

- **已修复(2026-08-21, 36d6d5e)｜行为缺口｜`POST /api/worker/v1/tickets/{ticketNo}/rollback` 仅审计不落库**：`internal/httpapi/worker/ticket_action.go::workerAuditOK` 对 rollback/reschedule 两动作只 `httpx.RecordAudit` + `respond{ok:true}`，**未修改 StageLog、未回退 stage、未动 Order/DispatchTicket 状态**。前端点击"回退上一环节"返回 200 成功 toast，但详情接口再查 stages 数组不变，时间轴不刷新。修复需：(1) 找到该工单 Order 当前 stage；(2) 删除/标废最新一条 StageLog（或新增一条 `result=ROLLED_BACK` 记录并前移 stage 指针）；(3) 同步 `dispatch_tickets.stage` 与 `orders.current_stage`；(4) 重启后端前注意 schema 迁移。前端已临时把 toast 文案改为"回退请求已记录，请下拉刷新查看最新进度"避免误操作预期，等后端补完整功能后再恢复正向文案。

## web/admin

- **已修复(2026-08-21, 2f0769f)｜信息不准｜`web/admin/scripts/dev-token.mjs` 已失效**：脚本按旧前缀 `POST {baseUrl}/auth/login` 请求登录，后端实际前缀是 `/api/admin/v1`（`src/lib/serverConfig.ts` 的 `API_PREFIX`），运行直接 HTTP 404。应改为 `/api/admin/v1/auth/login`，或删除脚本改由 curl + localStorage 注入（skill docs 已记录替代做法）。

## docs/pdfs《Suniway ODN 地理空间编码规范》V1.0（2026-08-20 生效）

- **信息不准｜省级索引与 PSA PSGC 2025-07-31 不一致**：规范列 83 个"省"，实际混入了 3 个高度城市化市（NCR/三宝颜市/伊利甘）且缺南三宝颜省、西三宝颜省两省未单列（PHL056~058 仅 北三宝颜/三宝赞市/三宝赞锡布格）。系统映射（migrations/000075）按 PSGC 事实裁定并在 note 列留痕。
- **信息不准｜城市归属错误 3 处**：塔布克列 PHL016 基里诺（实际 Kalinga 省会）、阿拉贝尔列 PHL072（实际 Sarangani 省会）、纳本图兰列 PHL064（实际 Davao de Oro 省会）。映射按 PSGC 事实归属。
- **信息不准｜Maguindanao 未跟进 2023 拆分**：PHL081"马京达瑙"在 PSA 已拆为北/南马京达瑙两省；规范另有 PHL082 南马京达瑙，PHL081 承继映射北马京达瑙省（note 留痕），建议规范再版时明确。
- **自相矛盾｜城市前缀位数**：正文与 5.4 校验正则均称"3 字母"（`^[A-Z]{3}\d{3}$`），但索引表含 CALM/VALC/TANDA/BAYW 等 23 个 4-5 字母前缀。系统 `odn_city_code.city_prefix` 按事实放宽为 `^[A-Z]{3,5}$`；规范再版需二选一。
- **信息缺失｜局点编码预留位口径**：第 3 章称预留 `MNL150~MNL999`，隐含 MNL002~149 已分配但索引只登记到 *001；城市内局点台账规范未提供完整清单，odn 域局点实体落地时需向规划部索取。
