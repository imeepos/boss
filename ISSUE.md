# ISSUE.md（上游/工具问题清单）

## 后端·worker

- **信息缺失｜`GET /api/worker/v1/tickets/{ticketNo}` 字段不全**：`internal/httpapi/worker/ticket.go::workerTicketDetailHandler` 仅返回 `ticketNo/bizNo/status/statusLabel/stages/quad/riskCheck`，缺 `type/typeLabel/product/customerName/customerPhoneMasked/address/splitterPort/preBindTag/scheduleSlot/faultTypeLabel/reportedAt/slaLeftMinutes/remoteDiagnosis/finishedAt/distanceKm`（参 `api/openapi/worker/schemas.yaml::TicketDetail`）。移动端工单详情需按 `designs/worker-order-detail-v1.spec.md` §4.2 做 4 屏分支（A 安装/装维中、B 报障/紧急、C 待领取、D 已完成），缺字段前端只能 fallback 占位（`type` 由 `stages.length` 推断 INSTALL=12/REPAIR=6，其余字段缺失则隐藏区块）。

## web/admin

- **信息不准｜`web/admin/scripts/dev-token.mjs` 已失效**：脚本按旧前缀 `POST {baseUrl}/auth/login` 请求登录，后端实际前缀是 `/api/admin/v1`（`src/lib/serverConfig.ts` 的 `API_PREFIX`），运行直接 HTTP 404。应改为 `/api/admin/v1/auth/login`，或删除脚本改由 curl + localStorage 注入（skill docs 已记录替代做法）。

## docs/pdfs《Suniway ODN 地理空间编码规范》V1.0（2026-08-20 生效）

- **信息不准｜省级索引与 PSA PSGC 2025-07-31 不一致**：规范列 83 个"省"，实际混入了 3 个高度城市化市（NCR/三宝颜市/伊利甘）且缺南三宝颜省、西三宝颜省两省未单列（PHL056~058 仅 北三宝颜/三宝赞市/三宝赞锡布格）。系统映射（migrations/000075）按 PSGC 事实裁定并在 note 列留痕。
- **信息不准｜城市归属错误 3 处**：塔布克列 PHL016 基里诺（实际 Kalinga 省会）、阿拉贝尔列 PHL072（实际 Sarangani 省会）、纳本图兰列 PHL064（实际 Davao de Oro 省会）。映射按 PSGC 事实归属。
- **信息不准｜Maguindanao 未跟进 2023 拆分**：PHL081"马京达瑙"在 PSA 已拆为北/南马京达瑙两省；规范另有 PHL082 南马京达瑙，PHL081 承继映射北马京达瑙省（note 留痕），建议规范再版时明确。
- **自相矛盾｜城市前缀位数**：正文与 5.4 校验正则均称"3 字母"（`^[A-Z]{3}\d{3}$`），但索引表含 CALM/VALC/TANDA/BAYW 等 23 个 4-5 字母前缀。系统 `odn_city_code.city_prefix` 按事实放宽为 `^[A-Z]{3,5}$`；规范再版需二选一。
- **信息缺失｜局点编码预留位口径**：第 3 章称预留 `MNL150~MNL999`，隐含 MNL002~149 已分配但索引只登记到 *001；城市内局点台账规范未提供完整清单，odn 域局点实体落地时需向规划部索取。
