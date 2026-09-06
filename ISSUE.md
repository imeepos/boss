# ISSUE.md（上游/工具问题清单）

## CI/deploy-102(2026-09-06 P1 波次发现)

- **已修复(2026-09-06 当日)｜部署静默停摆｜deploy-runner 镜像丢失致全部 run 秒取消**:P1 波次六次 main push 零部署——act_runner 能接单(pickup 日志正常),但 job 容器镜像 192.168.0.102:5000/boss/deploy-runner:latest 已被清(疑似 docker system prune 波及),runner 侧 pull 撞注册表鉴权墙(no basic auth credentials),run 以 cancelled 收场且 runner 日志无错误行(0.2.11 已知缺陷),gitea UI 之外不可见。连锁:镜像没了之后手工重建的 Dockerfile 又缺 docker-compose-linux-x86_64 二进制与 /root/.docker/config.json 注册表凭据(原镜像烤入,重建即失),修镜像分三步才通:①补 compose 二进制 ②烤入 ~/.docker/config.json ③builder prune 清 overlay2 损坏缓存。**根治建议(待办)**:把 deploy-runner 镜像构建固化进 deploy workflow 首步(docker build -f scripts/deploy-runner.Dockerfile 存在性检查+缺失即建,凭据 COPY 进镜像),或改为本地标签 docker://deploy-runner:latest 并有人守护;加密 deploy-run 失败告警(runner pickup 后 N 分钟无镜像 tag 更新即告警)。
- **已固化(2026-09-06,P2-B 会话分支 feat/ci-deploy-guard)｜根治落地｜按上方根治建议落地两件**:①deploy workflow 首步新增 runner-image-guard job(runs-on: windows-latest=daocloud 公网源 node:20-bookworm——102 每周日 04:00 docker-clean.sh 的 image prune -af 会把未被容器引用的 deploy-runner 和基座 bookworm 全清掉,该标签是唯一可匿名重拉、prune 自愈的锚点,与 deploy-runner 双缺场景守护仍可拉起,阻断鸡蛋互锁):经 docker.sock 引擎 API 只读探活本机镜像+注册表 v2 API 探副本(幂等零副作用,双在即 no-op),本机缺失即用 scripts/deploy-runner.Dockerfile 就地重建并推回注册表,本机在而注册表缺只补推——构建上下文固化入仓库 scripts/ops/(docker-compose 二进制 63MB + 内网注册表凭据 deploy-registry-config.json;凭据只含 192.168.0.102:5000,宿主 ~/.docker/config.json 里的 volces 云凭据刻意剔除,网段边界见 deploy-registry-config.README.md);Dockerfile 补 COPY 凭据行,重建镜像开箱可用。②部署成功标记落盘+每日巡检:workflow 尾步 deploy-marker-write.sh 全门禁通过后写宿主 /home/imeepos/boss-deploy-state/last-success.env(ts+sha),scripts/ops/deploy-guard-alert.sh 每日 08:25 cron 比对标记年龄,超 24h 输出 [deploy-guard] ALERT(挂进口径见 docs/ops/patrol-cron.md)。不动 runtime 代码与既有步骤语义;A1 删镜像自愈/A2 自检幂等/A3 告警三态 selftest 验证记录见分支提交。
- **附注｜诊断通道**:run 失败真相在 gitea 库 action_task.log_filename → gitea 容器 /var/lib/gitea/actions_log/<path>(zstd),宿主无 zstd 时借任意带 zstd 的容器(如 postgres:17-alpine)解压;gitea-postgres 与 boss-infra-postgres-1 是两个实例,别连错。

## 前端·web/admin 测试(2026-09-05,T20 月度填报轮发现)

