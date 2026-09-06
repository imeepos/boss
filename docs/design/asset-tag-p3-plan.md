# 资产标签 P3 收尾波计划（2026-09-06）

前情: P0 告警实战演练已由 Lead 于 102 完成并 PASS（OK 态 exit 0 / ALERT 态 exit 1 / docker-clean 保留过滤器在位）。asset_assignments 经查证已由 P2-W2 激活（11 行真实数据 + 领用/归还全链路），死表销项。

## 一、研究结论（社区最佳实践 → 本波裁定）

1. 分页选型: 管理后台需总数与跳页，数据万级以内 offset/limit 足够；keyset 适合深翻页流水型 feed，本场景属过度设计。TMF639 社区惯例即 limit/offset 查询参数 + 响应带总量。出处: TMF639 Filtering/Limiting/paginating 社区帖、API Pagination: Cursor vs Offset in 2026 (apiscout.dev)。
2. 排序与注入防护: 排序字段必须服务端白名单；ORDER BY 必须带 id tie-breaker 保证翻页稳定（不重不漏）。出处: apiscout 分页指南。
3. 身份列唯一性: 电信 ONU 身份（SN/MAC/LOID）是网络鉴权标识，全网必须唯一；Snipe-IT 弱去重曾致重复序列号竞态（issue #14476），社区补救方向即唯一性校验加固（PR #18866）。本波直接 DB 部分唯一索引硬保证（空值不占唯一名额）。
4. EPC Gen2 校验: 96-bit Gen2 标签标准十六进制表示为 24 位 hex；常见结构 SGTIN-96(0x30)/GRAI-96(0x32)/GIAI-96(0x33)/GID-96(0x35)。tags.epc_code 列已存在，本波在写路径校验（存量不回填不强制）。
5. 破坏性操作确认: GitHub/AWS 惯例为「输入资源名」确认；电信实物资产有实体铭牌，升级为三要素核对（资产编码必填 + SN/标签号按实物有无动态必填），后端同步强校验防绕过前端。

## 二、任务清单（单一会话粒度，机械验收）

| # | 任务 | 会话 | 迁移号 | 验收命令（main 部署后 102 执行，exit 0） |
|---|---|---|---|---|
| T1 | 资产/标签列表服务端分页 | P3-D | 无 | bash scripts/e2e/verify-asset-pagination-e2e.sh |
| T2 | 身份列 sn/mac/loid + EPC 校验 | P3-E | 000188 | bash scripts/e2e/verify-asset-identity-epc-e2e.sh |
| T4 | 历史 BIND/CREATE 事件回填 | P3-E | 000189 | bash scripts/e2e/verify-tag-event-backfill.sh |
| T3 | 报废三要素确认 | P3-F | 无 | bash scripts/e2e/verify-asset-scrap-confirm-e2e.sh |

全部任务另需: make check 全绿；e2e 脚本幂等（重跑 PASS）；BASE_URL 缺省 http://192.168.0.102:28080。

## 三、需求与边界

### T1 分页（P3-D）
- GET /api/admin/v1/assets 与 /tags 支持 offset/limit（默认 0/50，limit 上限 200 超限钳制不报错）与可选过滤 status/type/model_id/q（asset_code 前缀模糊）；响应包 items + total。
- 排序白名单: created_at/asset_code/tag_no/status（默认 created_at DESC, id DESC tie-breaker）；非法排序值 400。
- 前端资产/标签两列表改服务端分页（复用 Pagination 组件），页码/每页条数变更触发请求；筛选条件作为请求参数下发。
- 兼容: 不传分页参数时按默认分页返回（不保留全量模式，调用方仅前端一处）。
- i18n/menu.def 等中央登记文件改动压独立小提交。

### T2 身份列 + EPC（P3-E，迁移 000188）
- assets 加 sn/mac/loid 三可空文本列；部分唯一索引各一（WHERE 列非空非空串），重复写入 DB 层拒绝并映射 409 语义。
- 创建/编辑 API 接收三字段，格式校验: MAC 六组 hex（:或-分隔均可，入库原样）、LOID/SN 非空去首尾空格；空串一律转 NULL 存储。
- tags.epc_code 写路径（创建/绑定改绑）校验: 24 位 hex（大小写不敏感，入库统一大写）；头部字节若非 0x30/0x32/0x33/0x35 拒绝。
- 存量数据不回填不强制；表单增加三字段（选填，标注 SN/MAC/LOID 用途）。
### T4 事件回填（P3-E，迁移 000189）
- 为零事件资产各补一条 CREATE 事件、为零 BIND 事件的在绑标签各补一条 BIND 事件；changed 记回填来源标记；幂等（重跑行数不变）；down 可清仅回填行。
### T3 报废三要素（P3-F）
- POST /assets/:id/scrap 请求体加 confirmAssetCode/confirmSn/confirmTagNo: 与服务端现值不一致（或该填不填）返回 422；资产无 SN 时 confirmSn 必须为空串，未绑标签时 confirmTagNo 必须为空串。
- 前端 ScrapDialog: 展示待报废资产三要素参考值不直接可复制（核对实物铭牌场景），输入框动态必填；保留 reason；提交前本地预校验。
- 审计事件沿用现有 RecordAudit，附 confirmSn 尾四位即可（不落全量 SN）。

## 四、协作与合并协议（Lead 统一执行）

1. 三会话在各自 worktree（feat/p3-d / feat/p3-e / feat/p3-f）开发；只回报不合并不部署。
2. 迁移号独占: 仅 P3-E 可用 000188/000189（已核对全部分支无占号）；其余会话禁止新增迁移。
3. 合并序: 完成即报，Lead 逐个反向同步 main + make check + ff-only 合并；一个合完再下一个；全合并后 CI 部署 102，Lead 跑全部验收命令。
4. 验收全绿后: Lead 归档三会话 + 清理 worktree/分支（本地与 gitea）。

## 五、销项与遗留

- 销项: 告警演练（本轮已做）；asset_assignments 死表（已激活）；SN/MAC/LOID 身份列、EPC 校验、分页、报废三要素、事件回填（本波）。
- 遗留(不在本波): keyset 分页（数据量到十万级再议）；MAC 规范化存储与模糊匹配；光猫/ONU 字典归一（业务裁定待用户）；服务端分页推广到其他域列表（后续波次）。