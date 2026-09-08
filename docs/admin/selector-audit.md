# 选择器族体验审计矩阵(Wave 0,2026-09-08)

> 范围:web/admin/src/components/pickers/ 全族 + ResourcePicker 兼容层 + Dropdown 视觉基座(只读参照)。
> 维度对齐任务书 7 条需求;每格记「现状 / 本次修复 / 遗留」。页面侧问题只记录不改页面,供 Wave 1-3 消化。

## 一、选择器 × 维度矩阵

图例:✅ 达标(有验证动作) | 🔧 本次修复(本分支 commit) | ⚠️ 遗留(注明责任侧) | ➖ 不适用

### 1. 通用下拉基座 Dropdown(components/Dropdown.tsx,参照基座,本波未改)

> W3 收尾(feat/ux-final)已补 ①onOpenChange 基座回调并令牌化 ②onDark;下表保留审计时点原文,修复结论以划线标注。

| 维度 | 状态 | 说明 |
|---|---|---|
| 键盘/焦点 | ✅ | 箭头/Enter/Space 开合、上下移动跳禁用并回绕、Esc 关闭焦点归还触发器、Tab 自然离开;listbox/option/aria-activedescendant 齐全(vitest pickerCore.moveActive + 102 DOM 断言) |
| 点击外部收起 | ✅ | document mousedown 监听,浮层内 mousedown 不误关 |
| 搜索 | ✅ | searchable 本地过滤;remote 模式关键字上抛;loading 行 role=status;空态行;防抖 300ms(pickerCore.PICKER_DEBOUNCE_MS) |
| 回显 | ✅ | withPinnedValue 合成钉选 + resolveOptionMatch label 同值兜底(W1 裁定,warn 留痕) |
| ⚠️ 遗留 | 基座侧 | ① ~~无 onOpenChange 回调~~ **已修(W3,feat/ux-final)**:基座补 onOpenChange(仅开合过渡触发,mount 不触发),SimplePicker 服务端源在 open=true 且关键字非空时复位检索(清关键字+重发首屏),重开浮层不再出现「输入框为空而列表是旧检索结果」窗口;回归锁定 pickers/SimplePicker.test.tsx。② onDark 深色表面样式用 white/10、rgba 阴影等常量(非 tokens 令牌),属常青藏青表面的固有色,记债务。 |

### 2. SimplePicker(小数据量基座)

| 维度 | 状态 | 说明 |
|---|---|---|
| 打开即用 | ✅ | 复用 Dropdown 全套键位;静态源 mount 即就绪,服务端源 mount 即首屏检索,开浮层无等待 |
| 搜索 | 🔧 | 防抖/loading/空态/竞态 seq 原达标;**错误行移到触发器上方**——浮层向下展开会遮挡原位置的重试按钮,现失败时原因+重试始终可见可点(commit feat SimplePicker) |
| 已选回显 | ✅ | 调用方 pinnedOptions 优先,基座 withPinnedValue 兜底 |
| 清空 | ✅ | clearable+clearLabel 口径(canClearValue):值非空且未禁用才出钮 |
| 一致性 | ✅ | 结构性文案缺省取 pages.pickers.common(三语 keys.test 锁键集);tokens.css 双主题令牌 |
| ⚠️ 遗留 | 基座侧 | ~~重开浮层残留旧检索结果窗口~~ **已随 Dropdown ① 修复(W3,feat/ux-final)**;服务端源 rehydrate 回显裸编号需调用方传 pinnedOptions(见 ResourcePicker 行) |

### 3. DialogPicker(大数据量弹框基座)