- **环境限制｜bss/user/filter.test.ts「注册时间本地时区格式化」绑定进程时区**:fmtTime(src/lib/format.ts)按业务裁定固定渲染上海墙钟,但断言输入 '2026-08-21T10:00:00' 无时区后缀,按**进程本地时区**解析,期望 '2026-08-21 10:00:00' 仅在进程 TZ=Asia/Shanghai 时成立。本机(TZ=America/Los_Angeles)必挂(received 2026-08-22 01:00:00),102 CI(上海时区)绿。实测 TZ=Asia/Shanghai pnpm vitest run src/pages/bss/user/filter.test.ts 5/5 过。→ 建议:输入显式带后缀 '2026-08-21T10:00:00+08:00' 并断言上海墙钟,消除对进程 TZ 的依赖。非 T20 引入(T20 全量 413 用例在上海时区下全绿)。

## 后端·TL1 会话层(2026-09-02 T3 tl1sim 联测发现)

- **已修复(2026-09-04, dfdba439)｜行为限制｜session.login 不消化 DELAY**:login 已改为与 `Session.Do` 同款 DELAY 追帧循环(收到 DELAY 继续等同 ctag 最终帧,异 ctag 残帧丢弃,总时长仍受 CmdTimeout 与 ctx 截止约束),真实 U2000 对 LOGIN 回 DELAY 不再误报 ErrAuth;cmd/tl1sim/sim 对 LOGIN 的 delay 注入豁免同步解除,测试路径与真实路径一致。回归三例 internal/domain/provision/tl1/session_login_test.go;机械自测 scripts/ops/tl1-login-delay-selftest.sh(单元证据+静态断言+build/vet,FAIL 即 exit 1)。

## 后端·order 12 环节全流程模拟(2026-08-22 bossctl CLI 演示发现)

- **已修复(2026-08-21, d397e40)｜行为缺口｜置备资产不回填 tag 双向绑定**:`POST /provision/assets`(internal/domain/asset/pg_write.go::CreateAsset)只写 `assets.tag_id`,不回填 `tags.bound_asset_id`/`tags.status`。而环节9 扫码核对(internal/domain/quadlink/pg_scan.go::VerifyScan)要求 `tags.bound_asset_id` 非空且等于 quad_link.asset_id,否则报 40920"扫码与预绑定不一致"。e2e 测试(internal/app/e2e_pg_integration_test.go:224)是建 tag 时手工传 `BoundAssetID + Status="BOUND"` 才绕过。→ 置备端点或 CreateAsset 应在 tag_id 非空时同步 `UPDATE tags SET bound_asset_id=$asset, status='BOUND'`,与 e2e 口径对齐。

- **已修复(2026-08-27, 744abd23/6793bdca/57375a30)｜行为缺口(d397e40 修复不完整)｜资产↔标签双绑缺口 + DB 兜底**:d397e40 修复了 CreateAsset 单向回填,但用 `WHERE bound_asset_id IS NULL` 哑条件 → 业务流(标签侧先建并填 bound)静默跳过 → 资产变孤儿;CreateTag 完全无反向回填 → 124 条 B 端历史孤儿由此产生。修复:CreateAsset 回填改用 `IS NULL OR = $expected`(0 行必为冲突)、CreateTag 增加反向回填 + ErrBindingConflict 哨兵 → 40900 + reason 透传、DB 部分唯一约束 000158 兜底 + 23505 拆分映射 ErrBindingConflict、scripts/verify-asset-tag-binding.sh 三场景真接口验证、internal/domain/report/pg_patrol.go 增加双向巡检作为每日防线。详见 adopted note 2026-08-27-asset-tag-bidirectional-binding.md。

- **已修复(2026-08-21, 756d21c)｜行为缺口｜自动化 applyTag 的四码不带资产**:`internal/domain/order/pg_workflow.go:125` 构造 `QuadLinkBindReq` 时 AssetID 恒为 0(自动化链路没有"选资产"步骤),而 VerifyScan 要求 `tags.bound_asset_id == link.asset_id` → **走自动化链路(POST /orders/:no/charge)的订单,扫码环节永远 MISMATCH**。只有 e2e 手工预绑定(CreateLink 带 AssetID)能通。→ 要么 applyTag 自动挑一个 IN_STOCK 未绑资产入库四码,要么 VerifyScan 对 asset_id=0 的 link 做兜底(扫码时回填而非比对失败)。这是自动化链路 vs 手工链路的真实分歧,阻塞所有真实订单走完 12 环节。

