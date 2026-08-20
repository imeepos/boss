# 师傅端页面对接 API 约定(所有改造页面必须遵守)

> 契约: `api/openapi/worker.yaml`(拆分于 `api/openapi/worker/*.yaml`)
> 服务端: 真实服务端(Go,统一信封 `{code,msg,data}`,HTTP 恒 200;api.js 已解信封,页面消费平铺 data)
> 对接层: `docs/worker/api.js`,页面引入后使用 `window.API.*`;Android 端同构解信封见 `mobile/worker/android/.../api/Api.kt`

## 硬性规则

1. 每个页面在 `<script src="nav.js"></script>` 之前加 `<script src="api.js"></script>`。
2. 页面展示的**业务数据**(工单/客户/统计/消息等)一律改为从 `API.*` 拉取渲染;纯静态说明文案(notice 提示)保留。
3. 保持现有 DOM 结构与 class 不变(渲染容器可给 id),样式不重写;列表类内容用 JS 拼 HTML 字符串填充。
4. JS 风格 ES5(`var` / `function`),与现有页面一致;不用框架、不用模块。
5. 单文件不超过 300 行,函数不超过 30 行。
6. 工单号跨页传递统一用 URL query `?no=xxx`(如 `order.html?no=TKT-20250817-012`);取值:
   ```js
   var NO = new URLSearchParams(location.search).get('no') || 'ORD-20250817-001';
   ```
   页面间跳转链接涉及具体工单的,拼上 `?no=' + NO`。
7. API 调用统一:
   ```js
   API.ticket.detail(NO).then(function (d) { render(d); }).catch(function (e) { /* 页面内提示,不白屏 */ });
   ```
8. 操作类按钮(领取/抢单/签到/提交等)改为真实调用 `API.*`,成功后用返回的 `message` 提示(alert 或现有提示方式),再按需刷新/跳转。
9. 不新增页面,不删除现有入口。

## api.js 方法速查

- `API.home.get()` 工作台聚合
- `API.ticket.list(status)` / `.history(period)` / `.detail(no)` / `.accept(no)` / `.checkin(no,lat,lng)` / `.navi(no)` / `.transfer(no,reason,targetId,remark)` / `.reschedule(no,newDate,newSlot,reason,remark)` / `.rollback(no)` / `.retry(no)` / `.complaint(no,category,content)` / `.repairReport(no,result,remark)`
- `API.hall.list()` / `.grab(no)`
- `API.scan.bind(no,epc,offline)` / `.abnormal(no,payload)` / `.photos(no)` / `.uploadPhoto(no,scene)` / `.report(no)` / `.submitReport(no,remark)` / `.activation(no)` / `.activate(no)` / `.sign(no,signatureData)` / `.charge(no)` / `.submitCharge(no,amount,payMethod)`
- `API.asset.dismantleScan(no,epc)` / `.replace(no)` / `.submitReplace(no,oldEpc,newEpc)` / `.returnAsset(epc)` / `.materials()` / `.materialOut(id)` / `.tools()` / `.borrowTool(id)` / `.giveBackTool(id)` / `.maintenance()` / `.measure(no)` / `.resources(no)`
- `API.profile.get()` / `.performance(period)` / `.schedule(month)` / `.clock('IN'|'OUT')` / `.settings()` / `.saveSettings(s)` / `.feedbacks()`
- `API.misc.messages()` / `.readAll()` / `.clear()` / `.notices()` / `.faq(keyword)` / `.serviceMessages()` / `.sendServiceMessage(content)` / `.safetyCheck(workType,checklist)`
- `API.auth.login/smsCode/logout`

## 关键字段(与 mock 返回一致)

- 工单项: `ticketNo/typeLabel/address/distanceKm/scheduleSlot/stage/stageTotal/status/statusLabel/slaLeftMinutes/note/finishedAt`
- 工单详情: `product/customerName/customerPhoneMasked/splitterPort/preBindTag/scheduleSlot/faultTypeLabel/reportedAt/slaLeftMinutes/remoteDiagnosis/stages[{stage,name,result,finishedAt,note}]/quad{assetCode,customerCode,portCode,addrCode}/riskCheck{blacklistHit}`
- 扫码结果: `matched/result(MATCH|MISMATCH|OFFLINE_CACHED)/message/quad`
- 激活: `loid/status/statusLabel/lastTry`