| 维度 | 状态 | 说明 |
|---|---|---|
| 打开即用 | ✅ | ui/dialog(Radix)焦点圈闭 + Esc 关闭;候选表行 tabIndex=0,Enter/Space 选中 |
| 搜索 | ✅ | 关键字防抖 300ms + 多维筛选 + 分页(limit/offset 由 buildPickerQuery 组装);失败 role=alert + 重试(texts.retry 缺省取 common.retry) |
| 大数据量 | ✅ | 服务端分页 + 跨页累选(多选)+ chips 回显 rowLabel;重置/变页竞态由 seq 防护 |
| 改选/清除 | 🔧 | **新增 initialItems(加法)**:重开时预勾选上次确认实体,chips 同步回显,单选取首项;确认可原样回传(pickKeysFromItems 去重保序,vitest 回归)。底部 chips 单个移除+清空已选原达标 |
| 一致性 | ✅ | 文案由调用方 texts 注入,组件零硬编码;布局对齐附件选择器弹框 |

### 4. EntityPicker + 实体三选择器(CustomerPicker/WorkerPicker/UserPicker)

| 维度 | 状态 | 说明 |
|---|---|---|
| 搜索 | ✅ | 服务端 keyword 检索(/customers、/workers、/users),防抖/loading/空态/重试全继承基座 |
| 已选回显 | 🔧 | **本次修复裸编号回显**:检索结果 label 增量缓存(rememberOptionLabels),页面重hydrate/缓存漏项时走详情接口兜底钉选(echoPinFromCache+fetchDetail),触发器展示「姓名 · 手机号/工号」;详情实体与已选值不一致时 console.warn 留痕(vitest 回归) |
| 清空 | 🔧 | **三选择器开启 clearable**,clearLabel=pages.pickers.common.clear;清空即 onChange('') |
| 附加能力 | ✅ | 详情抽屉 + 前往管理页入口(既有) |
| ⚠️ 遗留 | 组件侧 | UserPicker 全站零页面消费(201 全量 grep),保留备用但记入盘点;详情抽屉内 worker 班组/区域、customer 区域兜底仍显示 `#id` 裸编号——详情接口未返回名称,需 API 侧补字段后由 detailItems 消化(Wave 后续批次) |

### 5. RegionCascadePicker(区域级联)

| 维度 | 状态 | 说明 |
|---|---|---|
| 打开即用 | ✅ | 懒加载下钻 + 每级本地搜索 + 关键字直搜(防抖 300ms)+ 默认国家兜底 PH;列 loading/空态齐备 |
| 回显/清除 | 🔧 | **本次发现并修复编辑回显必然落空的双重缺陷**(2c9ecd0a):① countryCode 显式传参时 country 初值即命中,boot 的 setCountry 同值 bailout,回显效应永不重放(booted 由 ref 改 state);② 效应先写 expandedFor(state)再异步反查,标记写入触发重渲染,alive 清理先于 promise 兑现把在途 resolvePath 自杀(expandedFor 改 ref)。地址页实证:修复前重开回显为空,修复后 `PH / PH-1300000000` 链回显+清除可用;jsdom 三路径回归锁定。直搜点选回显链 ✅(102 DOM 断言 `PH / … / PH-0304911022`) |
| 错误 | ✅ | role=alert + 重试(reloadTick);层列失败不缓存父码可重拉 |
| 一致性 | ✅ | 文案 regionCascade.* 三语;shell 令牌双主题 |
| ⚠️ 遗留 | 组件侧 | 直搜失败的「重试」重跑国家+一级加载,不单独重发直搜请求(低频,影响小);组件 250 行接近红线,后续加功能需先拆列组件 |

### 6. ResourcePicker(兼容层,约 16 处直用页面)

| 维度 | 状态 | 说明 |
|---|---|---|
| 行为继承 | ✅ | load/search 双模式全部转发 SimplePicker,防抖/loading/空态/重试/键位自动继承 |
| 清空 | 🔧 | **clearable 加法默认开启**(clearable={false} 可退出),clearLabel 缺省取 pages.pickers.common.clear——16 处直用页面零改动获得一键清空 |
| 已选回显 | ⚠️ 页面侧 | 支持 pinnedOptions 但 16 处直用页仅 OrderCreateDrawer 传入;其余页面刷新/重开时触发器按基座兜底回显裸 value。页面批次触及各页时应把「已选 label」经 pinnedOptions 传入(页面自身持有 label 来源) |