- **已修复(2026-08-21, a6c5d98)｜行为缺口｜worker/admin activate 不对称,段11/12 无独立推进入口**:worker 端 `POST /tickets/:no/activate`(internal/httpapi/worker/scan.go:224)只调 `Order.ActivateUser` 推段10;admin 端 `POST /tickets/:no/activate` 调 `Automation.AutoPostScan` 一次推段10-12,但 `advance` 状态机(internal/domain/order/pg_workflow.go:27)要求 `stage == step.stage-1` 严格顺序、非幂等 → worker 激活后订单卡在段10,admin 端 activate 再调报 42200(illegal transition),没有任何 HTTP 接口能把订单从段10 推到 12。激活回调重试接口 `POST /activation-callbacks/:id/retry` 只对已落库的回调日志有效,不推订单 stage。→ 修复方向:AutoPostScan 的 run() 对已 DONE 环节应跳过(幂等续推),或给段11/12 补独立端点。

- **已修复(2026-08-21, 2f0769f)｜信息不准｜admin 端 `POST /orders` 与三端分工口径冲突**:api/openapi/admin/order.yaml:34 描述为"客服代客下单",但项目裁定分工为"下单是 customer 自助(user 端),admin 只做建号/审批/调度/收费"(SKILL.md 基础原则)。admin 端该路由无调用方限定(仅 requirePerm menu:order),联调时极易误用 admin 批量代客下单(本次模拟即踩坑:11 单全用 admin 下,后全部 cancel 重来)。→ 建议 spec 描述补"仅限线下代客极少数场景"或直接下线路由,让 user 端 `/api/user/v1/orders` 成为唯一下单口径。

- **已修复(2026-08-21, 52612c4)｜信息缺失｜`POST /provision/channels` 渠道重复无专用错误码**:channels.code 唯一冲突时返回 50000 内部错误(internal/domain/order/pg_channel.go:44 的 SQLSTATE 23505 未映射),用户只看到"内部错误"无法判断是渠道已存在。→ 应映射为 40900 类业务码并注明"code 重复"。

- **已修复(2026-08-21, 52612c4)｜信息缺失｜渠道目录无查询接口**:channels 只有 POST /provision/channels 创建,没有 GET 列表。下单需要 channelId,联调时只能靠翻 DB 或从已有订单反查。→ 应补 GET /channels(或在下单预取接口里带出)。

- **已修复(2026-08-21, c364f4a)｜行为怪象｜订单/工单号日期段按业务时区但列表展示按 UTC**:orderNo 是 ORD-20260822-xxx(业务时区 8/22),而 orders.createdAt 返回 2026-08-21T23:52:48Z(UTC 8/21)。pg.go:117 注释已说明发号按业务时区切日,但前端/admin 列表直接展示 UTC 时间戳,同一天的单出现"8/21 创建却 8/22 单号"的观感错位。→ 展示层应统一转业务时区,或 createdAt 序列化带时区标注。

## 第4轮全流程重验(2026-08-22,零 SQL 补救贯通 12 环节)

