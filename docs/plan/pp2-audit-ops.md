# PP2-W3 计费+运维域 30 页体验走查 · Phase A 审计(2026-09-07)

> 会话 W3 | 分支 feat/pp2-w3-ops | 范围:billing 全域 7 + oss 全域 8 + provision 3 + quad 3 + alarm 1 + aaa 3 + intel 残留 4,共 29 个路由组(33 个源文件;「30 页」按任务书口径含 billing 页内 invoices 子页签)。
> 方法:源码静态走查(逐文件实读,file:line 引用)+ 后端路由核对(internal/httpapi/admin)+ 102 真实走查(cdp-admin-capture.mjs,base http://192.168.0.102:5180,theme=light lang=zh-CN,29 路由全实截 /tmp/pp2-w3/,不入库)。
> 契约依据:terms.md(状态枚举)/fields.md(§3.3-3.5、§4.2、§4.4、§5.1、§1.5.3-1.5.10、§8I/8J)/domain-map.md;路线图第三节为统一验收规则。

## 一、102 实截与 DOM 断言结论(2026-09-07)

- 29/29 路由实截成功,截图 76KB-205KB(非空白);DOM 断言:location.pathname 正确、页头标题正确、表格/卡片存在,详见 /tmp/pp2-w3/logs/*.stdout 的 eval 行。
- console 报错:29 页全部 0;失败请求:28 页 0;唯一异常 intel/gis 外部高德瓦片 10 条 no-status(代理拦截,非业务接口,登记环境因素)。
- 走查基线=102 部署前端现版本;本审计的阶段 B 修复以本地门禁+102 复验为准。

## 二、逐页走查表

### 2.1 billing 计费与账务(7 页)

| 页面 | 典型操作步数 | 反馈链缺口 | 表单问题 | 表格问题 | 选择器 |
|---|---|---|---|---|---|
| /billing/arrears(102行) | 停机/复机行内 2 步(danger 确认 41,44) | 成功静默 reload(44-45);失败内联 banner(70) | 无检索输入(28-31) | 客户列 名称或#id 回退(77);stop 不按 status 置灰(83-85) | 无 |
| /billing/billing(110行+invoices 126+run-modal 68) | 出账≈3 步(页头按钮→账期→提交,index:58/run-modal:36);发票作废/回填/重开各 2 步(invoices:88-90) | invoices load 无 busy(29-34),操作成功静默(48);run-modal 全链最完整(53/57/62)为域内范本 | 账期自由文本无格式校验(run-modal:54;对照 paycheck:13 PERIOD_RE);发票弹层手写 overlay+裸 input(106-113) | status/taxStatus 裸英文文本(83,85),StatusTag 注册表无 invoice/tax 域(registry.ts:12-15);taxNo 与 taxStatus 混一列(85) | 两处客户过滤 ResourcePicker(47/60)✓ |
| /billing/collection-tasks(131行) | 状态筛选 1 步;行内开始/完结/失败 2 步(含确认 40) | 成功静默(44-45) | 状态枚举用按钮组非 Dropdown | 裸 id 列(87);分页无总数文案 | 无 |
| /billing/daily-close(152行) | 日期改即刷;行内回填 2 步(101-111) | 行级 saving(111)+成功/差异 notice(61)✓ 域内最佳 | counted 有 placeholder/min/step ✓ | 明细 #billId 裸(133);diff 金额未 fmtFee(61);pagerTexts 误用 payment 文案(147) | 无 |
| /billing/paycheck(194行) | 平账行内 2 步(danger 79);账实=账期+查询 2 步 | settle 成功静默(82-83);账实服务端分页+汇总 ✓(60-67) | 账期有正则校验(13)✓,无月份选择组件 | settle 仅 DIFF_PENDING 可平(135)✓;statement/auto 缺入口 | 无 |
| /billing/payment(91行+CounterPaymentForm 132) | 柜面收款≈9 步(页头 1 步+6 字段+确认+提交);权限 gating ✓(19,52) | 成功 notice 带 payNo(46,86)+SubmitButton loading(126)✓;site/bill 拉取失败静默置空(43,57)→账单拉失败仅剩「预存」选项,资金错通道风险(P0) | amount 无单位/辅助说明(106);counter 无辅助(121) | #billId 裸且无客户列(65);billId 过滤裸数字输入(49-50);POST /payments/:id/refund 全前端无入口 | CustomerPicker+账单联动 Dropdown+method/site Dropdown 全 ✓(91,96,110,118) |
| /billing/stopsrv(104行) | 重试 2 步(86;确认 39 非 danger) | 成功静默(42-43) | 客户过滤 ResourcePicker ✓(58) | #id/#customerId/#loAccountId 三列全裸 ID(78-80),全域最差;retry 仅 FAILED ✓(84) | ResourcePicker ✓ |

### 2.2 oss 网络资源(8 组)

| 页面 | 典型操作步数 | 反馈链缺口 | 表单问题 | 表格问题 | 选择器 |
|---|---|---|---|---|---|
| /oss/resource(130行+path-drawer 108) | 链路反查/变更历史行内 1 步(87-88)✓ | resources 失败静默吞(40);历史失败静默空(50) | 设备筛选 Dropdown 未开 searchable(60-66) | #addressId/#orderId 裸(82,84);客户端分页 total=前端行数(99) | 未用 ResourcePicker |
| /oss/capacity(113行) | 告警扫描 1 步(76) | scanMsg 内联成功文案(80) | 阈值 80 前端硬编码(12),与后端 CapacityWarnThresholdPct 双份 | ≥80% 行内联高亮(87,96) | — |
| /oss/device(134行) | 只读 | — | — | 页签内联 style+#f0f0f0/#1677ff/#666(61-67),暗色主题必破相;metrics 首列 #id(94);maint 页签无筛选 | — |
| /oss/reserve(101行) | 释放 2 步(行按钮→danger confirm 37→POST),仅 HELD 行显示(81)✓ | — | — | reserveId/portId/orderId 全裸 #id(76-78) | ResourcePicker 直用且 load 全量拉 /ports(56-65),未用 search 防抖 |
| /oss/transfer(178行) | 审批/驳回 2 步(reject danger 70,111-113);创建抽屉 2 步 | — | 区域 Dropdown 无 searchable(155,164) | #resourceId 裸(104);regionName 回退 #id(82);内联 margin:0(172) | ResourcePicker(142) |
| /oss/expand(153行) | 创建 2 步 | — | expectedPorts 无辅助说明(143) | 无行操作=与后端仅 GET+POST 对齐 ✓;内联 margin:0(147) | — |
| /oss/loaccount(114行+ResetPassword 41) | 重置密码 2 步(danger 43→POST→明文一次性弹层+CopyButton 26-29)✓ | 错误 banner | 关键字每击键即请求无防抖(39,63-64);对照 ResourcePicker 自带 300ms | #customerId/#qosTemplateId 裸(87,90);唯一服务端 total 分页 ✓(32-35);状态枚举 Dropdown ✓(67-72) | — |
| /oss/odn(118行+coverage 99) | 退役 2 步(danger confirm 71)✓;新增 1 步展开表单+保存 | submit/load/resolve catch 静默吞服务端错误(60,67;coverage 35,53);成功零反馈(仅 reload) | 枚举字段全自由文本:facility kind(108)/grid status(107)/device kind(110);数值靠正则猜类型(99-103);coverage addressId/deviceId 裸 ID 手输(66-68);CityFilter 硬编码中文不走 i18n(32-33)+默认 PHL001/MNL 写死(42-43) | status 原样文本无 StatusTag(114-117);四页签均无分页;coverage 固定 limit 200(coverage:35)/updatedAt 未格式化(94);device parentId 裸 id 列(117) | 无关联选择器 |

### 2.3 provision 配置下发(3 页)

| 页面 | 典型操作步数 | 反馈链缺口 | 表单问题 | 表格问题 | 选择器 |
|---|---|---|---|---|---|
| /provision/provision(99行) | 重试 2 步(FAILED 行内+确认 79-83)✓ | 重试成功静默 reload(已知登记债) | 状态筛选 Dropdown ✓(56-61) | #orderId/#loAccountId/#templateId 全裸(73,75,76);客户端分页无服务端检索 | — |
| /provision/template(141行+TemplateForm 31) | 新建/编辑抽屉 2 步;删除 danger ✓(62) | 成功静默 | toggleStatus 一键禁用无确认(48-59);legalEntity Dropdown 不可搜(TemplateForm:29)/无 placeholder;JSON 错误文案硬编码英文不走 i18n(23) | #id 裸(103);状态列纯文本非 StatusTag(108);boundOffers ✓(111) | legalEntity 待 searchable |
| /provision/provlog(108行+detail 65) | 详情抽屉 1 步(43),含指令/应答原文+FAILED 高亮(57),域内最佳 | 失败 banner | task 过滤 ResourcePicker+result 下拉 ✓ | #id/#taskId 裸(83-84) | ResourcePicker(60) |

### 2.4 quad 四码合一(3 页)+ alarm(1 页)

| 页面 | 典型操作步数 | 反馈链缺口 | 表单问题 | 表格问题 | 选择器 |
|---|---|---|---|---|---|
| /quad/quadlink(68行) | 仅刷新按钮(38)——建链与 by-asset/customer/port/address 四反查零入口(P1 能力缺口,后端 POST /quad-links+GET by-* 均在 quadlink.go:15-19) | — | 无表单(因无入口) | 5 列全裸 #id/#assetId/#customerId/#portId/#addressId(47-51);无筛选 | 无 |
| /quad/check(114行) | reconcile/resolve 各 2 步(含确认 38,57)✓;reconcile 结构化 hint(43-47)✓ | 成功=内联蓝字 hint;失败 banner | 状态筛选仅 URL query(22),无页面控件 | 【PP1-C 遗债】81 行 hint div 内联 style+裸 #1677ff 且与 className 双写冲突;resolve 按钮未按状态门控(95-99);#id/#assetId/#customerId/#portId/#addressId 全裸(89-93)——fields.md §5.1 明确 API 有 assetCode/customerCode/portCode/addrCode 展示冗余可用 | 无 |
| /quad/scanlog(78行) | 只读 | — | — | #orderId 裸(60,有 picker 却不显 orderNo);无时间列(59-63);result StatusTag ✓(63) | ResourcePicker 全量拉 /orders(42) |
| /alarm/alarm(149行) | ack 2 步(OPEN 行门控+确认 41-52,109-113)✓;batch-retest 抽屉(scope 必填,返回 taskNo 54-70)✓ | 成功=内联 hint(62);142 行校验条件恒 false,错误提示永不显示(死代码) | scope 有 label/placeholder ✓(139-141) | 【PP1-C 遗债】94 行内联 style+裸 #1677ff;按资源过滤但行内无资源归属列;status/level 筛选仅 URL(24),无下拉 | ResourcePicker 直用(80-89) |

### 2.5 aaa(3 页)+ intel 残留(4 页)

| 页面 | 典型操作步数 | 反馈链缺口 | 表单问题 | 表格问题 | 选择器 |
|---|---|---|---|---|---|
| /aaa/dashboard(88行) | 刷新 1 步 | — | — | 六指标卡+最近刷新 ✓;closed 指标取用未展示(11) | — |
| /aaa/aaasession(148行) | 强制下线 2 步(danger 确认+异步受理 banner+行级 busyId,90,128-136)✓ 全域反馈最佳 | — | status 下拉缺 OFFLINE/OFFLINE_FAILED 两枚举(115-124);loid 输入无辅助 | 会话字段齐 ✓;服务端分页 ✓;内联 style width(109,112) | — |
| /aaa/aaalog(107行) | 只读+过滤 | loid 过滤不生效:useEffect 依赖缺 loid(40),输入后须手点刷新(P1 功能缺陷) | loid 输入无防抖/辅助 | sessionTime 裸秒数(71);认证结果纯文本非 StatusTag(74,90);内联 style(56,57) | — |
| /intel/gis(215行) | 层级 Dropdown+刷新 1-2 步 | 点位点击仅认 6-8 级(162),ODN 设施/局点/设备点位点击零响应零提示(【U5 rider 现场】);ODN 图层全量拉取不随视域(66-67 自注) | parentId 裸数字输入框(126),无树选择 | drill 表 #id 裸(179);统计卡 ✓;外部瓦片代理拦截登记 | Dropdown ✓(120,130) |
| /intel/analytics(126行) | 只读 | 并发 3 接口 ✓ | emptyText 硬编码中文不走 i18n(89) | heat 回退 #addressId(54) | — |
| /intel/report(296行) | — | 全域唯一 sonner toast 合规页(6:成败 toast+失败可复制),Phase B 反馈链改造范本 | — | — | — |
| /intel/monthly(162行+子组件) | 服务端分页 ✓(56) | — | 派生列禁编辑 ✓ | DataTable/EditDrawer 组件化较完整 | — |

## 三、PP1-C 登记债现场核实(本域清偿对象)

| 位置 | 现场 | 处置建议(Phase B) |
|---|---|---|
| quad/check/index.tsx:81 | hint div 内联 style(padding/color/fontSize)+裸 #1677ff,且 className 已有 text-[var(--color-text-link)] 被内联覆盖,双写冲突 | 删内联对象,统一 hint 组件或令牌 className |
| alarm/index.tsx:94 | 同款:纯内联 style+裸 #1677ff | 同上 |
| 同类新增(登记):oss/device/index.tsx:61-67(#f0f0f0/#1677ff/#666)、oss/resource/path-drawer.tsx:57(var(--color-success, #52c41a) 裸回退)与 :44、billing/billing/invoices.tsx:57,114、billing/paycheck/index.tsx:107,110、oss/transfer:172、oss/expand:147、oss/capacity:87,96、aaa-session:109,112、aaalog:56,57 | 内联 style 或裸色值,暗色主题破相风险 | 全部收敛为 theme tokens/令牌 className |
| provision 全域成功反馈未走 toast | 登记债确认:provision/provision 重试成功静默、template 成功静默 | 并入 P0 反馈链统一修 |

## 四、删除类/不可逆操作 × 后端路由核对(逐项)

| 操作 | 后端路由(语义) | 前端现场 | 结论 |
|---|---|---|---|
| bills/payments/daily-closings/reconciliations/ar-* 删除 | 无任何 DELETE 路由(billing.go/tax.go 全表核对) | 前端也无删除按钮 | ✓ 计费只读事实记录未强加删除,保持 |
| 发票作废/重开 | POST /invoices/:id/void,reissue(tax.go:25,27,状态流转非物理删,void_reason 留痕) | 手写弹层+原因必填(109-117) | 对齐;弹层宜迁 ConfirmDialog(danger) |
| 欠费停机 | POST /arrears/:customerId/stop(停服,经 LO 账号即时生效) | danger 确认(41)✓ | 对齐;补明示「不可恢复需复机」文案 |
| 端口预占释放 | POST /reserves/:id/release(resource.go:28) | danger 确认(37)✓ | 对齐 |
| ODN 网格/设施/局点/设备退役 | DELETE /odn/* → odnRetireGridHandler→a.ODN.RetireGrid,退役非物理删(odn_handlers.go:70-83;RETIRED 永久锁定) | danger 确认(71)✓ | 对齐;文案可补「退役后编码不可复用」 |
| ODN 绑定解绑 | DELETE /odn/bindings/port/:portId(odn_bindings.go:28) | 零 UI | 并入 ODN 能力缺口 |
| AAA NAS 删除 | DELETE /aaa/nas/:id = 真物理删(aaa_nas.go:136-154,svc.DeleteNas+审计) | 全前端零调用、零 UI(grep aaa/nas 零命中) | P1 能力盲区:Phase B 补管理入口必须 danger ConfirmDialog+明示不可恢复 |
| 下发模板删除 | DELETE /provision-templates/:templateId(物理删;守卫仅查 provision_tasks 引用,pg_template_custom.go:47-55,被套餐绑定但无任务的模板可删→悬空引用) | 删除 danger ✓(62) | 前端不动;后端守卫缺口登记(禁改后端,报负责人) |
| quad 解链/告警关闭/删除 | 后端无路由 | 前端也无假按钮 | ✓ 无假入口 |

## 五、修复清单(Phase B 执行序,按任务书分级)

### P0 反馈链缺失(全域横切,最小统一修)

1. 操作成功反馈统一 toast.success:billing 5 页静默(arrears 44-45/invoices 48/collection-tasks 44-45/paycheck 82-83/stopsrv 42-43)+provision 2 页+oss/odn 提交+quad check reconcile+alarm batch-retest(范本=intel/report:6 与 aaasession 异步 banner)。
2. 失败原因可见且可复制:oss/odn(60,67)与 coverage(35,53)catch 静默吞 e.message 必须透出;ErrorBanner 补复制按钮(components/business/page-head.tsx:25-31,全域共用)。
3. billing/payment site/bill 拉取失败静默置空(43,57)→失败必须提示并阻断提交(资金错通道风险);invoices load 补 busy(29-34)。

### P1 表单/表格债与遗债(含能力缺口)

1. 裸内部编号换人读名(fields.md §5.1 四码 code 冗余、订单号/LOID/模板名等;取不到降级 #ID 并留痕):stopsrv 78-80、quad/check 89-93、quadlink 47-51、scanlog 60、provision 73-76、provlog 83-84、resource 82,84、reserve 76-78、transfer 104、loaccount 87,90、device 94、collection-tasks 87、daily-close 133、payment 65、gis 179。
2. 遗债清理:三节表格所列内联 style/裸色值全部收敛(含 PP1-C 两处 :81/:94)。
3. 表单选择器化:oss/odn 枚举自由文本(kind/status)改 Dropdown、coverage addressId/deviceId 接可搜选择器;invoice/tax 状态接 StatusTag(补注册表);aaasession status 枚举补全;TemplateForm legalEntity searchable+placeholder+英文文案 i18n;账期(run-modal:54)复用 PERIOD_RE。
4. 能力入口对齐(API 有前端无,Phase B 逐项与负责人确认后补):NAS 注册表管理页(真删须 danger+不可恢复明示)、quadlink 建链+四反查、POST /payments/:id/refund、发票 tax-submit/retry/replay+tax-events/tax-failure、reconciliations statement/auto、AR 四组(aging-snapshots/payment-promises/writeoffs/dunning-runs,自然挂载点=billing/arrears)、ODN 22 条路由(光缆/施工/端口/绑定/生命周期,归 U5 或拆后续任务)。
5. 功能缺陷:aaalog loid 过滤 useEffect 依赖(40);alarm 142 死条件;arrears stop 状态门控(83);check resolve 状态门控(95-99);template toggleStatus 加确认(48-59);loaccount keyword 防抖(39);collection-tasks 标 FAILED/stopsrv retry 确认改 danger。
6. oss/odn CityFilter 硬编码中文 i18n 化(32-33)+城市来源下拉化。

### P2 一致性

1. 客户端分页 6 页(resource/capacity/device/reserve/transfer/expand)与 quad 三页尽量迁服务端分页或补总数口径;分页文案 pagerTexts 误用清理(daily-close:147)。
2. ResourcePicker 直用 9 页(alarm 80/billing 47/invoices 60/stopsrv 58/provlog 60/scanlog 42/reserve 56/transfer 142 等)在 W0 兼容层合并后零改动受益,逐页复验即可;scanlog 全量拉 orders 与 reserve 全量拉 ports 改远程检索。
3. Dropdown 补 searchable(资源/区域/法人);手写 overlay 弹层(invoices 106-113/payment 84)统一 Dialog/ConfirmDialog;跨页分页文案与列宽节奏统一;intel/analytics 中文硬编码 i18n;capacity 阈值前后端双份登记待配置化。

## 六、Phase B 验收对照(预告,路线图第三节+纪律第四节)

1. make web-admin-check 全绿(TZ=Asia/Shanghai);quad/alarm 内联 style 清零(grep style={{ 与 #hex 双零);逐页过第三节 6 条规则;102 真实走查复验(实截+DOM 断言+console/网络零报错);证据落 docs/acceptance/2026-09-07-pp2-w3-ops.md。
2. 前置:merge main(须已含 W0 选择器基座)后再动手;U5 rider(T14 ODN 前端强化)有余力再做,单独提交+独立验收文档。
3. 硬纪律提醒:禁新增业务功能(能力入口类 P1-4 项须负责人逐项点头)、禁改后端(NAS/template 守卫缺口只登记)、禁新依赖、禁原生 select/裸 alert、i18n 三语键 append-only 独立小提交、单文件 ≤300 行、vite dev 用完关停。