### 7. 相邻基座参照(未在本波范围,记录口径)

- MultiSelect:勾选集合语义,不做 label 兜底(W1 裁定);禁用时 chips 移除钮隐藏 ✅。
- Dropdown 被顶栏/页面直用的裸调用(无 emptyText):行为不变(向后兼容 prop 设计)✅。

## 二、页面侧问题清单(只记录,Wave 1-3 逐页消化)

1. **oss/odn/coverage.tsx**:`clearLabel="×"` 裸字面量当 aria 文案,违反 i18n 红线;应改 pages.pickers.common.clear。
2. **ResourcePicker 直用页 15/16 未传 pinnedOptions**(provision/provlog、ams/replace、quad/scanlog、boss/dispatch、boss/message、boss/dismantle、boss/worker、TeamDialogs、worker-registration、oss/transfer、oss/reserve、alarm、billing/stopsrv、billing/billing、billing/invoices):筛选值从 URL/localStorage 重hydrate 后触发器回显裸编号;各页触及时应传 pinnedOptions 或迁移 SimplePicker 自管回显。
3. **DialogPicker 两处消费(boss/release、oss/odn/MaterialIssuesCard)**:重开不回显上次选择;现基座已支持 initialItems,页面触及时应把上次 onPick 结果回传。
4. **detailItems.ts 裸编号**:worker 班组 `#groupId`/区域 `#regionId`、customer 区域兜底 `#regionId`——需详情接口补名称字段(API 侧)后消化。
5. **UserPicker 零消费**:确认是否为规划中能力;若废弃随清理批次删除。
6. **基座遗留**(见矩阵 ①②):Dropdown onOpenChange、onDark 常量色——建议下一轮基座小批次处理,不阻塞页面波次。

## 三、本次修复的验证锚点

- vitest:pickerCore.test.ts 新增 echoPinFromCache / rememberOptionLabels / pickKeysFromItems 三组回归 + RegionCascadePicker.test.tsx jsdom 渲染回归(编辑回显显式国家/默认国家端点/空值三路径);全量 84 文件 545 用例绿(含 i18n 三语键集一致性)。
- 门禁:`TZ=Asia/Shanghai pnpm typecheck && pnpm test && pnpm build` 全绿(worktree feat/ux-selector)。
- 真实页面 DOM 断言(本地 dev 分支构建 + 102 后端真实数据):
  - 缴费登记页 CustomerPicker:开浮层→防抖检索 348→250 条→键盘/选中→触发器回显「OWPAL45451 · 09990000001」→重开焦点入检索框→Esc 关闭焦点归还→清除钮回占位;--logs 0 console error / 0 失败请求。
  - 地址页 RegionCascadePicker:直搜 Cavite 7 条→点选回显五级链 `PH / … / PH-0304911022`→清除复位;修复前重开回显为空、修复后 `PH / PH-1300000000` 链回显(对照证据)。
- 102:5180 部署态抽验(main 已并入门集 e7d1fb0d..a39efe8f 后 CI 重建):客户选择器检索/人读回显/清除钮全部在真实部署生效;cascade 修复(2c9ecd0a)待负责人二次合并后生效。
- 双主题浮层计算样式断言(缴费页浮层):light bg rgb(255,255,255)/边 rgb(239,241,245)/文字 rgb(78,86,100);dark bg rgb(16,32,63)/边 rgba(255,255,255,0.06)/文字 rgb(166,177,195)——与 tokens.css 两主题令牌逐一对应;截图 /tmp/selector-theme-light.png、/tmp/selector-theme-dark.png 及各页过程截图 /tmp/selector-*.png。