- **已修复(2026-08-22, 61b0dcd/000097)｜行为缺口｜000056 残留 `*_active` 索引阻塞复购客户**:000088 裁定 customer 可 1:N(一客户多链路)并撤销 000086 的 `uq_quad_links_customer`,但 000056 时代的四个 `uq_quad_links_{customer,asset,port,address}_active` 部分唯一索引从未被任何迁移删除 → 客户已有 LINKED 链路后,第二单扫码置 LINKED 必撞 23505(实测 ORD-20260822-000432 scan-bind 报 duplicate key uq_quad_links_customer_active)。修复:迁移 000097 DROP 四个残留索引(customer 列违反 1:N 契约;asset/port/address 三列被 000086 非空唯一完全覆盖属纯冗余),e2e 新增 W8c 回归(同客户两地址两单,双双扫码 LINKED)。
- **已修复(2026-08-24, 9f76b26)｜信息不准｜`GET /provision/channels` 响应字段 PascalCase**:52612c4 新增的渠道目录返回 `ID/Code/Name/Status`(domain struct 无 json tag 直出),违反 fields.md §0 的 JSON lowerCamelCase 规则(应为 `id/code/name/status`)。修复:Channel struct 加 json tag + OpenAPI 响应 schema 同步,102 实测返回 `{"id":102,"code":"HALL",...}`。

## 后端·worker

- **已修复(2026-08-21, 94b6078)｜信息缺失｜`GET /api/worker/v1/tickets/{ticketNo}` 字段不全**：`internal/httpapi/worker/ticket.go::workerTicketDetailHandler` 仅返回 `ticketNo/bizNo/status/statusLabel/stages/quad/riskCheck`，缺 `type/typeLabel/product/customerName/customerPhoneMasked/address/splitterPort/preBindTag/scheduleSlot/faultTypeLabel/reportedAt/slaLeftMinutes/remoteDiagnosis/finishedAt/distanceKm`（参 `api/openapi/worker/schemas.yaml::TicketDetail`）。移动端工单详情需按 `designs/worker-order-detail-v1.spec.md` §4.2 做 4 屏分支（A 安装/装维中、B 报障/紧急、C 待领取、D 已完成），缺字段前端只能 fallback 占位（`type` 由 `stages.length` 推断 INSTALL=12/REPAIR=6，其余字段缺失则隐藏区块）。

- **已修复(2026-08-21, 36d6d5e)｜行为缺口｜`POST /api/worker/v1/tickets/{ticketNo}/rollback` 仅审计不落库**：`internal/httpapi/worker/ticket_action.go::workerAuditOK` 对 rollback/reschedule 两动作只 `httpx.RecordAudit` + `respond{ok:true}`，**未修改 StageLog、未回退 stage、未动 Order/DispatchTicket 状态**。前端点击"回退上一环节"返回 200 成功 toast，但详情接口再查 stages 数组不变，时间轴不刷新。修复需：(1) 找到该工单 Order 当前 stage；(2) 删除/标废最新一条 StageLog（或新增一条 `result=ROLLED_BACK` 记录并前移 stage 指针）；(3) 同步 `dispatch_tickets.stage` 与 `orders.current_stage`；(4) 重启后端前注意 schema 迁移。前端已临时把 toast 文案改为"回退请求已记录，请下拉刷新查看最新进度"避免误操作预期，等后端补完整功能后再恢复正向文案。

## CI/部署(deploy-102)

- **已定位+绕法(2026-08-25)｜工具 bug｜act_runner 0.2.11 丢失 job 中间步骤日志**:actions_log 的 zst 只含 Clone 首尾行与最终结论,中间步骤输出全部丢失(成功/失败 run 同样),失败无法从 run 日志定位。**绕法**:runner config level 调 debug + `docker logs gitea-runner`,能看到每步的步骤名与 exitcode(本次定位 SIGPIPE 的关键);排查完调回 info。
- **已修复(2026-08-25, c2a2df61)｜部署怪象｜deploy-102 连续 28 个 run 5 秒内死于 Clone 后(run 1716-1743)**:根因=Classify 步骤 `deployed=$(docker images | grep -Ev ... | head -1)` 在 pipefail 下的 **SIGPIPE 竞态**——102 本地 boss/server sha tag 随部署累积增多后,head -1 提前关管道使 grep 收 141(SIGPIPE),pipefail 放大为管道失败 → set -e 杀步骤。代码/workflow/凭据全没变却从 16:53 起必现,重启 runner 无效(竞态在脚本层)。修复:管道尾 `|| true` 兜底;task 2862 debug 日志实证 exitcode 141,run 1745 起全绿。2026-09-22 会话记录的"runner 任务状态机卡死"同症状,实为此因。

