# oss-odn v2 重构验收证据(2026-09-08,O1-O5)

> 分支 feat/oss-odn-v2-ui(worktree boss-odn-v2);后端 102 http://192.168.0.102:28080(无新增后端、无迁移)。
> 实现规格 designs/oss-odn-v2.spec.md(commit 3d398e6a);截图经 cdp-admin-capture.mjs(vite dev @5317 → 102 登录注入)。

## §8 六项实现验收自查

1. **左树右表 + 四级树 + 叶子联动 — 通过**。左树卡片 280px 固定 + 右主区自适应,间距 16px;
   树 省(PHL001)→市(MNL)→分组(OLT 设备/ODN 网格/局点/施工项目,折叠开关+图标+计数 badge)→
   节点(类型图标+编码/名称+状态点:绿=ACTIVE/IN_USE/ONLINE,金=RESERVED/容量预警/BUILDING,灰=RETIRED/PENDING)。
   点 L2 主区切市汇总,点 L4 切 Tab 并过滤该行(截图证据:MNL 下 网格码 91 行 + 树 91 acc-grid-91 / P91011 叶子;
   采集轮 eval 实测:点 OLT 叶子跳 /oss/device?resourceId=228,点市节点 URL 变 city=MNL 且网格行可见)。
   路由 ?prv=&city=&tab= 同步,刷新可还原(preserve-unknown-keys 语义,city 锚定经 URL 修正)。
2. **新增/编辑只经抽屉 — 通过**。ODNForm 删除(grep 断言 0 命中),ResourceDrawer.tsx 440px 右侧 Drawer
   (遮罩/ESC 关闭);表格行「编辑」与详情抽屉「编辑」均开同一抽屉。批量导入保留页头。
3. **详情只经抽屉 — 通过**。DetailDrawer.tsx 480px 四分区(基本信息两列 k-v / 关联关系纵向链行卡点击换
   详情对象 / 资产信息 assetReg 凭证 badge 或未登记灰字 / 操作记录时间线);表格行「详情」开抽屉。
4. **无原生 select — 通过**。`! grep -rn '<select' src/pages/oss/odn` 断言 0 命中;枚举/关联一律
   components/Dropdown.tsx(省/市/类型/网格/局点/上级/状态/生命周期,网格码 01-99 可搜索 Dropdown)。
5. **关联列 + 关联 OLT KPI — 通过**。设施表所属网格、设备表所属局点(NodeCode 形态)、设施/局点/设备表
   关联 OLT 蓝色圆角 chip(color-mix 14% 蓝 tinted + mono 12px),点击跳 /oss/device?resourceId= 预过滤
   (device 页已接 query param);KPI 四卡 网格/设施/局点/关联 OLT(第四卡蓝 tinted 图标+网格卡容量条)。
   i18n 锚点 linkedOlt 存在于 zh-CN/en-US/ms-MY/types.ts(grep 断言);keys.test.ts 三语键集对齐绿(531 tests)。
6. **三门禁全绿 + 双主题截图 — 通过**。pnpm typecheck / test(531 passed)/ build(含 web-ui-audit)全绿;
   截图 docs/acceptance/assets/oss-odn-v2-light.png、oss-odn-v2-dark.png(1600×900,MNL 数据态)。

## 截图

- 亮色:docs/acceptance/assets/oss-odn-v2-light.png(PHL001/MNL,树四级展开、四 KPI 卡、网格分区表+分页)
- 暗色:docs/acceptance/assets/oss-odn-v2-dark.png(同数据态,全主题变量渲染,tab 金色下划线/树选中态/预警点)

## console / 网络采集结论(cdp-capture --logs)

- light:entries 391,console error **0**,network ≥400 **0**(380 条网络事件);warnings 4 条均为
  React Router v7 future-flag 存量框架提示,与本页无关。
- dark:entries 388,console error **0**,network ≥400 **0**;warnings 同上 4 条。
- 采集时加载失败路径均未触发(四列表 allSettled 加载 / OLT / 施工项目 / 覆盖 / 审计均 200)。

## 颜色令牌复查

- grep 断言:odn 目录 .tsx/.ts 无裸 hex、无裸 rgba;全部经主题变量或 color-mix(tinted 底 14% alpha)。
- 金色 #D5A63A(--color-brand-gold-500)仅三处用途:Tab 激活下划线、树 L4 容量预警/RESERVED 点、
  树选中态经 --shell-menu-active-bg 令牌(暗色随主题翻转)。DetailDrawer 关联链网格行原用 gold-300,
  已收敛为 --color-warning 以守住金色三条用途。
- 状态色面积 ≤5%(状态点 8px/Tag 描边浅底),关联 OLT chip 非状态色。

## 接口信封复核(对接前 curl 实测)

- /odn/regions|cities|grids|facilities|sites|devices|constructions|coverage/list:envelope data 为**裸数组**;
- /resources:data 为 **{items}**(unwrap 后取 items,按 type=OLT 过滤);
- /device/metrics:{items};/audit-logs:{items},targetType=odn_facility/odn_site/odn_device
  (internal/httpapi/admin/odn_lifecycle.go RecordAudit),无 targetId 服务端过滤,客户端按 targetId 筛。
- 设施/局点/设备创建 body 需含 prvCode/cityPrefix(handler BindAndValidate required),抽屉已补齐(旧内联表单缺漏,顺带修复)。

## 偏离与裁定(实现期)

1. 设施/局点/设备**编辑档位**:契约只有 lifecycle PUT(无字段编辑端点),编辑抽屉对这三类实体为
   身份字段只读 + 生命周期 Dropdown;网格为字段 PUT 可改。任务书「设备必填所属局点」按任务执行,
   契约本身允许市域设备可空,后端校验不变。
2. 表格坐标列移除(设计稿表格列无经纬度,坐标移至详情抽屉基本信息),资产列移至详情抽屉资产信息区
   (W8 资产列先例保留数据不丢)。
3. 关联 OLT chip 与树 OLT-市关联为**编码段逻辑桥接**(OLT-<CITY>-NN,spec §3 桥接裁定,不做 FK 直穿);
   102 无 odn_device 存量数据,设备 Tab 关联链以 OLT/施工项目叶子和覆盖环为可验证形态。
4. 关联链覆盖环数据源 /odn/coverage/list(limit 200)按 facilityCode/deviceId 归链统计 SERVED。
5. 操作记录时间线依赖 menu:audit 权限;无权限时 403 → 「暂无操作记录」空态,不阻塞详情(sysadmin 实测可见 odn.lifecycle 记录)。
6. 施工项目叶子点击联动切 Tab(面板自治无行过滤接口),未做行级过滤;网格/设施/局点叶子为行级过滤联动。

## 任务/commit 台账

| 任务 | commit | 三门禁 |
|:-----|:-------|:-------|
| O1 左树右表+四级树 | a5592bad | typecheck ✓ / test 531 ✓ / build+audit ✓ |
| O2 KPI+关联列+关联链 | e97aa934 | typecheck ✓ / test 531 ✓ / build+audit ✓ |
| O3 新增/编辑抽屉 | 457b9b6a | typecheck ✓ / test 531 ✓ / build+audit ✓ |
| O4 详情抽屉 | 0cfd912e | typecheck ✓ / test 531 ✓ / build+audit ✓ |
| O5 收尾+证据 | (本提交) | 见上,截图双主题已存档 |