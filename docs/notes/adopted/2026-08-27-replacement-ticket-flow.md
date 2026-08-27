# 2026-08-27 设备更换单接入执行流（不生成派单工单，asset 域自治状态机）

> 依据: 用户报"新建完成后 RPL-20260827-806379389990 一直是待执行，没有后续按钮"。
> SQL 直查 boss-infra-postgres-1 `boss.replacements` 确认全表仅 1 行(id=1, status=PENDING,
> legal_entity_name 空串)；接口复核 GET /api/admin/v1/replacements 同读路径。
> 全仓 grep 证实: replacements 表只有 List/Create 两个 store 方法、两个 HTTP 端点,
> 无任何状态流转写入方(无 handler/无定时任务/师傅端零关联),PENDING 为吸收态。

## 一、问题定级

契约缺口(设计层面):`replacements` 只实现"建单+看单",状态机在契约与实现两层均缺失。

| 层 | 现状 | 缺口 |
|------|------|------|
| DB | `replacements.status PENDING/DOING/DONE/FAILED`(000007) | 枚举齐但无流转写入方 |
| store | `internal/domain/asset/pg.go` 仅 List/Create | 无 Get/Update/状态流转方法 |
| admin API | `POST /replacements`(建单,默认 PENDING) + `GET /replacements` | 无派单/流转端点;create 未回填法人快照(fields.md §8.1 偏离,实测空串) |
| worker API | 换机页 `/tickets/:ticketNo/replace` 挂在派单工单下 | 换新单无订单→无工单→师傅端不可见 |
| admin UI | `web/admin/src/pages/ams/replace/index.tsx` 列表+新建 | 行上无操作按钮 |

静态原型 `docs/admin/replace.html` 曾画过"派单"按钮(POST action=dispatch),Go 后端从未实现;
terms.md §4 亦从未登记 replacement.status 枚举——换新单是阶段3 建表后一直未接执行流的半成品。

## 二、本次裁定(不可逆)

1. **换新单不生成 dispatch_tickets 行,不改其 order_id 约束**。派单工单与订单 1:1 强绑
   (`order_id BIGINT NOT NULL UNIQUE REFERENCES orders(id)`,000017),师傅端全部资产操作
   (换件/拆机/测速/资源核查)经 ticketOrder() 联订单取 AddressID。换新单是资产域内部
   保障行为,没有订单可挂;放宽约束会波及师傅端核心读路径,收益为零。
2. **replacements 自身升级为 asset 域执行载体**,状态机对齐 terms.md §4 既有
   `task.status` 枚举(PENDING/DOING/DONE/FAILED,零新增枚举值):
   `PENDING --assign(指派师傅)--> DOING --complete(SUCCESS/FAILED)--> DONE / FAILED`。
   终态不可再流转;FAILED 修复重做走"重开新单",不做 FAILED→DOING 回退。
3. **师傅端经独立轻量端点接入**(前缀 /api/worker/v1,沿用 wauth 组):
   - `GET /replacements`:我的换新任务(worker_id=本人 且 status=DOING)
   - `POST /replacements/{id}/complete` {oldEpc,newEpc,result}:完成即落既有
     `worker_replace_logs`(dispatch_ticket_id=0,ticket_no 落 RPL 单号作展示快照),
     同步回写 replacements 终态——师傅端换机页操作习惯不变,新增的只是任务来源。
4. **资产状态联动**(完成时,AppendLifecycle 留轨迹):旧资产 DEPLOYED→MAINTENANCE;
   新资产(newEpc 经 tags.EpcCode 反解)IN_STOCK→DEPLOYED。失败(FAILED)仅旧资产转 MAINTENANCE。
5. **create handler 补齐法人快照**:按 assetId GetAsset 回填 legal_entity_id/legal_entity_name,
   消除 fields.md §8.1 企业锚点铁律的既有偏离。

## 三、放弃了什么(被否决项)

- **放宽 dispatch_tickets.order_id 为可空 + work_type=REPLACE**:工单详情/列表联表 orders
  是师傅端一切资产读路径的寻址根,改为 LEFT JOIN + 全字段 COALESCE 伤核;UNIQUE(order_id)
  约束取消后,取消订单联动、端口释放等"工单必有订单"的语义全部失守。
- **换新单落轻量订单(orders)再挂工单**:订单 12 环节是新装开通语义(terms.md §1,
  禁止增删改序),换新无资源核查/收费/预下发,硬套环节要么造假数据要么改环节序,双输。
- **FAILED→DOING 回退(重做)**:重开新单有新单号可审计,状态回退让 worker_replace_logs
  与终态的多对一关系无法解释。

## 四、契约明细(实现同步项)

- 迁移(预留 **000159**,实现日重查 main 与未合并分支):
  `ALTER TABLE replacements ADD COLUMN worker_id BIGINT REFERENCES workers(id),
   ADD COLUMN worker_name VARCHAR(64) NOT NULL DEFAULT '', ADD COLUMN finished_at TIMESTAMPTZ;`
- terms.md §4 增行:`换新单 replacement.status | PENDING / DOING / DONE / FAILED | 待执行/执行中/完成/失败(复用 task.status 枚举,本 note 裁定)`
- fields.md 增 replacements 三列对齐表(replacementNo/assetId/priority/status/workerId/finishedAt)。
- openapi:admin/asset.yaml 增 `POST /replacements/{id}/assign` {workerId}(PENDING→DOING,
  requirePerm menu:replace);worker/asset.yaml 增上节两端点。
- admin UI:replace 页行操作"派单"(选师傅,assign)+ 状态列已有;师傅端换机页任务列表加来源。
- 失败留痕:assign/complete 落 httpx.RecordAudit;complete 落流水失败必须 slog
  `[replacement] COMPLETE FAILED` 级日志(AGENTS.md 红线),不得静默吞错。

## 五、验收

- 实现分支回归:go test ./internal/domain/asset/... ./internal/httpapi/...;
  新增 store 状态流转 case(非法流转拒止)+ handler 权限/审计 case。
- 102 部署后 E2E:建单(RPL 生成,法人快照非空)→assign→师傅端列表可见→complete→
  DONE + worker_replace_logs 落行 + 资产两态联动;RPL-20260827-806379389990 经
  assign→complete 存量单演示闭环。
- make check 全绿(含 check-contract-sync:迁移号/契约同步项)。

## 六、遗留(本任务不解决)

- PENDING→CANCELED(取消换新)与存量 PENDING 单的批量派单:枚举无 CANCELED,待需求。
- 换新 SLA(参照报障 SLA)与消息中心 todo 联动:待需求。