- **已修复(2026-08-24, 4e68347)｜部署怪象｜`docker-compose up -d --force-recreate` 后容器滞留 Created 不启动**:deploy-102 workflow 的 Deploy 步骤执行后,boss-server/boss-report/boss-admin-web 常处于 "Created" 状态而非 Up,需人工 `docker start`。根因(任务 2289/2291 日志):compose 固定 `container_name` 被其他项目(手工 deployments 部署/无 label docker run)的同名容器占用,`--force-recreate` 在 `Conflict. The container name "/boss-admin-web" is already in use` 处中止,已 Recreate 的容器滞留 Created。修复:Deploy 步骤先 `docker rm -f boss-server boss-aaa boss-report boss-admin-web` 清残留,up 后逐容器断言 running(新增 Verify all containers running 步骤);任务 2579 起全绿。
- **已修复(2026-08-24, eee7fd9)｜信息缺失｜compose 未设 `BOSS_CORS_ORIGINS`,5180 直连 28080 必挂**:后端 CORS 白名单默认仅 localhost:5173/5174(internal/pkg/config/config.go:130),102 上 admin-web(5180)若在"服务端配置"里填 `http://192.168.0.102:28080` 直连,预检 OPTIONS 404 全端不可用。修复:compose server environment 显式加 `BOSS_CORS_ORIGINS: "http://192.168.0.102:5180,http://localhost:5173,http://localhost:5174"`(中间件 477ec0b 起任意 Origin 回显放行,此值作显式配置与收紧护栏);102 实测 OPTIONS 预检 204 + `Access-Control-Allow-Origin: http://192.168.0.102:5180`。

## web/admin

- **已修复(2026-08-21, 2f0769f)｜信息不准｜`web/admin/scripts/dev-token.mjs` 已失效**：脚本按旧前缀 `POST {baseUrl}/auth/login` 请求登录，后端实际前缀是 `/api/admin/v1`（`src/lib/serverConfig.ts` 的 `API_PREFIX`），运行直接 HTTP 404。应改为 `/api/admin/v1/auth/login`，或删除脚本改由 curl + localStorage 注入（skill docs 已记录替代做法）。

## docs/pdfs《Suniway ODN 地理空间编码规范》V1.0（2026-08-20 生效）

- **信息不准｜省级索引与 PSA PSGC 2025-07-31 不一致**：规范列 83 个"省"，实际混入了 3 个高度城市化市（NCR/三宝颜市/伊利甘）且缺南三宝颜省、西三宝颜省两省未单列（PHL056~058 仅 北三宝颜/三宝赞市/三宝赞锡布格）。系统映射（migrations/000075）按 PSGC 事实裁定并在 note 列留痕。
- **信息不准｜城市归属错误 3 处**：塔布克列 PHL016 基里诺（实际 Kalinga 省会）、阿拉贝尔列 PHL072（实际 Sarangani 省会）、纳本图兰列 PHL064（实际 Davao de Oro 省会）。映射按 PSGC 事实归属。
- **信息不准｜Maguindanao 未跟进 2023 拆分**：PHL081"马京达瑙"在 PSA 已拆为北/南马京达瑙两省；规范另有 PHL082 南马京达瑙，PHL081 承继映射北马京达瑙省（note 留痕），建议规范再版时明确。
- **自相矛盾｜城市前缀位数**：正文与 5.4 校验正则均称"3 字母"（`^[A-Z]{3}\d{3}$`），但索引表含 CALM/VALC/TANDA/BAYW 等 23 个 4-5 字母前缀。系统 `odn_city_code.city_prefix` 按事实放宽为 `^[A-Z]{3,5}$`；规范再版需二选一。
- **信息缺失｜局点编码预留位口径**：第 3 章称预留 `MNL150~MNL999`，隐含 MNL002~149 已分配但索引只登记到 *001；城市内局点台账规范未提供完整清单，odn 域局点实体落地时需向规划部索取。

