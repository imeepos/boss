# 全站体验一致性复核报告(Wave 3 收官,2026-09-08)

> 执行分支 feat/ux-final(基于 main 09582ee9);范围=基座遗留 + 外围页扫尾 + 全站一致性复核。
> 走查证据:102 真实环境(http://192.168.0.102:5180,cdp-admin-capture DOM 断言 + console/网络采集)
> + 本地 dev 分支构建(http://localhost:5213,dark/en-US)改造页复验;截图与 logs JSON 存 /tmp/ux-final-*。

## 一、各域覆盖状态

| 域 | 批次 | 状态 | 备注 |
|---|---|---|---|
| bss / boss / billing | W1 | ✅ 已收官(main) | 逐页走查完毕 |
| org / base / ams / quad / provision | W2 | ✅ 已收官(main) | 逐页走查完毕 |
| intel / oss / partner / profile | W2 | ✅ 已收官(main) | 逐页走查完毕 |
| 选择器族(components/pickers + Dropdown 基座) | W0 + W3 | ✅ 本轮补齐基座两条遗留 | 见 §二.A |
| 外围页(home/dashboard/alarm/aaalog/aaa-session/aaa-dashboard/backup/news/profile) | W3 | ✅ 本轮走查+打磨 | 见 §二.B |
| 壳页(auth-shell/auth-ads/login/error/placeholder) | W3 抽查 | ✅ 只做顺手核对 | 未发现需修项;news 详情加载态 i18n 已修 |

组件采用全站盘点(feat/ux-final 工作树,254 个页面 tsx):
StatusTag 57 页 / ConfirmDialog 63 页 / sonner 反馈 115 页 / ui-table 103 页 / ui-card 115 页 /
ErrorBanner 142 页 / pickers 族+ResourcePicker 35 页;原生 select 0、window.confirm 0。

## 二、本轮修复(feat/ux-final 提交清单)

### A. 基座遗留(selector-audit §一.1 ①②销账)

| # | 修复 | 提交 | 验证 |
|---|---|---|---|
| 1 | Dropdown 补 `onOpenChange`(仅开合过渡触发,mount 不触发,ref 转发);SimplePicker 服务端源 open=true 且关键字非空时经新增 `srv.reopen()` 复位检索(关键字/committed 清零+retryTick 强制重发首屏),修「重开浮层输入框已清而列表残留旧检索结果」窗口 | a6370c77 | vitest SimplePicker.test.tsx 两路径(开合通知矩阵/重开复位重发首屏);ResourcePicker 服务端源 16 直用页自动继承 |
| 2 | onDark 深色表面样式令牌化:tokens.css 双主题块新增 `--shell-dark-input-*`(bg/border/hover/focus/text)与 `--shell-dark-pop-*`(bg/border/shadow/hover)九键,Dropdown 常量色(white/5、white/10、white/20、white/40、white/60、rgba 阴影)全部换令牌 | 4223ba9f | 令牌值与原常量逐一相同(Tailwind white/N 即 rgba(255,255,255,N%),navy-900=#10203F),计算样式按构造不变;顶栏 TopNav 调用方零改动 |

### B. aqp 批次组件建议(components/ 只做加法向后兼容)

| # | 修复 | 提交 | 验证 |
|---|---|---|---|
| 3 | ToolbarButton 支持 `className`(cn 合并,冲突类调用方胜出);OrderItemsEditor 删行钮去 wrapper div 改直挂 `col-span-1 w-full` | e6159ee3 | vitest page-head.test.tsx(默认/primary 两态合并断言) |
| 4 | SubmitButton 增加显式 `danger` prop(危险动作 idle/loading 危险色实底,success/failed 语义色不变) | 81c00a6c | vitest submit-button.test.tsx(danger/非 danger 分支底色断言) |
| 5 | inventoryPage 分页词条六键自立(prev/next/perPage/rangeText/jumpText/pageUnit,types+三语 append-only),去 pages.company 借用 | 95727ed2 | typecheck + keys.test 三语键集一致;文案值与原借词条相同 |

### C. 外围页扫尾(102 真实走查 + DOM 断言留证)

走查结论:9 页全部真实渲染、三语/双主题抽查通过、console error=0、失败请求=0(证据 /tmp/ux-final-*.json、/tmp/w3-*.json)。注:alarm/aaalog/aaa-session/aaa-dashboard/backup 真实路由为 /alarm/alarm、/aaa/aaalog、/aaa/aaasession、/aaa/dashboard、/base/backup(首探 404 系路径映射,非页面缺陷)。

| 页面 | 缺口 | 修复 | 提交 |
|---|---|---|---|
| alarm | 裸卡片壳 div + 裸 table(巨串 thead/td 类)+ 裸 button 工具钮 + 无样式裸确认钮 + 错误行不可复制 | Card/ui-table/ToolbarButton/ErrorBanner/ActionLink 收口 | a01e9c58 |
| aaalog | 同上(双页签两套裸 table)+ 筛选行裸贴卡边 | 同上 + 筛选行补 px-4 | a01e9c58 |
| aaa-session | 裸卡片壳 + 裸 table(SessionsTable)+ 裸工具钮 + 错误行不可复制 | Card/ui-table/ToolbarButton/ErrorBanner;强制下线钮保留原 danger 描边设计 | a01e9c58 |
| dashboard | 订单状态分布裸 table(shadcn 语义类) | 换 ui-table,键盘下钻行(Tab/Enter/role=link)保留 | aec8c062 |
| aaa-dashboard | 更新时间 toLocaleString 随机器时区;错误行不可复制 | fmtTime + CopyButton(对齐 dashboard 错误横幅模式) | aec8c062 |
| backup | 裸页头块;错误行不可复制 | PageHead + CopyButton | aec8c062 |
| news | 加载态硬编码 Loading… | t.common.loading 三语词条 | aec8c062 |
| home | — | 无缺口(分区组件化、i18n/令牌完整) | — |
| profile(/ucenter/*) | — | 无缺口(SubmitButton/Input/令牌齐备) | — |

### D. 一致性复核顺手修

| # | 修复 | 提交 |
|---|---|---|
| 6 | oss/odn/coverage.tsx `clearLabel="×"` 裸字面量换 pages.pickers.common.clear(audit §二.1 销账) | 5d73ab67 |

## 三、全站机械四项残留清单(修后复测)

| 模式 | 修前命中 | 修后 | 说明 |
|---|---|---|---|
| 原生 `<select>` | 0 | **0** | 无回潮 |
| `window.confirm` | 0 | **0** | ConfirmDialog.tsx 仅注释提及,豁免 |
| 裸卡片壳串 `mb-4 rounded-md border border-[var(--shell-card-border)]…` | 4 文件(alarm/aaa-session/aaalog/ui-card) | **0**(仅 ui/card.tsx 自述注释) | 三页已收口 |
| 裸 table 壳串 `w-full border-collapse` | 5 文件(dashboard/alarm/aaa-session/aaalog/ui-table) | **0**(仅 ui/table.tsx 规范本体) | 四页已收口 |

## 四、StatusTag registry 覆盖度(登记建议,不强改)

现注册 30 域(order/port/asset/bill/payment/service/realName/recon/ledgerRecon/reserve/product/quad/tag/resource/loAccount/aaaSession/ticket/task/complaint/scan/alarmLevel/alarmStatus/maintPriority/accountStatus/message/backupStatus/procurement/receipt/installLog/userdata)。以下页内三语状态映射建议后续入册(枚举口径以 docs/contract/fields.md 为准):

| 建议域名 | 现状位置 | 说明 |
|---|---|---|
| odnResourceStatus | oss/odn/DetailDrawer.tsx + ResourceTree.tsx 内联 ACTIVE/IN_USE/IN_SERVICE/ONLINE/RESERVED/BUILDING/PENDING→色变体映射 | 两处重复实现,最值得入册 |
| odnProjStatus | i18n odnPage.projStatus(PENDING/BUILDING/ACCEPTED),AssetsPanel/permits 两处消费 | 选项构造处裸拼 |
| invoiceStatus | billing/billing/invoices.tsx statusTexts(页面注明「注册表无 invoice 域,不越界改公共件」) | |
| invoiceTaxStatus | invoices.tsx taxStatusTexts | |
| payableStatus | billing/payables/index.tsx statusTexts | |
| cdrAcctStatus | aaalog acctStatus 数值枚举三语数组 | |
| billingStatus | aaalog BILLED/UNBILLED 三语 | |
| authResult | aaalog SUCCESS/FAILED 三语;aaa-session 已有 aaaSession 域,可合并口径 | |
| callbackResult | boss/callback result 现裸渲染英文枚举(连 i18n 都缺,优先级最高) | |

## 五、遗留债务汇总(不阻塞,勿重复登记)

1. **后端侧(中央登记 docs/contract/data-relations.md §6,三条)**:归属链公司名行内缺失;worker/customer 详情缺班组/区域名称字段(detailItems #id 兜底);存储配置缺 /storage-config/test。
2. **ResourcePicker 直用页 16 处,pinnedOptions 已传 8 页**(OrderCreateDrawer/OrderEditDrawer/billing/billing/invoices/stopsrv/dispatch/message/worker);未传 8 处(alarm 无 URL 重hydrate 无实际缺口、ams/replace、boss/dismantle、boss/worker-registration、oss/reserve、oss/transfer、provision/provlog、quad/scanlog、TeamDialogs)——刷新/重开筛选值回显裸编号,页面触及时补传或迁移 SimplePicker。
3. **DialogPicker initialItems**:boss/release 已接;oss/odn/MaterialIssuesCard 未接(重开不回显上次选择)。
4. **UserPicker 全站零消费**:保留备用,随清理批次裁定去留。
5. **EntityPicker 详情抽屉**:worker 班组/区域、customer 区域仍 #id 裸编号(债务 1 关联,API 补字段后消化)。
6. **RegionCascadePicker 直搜失败重试**不单独重发直搜(低频);组件 250 行近红线,加功能先拆列组件。
7. **键盘/焦点回归基线**:本轮 Dropdown 仅加通知不改键控路径;pickerCore.moveActive 键盘回归与 102 DOM 断言既有锚点不变。

## 六、给后续迭代的建议

1. **机械四项纳入 CI 门禁**:三条 grep(原生 select/window.confirm/裸壳串)已在 W3 归零,建议进 make check 或 CI lint 防「回潮」,豁免名单只留 ui/card、ui/table、ConfirmDialog 注释。
2. **onOpenChange 类回调继续加法**:DialogPicker/MultiSelect 如需重开复位/开合埋点,复用同签名 `(open: boolean) => void`,与 Dropdown 对齐。
3. **registry 登记节奏**:§四 清单随各页下一次触碰入册(一次一域,append-only 小提交),callbackResult 因裸枚举展示建议提前。
4. **外围页双主题截图人工复核**:本轮以 DOM/计算样式断言为主,截图存 /tmp 供人工抽验(w3-alarm/aaalog/aaa-session/dashboard/aaa-dashboard/backup.png 为 dark+en-US 改造后形态)。
5. ** debt 清理顺序建议**:先 API 侧名称字段(债务 1/2/5 同源)→ detailItems 消化一批;再 pinnedOptions 剩余 8 页(多数随功能迭代自然触碰)。

## 七、验证锚点

- 门禁:`TZ=Asia/Shanghai pnpm typecheck && pnpm test && pnpm build` 全绿(vitest 86 文件 550 用例,含新增 SimplePicker.test.tsx 2 用例、page-head.test.tsx 2 用例、submit-button.test.tsx 扩展 1 用例)。
- 102 走查:9 外围页 + /ucenter/personal/security + news 详情(m4-first-order-campaign)DOM 断言通过,console error=0、失败请求=0。
- 本地 dev 复验(dark/en-US,连 102 后端):6 改造页标题/表格/th 令牌色 rgba(255,255,255,0.04)(--shell-menu-hover-bg 暗值)断言通过;oss/odn 覆盖页签两选择器渲染正常。
- 部署核对惯例:push 后以远端 index.html bundle hash 与本地 dist/assets 同名比对,再 grep bundle 内新标记(如 --shell-dark-input-bg)二次确认。