## CI runner 卡死复发记录(2026-08-25 晚)
- 症状与 2026-09-22 会话记录完全一致:deploy-102 run 1716-1741 连续 5 秒内死于 Clone 后,日志只剩首尾行,docs-only run 同样失败。
- 本次新增动作:重启 gitea-runner 容器(上次未尝试);重启后本条 push 触发的 run 结果见 gitea UI。
- 服务恢复方式同上次:ssh 102 手动复刻 CI 步骤(clone/build/push/compose up),boss-server 已更新至 c683a55b。

## docs/user 静态演示门户与真实 user API 信封不匹配 + locale.js 污染(2026-08-26 支付验收发现)

- **信息不准｜demo 页消费平铺响应**:docs/user 全部 13 个页面写 `d.items`(如 bills.html:49 / pay.html:47-58),
  真实 user API 返回 `{code,data:{items},msg}` 信封 → 浏览器列表全空。修法:消费处改 `(d.data||d).items`。
- **信息缺失｜locale.js 模板污染**:docs/user/locale.js 约 30 行被 JS 模板片段污染(如 `'user.profile.text11': '在用'' : ''tag-gray">未用'') + ''',`
  ×3 语区,js 解析直接挂(locale.js 加载后 window.L 为 undefined)。需按语区重建被污染键的值。
- 影响:web 演示门户不可用;不影响 Android 用户端(按信封消费)。属 domain-map「PORT 待完善」既有项,非支付链路缺陷。

## user Android 两周上线-后端依赖缺口(2026-08-27 上线计划 D1-D4 发现)

- **已修复(2026-08-27, c8c5bb7a/6340b413)｜信息缺失｜积分兑换无可兑换券模板列表端点**:api/openapi/user/loy.yaml 的 `/points/exchange` 契约要求 `templateId`,
   但 user 侧没有任何端点能列出可兑换券模板(`coupon_templates.points_price>0`,fields.md §8D-2)。
   → 已实现 `GET /points/exchange-offers`(契约+promotion ListExchangeOffers(ENABLED 且 points_price>0)+user handler);
   Android PointsPage 兑换区真实化(模板列表+确认兑换+终态文案)。真实环境验证: 带 token 返回真实模板,
   Android 真实点兑 300→100 积分、新券 ISSUED 落库、流水 EXCHANGE -200。(原"建设中占位"已移除)
- **已修复(2026-08-27, 实测 POST 401=路由在)｜信息缺失｜`/push/device` 契约-部署漂移**:api/openapi/user/misc.yaml 已定义,
  曾用 **GET** 探测误报 404;实为 **POST** 端点且后端已落地(GET 当然 404)。修正:POST /api/user/v1/push/device → 401(需鉴权),
  handler `internal/httpapi/user/push_device.go` + 路由已注册(auth.go biz 组),push_devices 表 000095 PG 持久化,
  RegisterDevice 幂等(同 registrationId 换绑主体)。→ Android B 轨接线完成(登录/启动上报 deviceId,见 feat(user-android): push-device 注册)。

## 工具·DSH 宿主(2026-08-29,后台代客闭环轮发现)

- **未修复｜工具限制｜read_image 被运行时模型能力声明拒绝,与实际模型能力不符**:本会话模型
  GLM-5.3-Flash 实际可理解图像,但 dsh 运行时的模型能力声明(role manifest / model metadata)
  未包含图像输入标志,`read_image` 一律报
  `model "GLM-5.3-Flash" does not declare image input; switch to an image-capable model`
  (2026-08-29 会话两次重试一致复现;PNG 文件本身有效,`file` 确认 1600x900 RGB 非损坏)。
  影响:浏览器截图(cdp-capture 产物)无法直接目检,只能靠 --eval DOM 断言或转发给图像模型会话。
  → 修法:dsh 侧把 GLM-5.3-Flash 的输入能力声明补上 image,或 read_image 的能力校验放宽为
  "尝试投递、上游拒绝再报错"。在修好前,涉及截图审阅的任务请改用 DOM 断言路径
  (见 .agents/skills/self-evolving 高频红线 7 的替代用法)。

## 工具·self-evolving 脚本(2026-08-29,内联建址轮发现)

- **已修复(2026-09-07)｜脚本 bug｜cdp-admin-capture.mjs parseArgs 只透传第一个 --eval**:eval 分支内
  `i += 1` 后 `continue`,而 for 循环 update 又执行 `i += 2`,实际步进 3 格,把第 2 个及以后的
  `--eval` 值误解析为垃圾键(如 `args['E3']='--settle'`),后续 eval 全部静默丢失。
  表象:多个 --eval 只有第 1 个业务 eval 执行,其余无输出无报错(2026-08-29 两次复现)。
  → 修复:eval 分支去掉分支内 `i += 1`,只靠 continue 走 update 步进;内联样例验证双 eval
  均入列。另录环境限制:DSH 沙箱内 Chrome CDP 的 WebSocket 悬挂不开、--remote-debugging-pipe
  模式 Chrome SIGTRAP 自毁——浏览器级交互测试在本沙箱不可行,须在沙箱外跑 cdp 工具族。
- **未修复｜worktree 陷阱｜git worktree add 后 checkout 可能未落地即返回 success**:
  `git worktree add <dir> -b <branch>` 打印 "HEAD is now at <sha>" 但目标目录内无 .git 指针、
  无任何仓库文件(worktree list 却显示已注册);其后向该目录写文件全部落在 git 管辖外。
  2026-08-29 复现一次,prune/remove 需用相对名 `git worktree remove --force wt-admin-address-chain`
  (绝对路径报 not a working tree),随后删目录重建才正常 checkout。
  → 修法(流程非工具):worktree add 后必须 `ls <dir>/.git` 确认指针存在再写任何文件。

## 工具·deploy-102 流水线(2026-08-29,内联建址联调轮发现)

- **部分修复(2026-09-07)｜编排缺陷｜chore(deploy) 空提交重触发手法对 Classify 失效**:deploy-102.yml 的
  Classify 按触发提交 diff 判 runtime,空提交 diff 为空 → files 空 → runtime=false → 部署被
  skip(run 空转但显示完成)。陈默 ac1902d7 重触发即踩此坑(run 起了但没部署),林晓实测窗口内
  102 仍是旧 server(POST /orders/address 40400)+旧前端(Cb7RTl8M),三路验收被误判 Gate0。
  同窗叠加因素:concurrency cancel-in-progress 会吞掉排队中的旧 run;admin-web 全量构建
  ~28min,部署窗口内 index.html 可能长时间指向旧 bundle。
  → 本次实际生效部署=1afb5afe(其 Classify 恰取到旧 deployed tag,diff 含 web/admin → true)。
  修法进展(2026-09-07):②已落地——deploy-102.yml 新增「Verify deployment fingerprint」步,
  跑 scripts/ops/verify-deploy.sh --expect-sha $GITHUB_SHA,服务端 healthz commit(ldflags
  GIT_SHA 注入)+bundle 指纹任一不过即红,复验口诀流水线化;③已存在——admin-web.nginx.conf
  对 index.html 与 SPA fallback 均 no-cache(复查确认);①部分覆盖——Classify 已有「与 102
  实际部署镜像 sha 比对」优先策略,先前未部署的 runtime 变更不会被后续 docs-only 提交漏判,
  「diff 为空显式输出 no-op 原因」仍待 owner(现为静默 skip 文案,无强制 runtime)。
  复验口诀(部署后):curl :5180 取 index-*.js 文件名 + grep 特征串,双端各一个;**特征串 grep 必须扫懒加载分片**(2026-09-01 实证:index-*.js 是壳,页面代码在 assets/<Page>-*.js,只 grep index 假 0 命中误判脑裂;先拉分片清单再逐片 grep);
  或直接 `bash scripts/ops/verify-deploy.sh --expect-sha <sha>`。

## 工具/环境(2026-08-30 session-handoff DSH 插件开发轮发现)

- **未修复｜工具 bug｜dsh-plugin-dev check.sh 在 macOS 上 sed 报错**：`scripts/check.sh` 守卫 E 段第 88 行 `tr \' \n\' | sed \'/^$/d\'` 引号嵌套在 bash/macOS BSD sed 下炸出 `sed: 1: "'/^$/d'": invalid command code '`，只是噪音（守卫结论仍正确），但每次交卷都刷屏。→ 改为 `grep -v '^$'` 或独立管道段。
- **未修复｜环境｜npm 默认缓存目录 root 属主**：`/Users/imeepos/ext512/dev-cache/npm` 内有 root 属主文件，任何 npm install/pack 直接 EPERM。→ 要么 `sudo chown -R 501:20` 修属主，要么本轮做法：npm 命令一律加 `--cache /tmp/npm-cache-<场景>`。

## 下发链路验证轮(2026-09-01 offer-provision-binding/POQ 会话发现)

- **未修复｜行为缺口(geo-unify/多区域域内)｜mainchain-acceptance.sh 与派单区域强匹配不兼容**:
  geo-unify/多区域合流后,`POST /dispatch/pool/:ticketNo/assign` 强制"工单区域=师傅区域"
  (40900 + forceRequired:true),但验收车 `scripts/ops/mainchain-acceptance.sh` 造的验收地址
  不带 region → 工单区域解析为根区域"集团"(region 1),与 MASTER(如 6 号王测试=区域4 马尼拉,
  workerRegionIds=[4])必然 mismatch → 验收车在 assign 步骤 FAIL(2026-09-01 03:44 实测
  DT-20260901-000598)。修法二选一,由 geo-unify/worker 会话裁决:
  ①验收车 POST /addresses 带上与 MASTER 同域的 regionId/region_path(顺带真实化区域派单覆盖);
  ②或验收脚本走 force 指派通道(如存在)。另:派单建单 LATERAL 的 ltree=varchar 42883 已由
  fix fd235d4e 修复上线(path::text 显式转型),本条只余区域匹配语义部分。
- **已修复(2026-09-01, fd235d4e)｜行为怪象｜createTicketOnDispatch LATERAL ltree 与 varchar 裸比较**:
  regions.path(ltree) 与 orders.region_path(varchar) 直接等值/LIKE,区域非空即报
  `operator does not exist: ltree = character varying (42883)`,环节8 派单建单失败 →
  AutoPreScan 50000、主链全断。geo-unify/多区域合流后新单必现(orders.region_path 开始
  非空填充)。修复:`path::text = o.region_path OR o.region_path LIKE path::text || '.%'`。
  注:pgxmock 单测无法拦截此类 SQL 类型错误,真实库回归(mainchain)才是防线。

## 2026-09-02 provision_tasks 孤儿任务堆积(巡检未覆盖,复发)
- `provision_tasks` 对 `orders` 无外键,订单删除后任务残留;任务又把 `provision_templates` 钉死(DELETE 守卫查"任何任务引用",含 DONE),垃圾模板永远删不掉。2026-08-30 note 已记"验收清理漏删 14 条孤儿,巡检脚本覆盖待后续",2026-09-02 复发积到 158 条(158 任务/275 日志已手工清理)。
- 建议:订单删除路径级联删任务+日志,或验收巡检 SQL 加"孤儿 provision_tasks/provision_logs 计数"门禁(docs/ops/patrol-cron.md)。
