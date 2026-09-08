# 选择器选用准则(web/admin)

> 落位:components/pickers/。红线:全站禁止新增原生 select,下拉一律走本体系(components/Dropdown.tsx 视觉基座)。
> W0(2026-09-07)起基座统一供给八项体验契约(见下),各页不得绕过。

## 全局体验契约(基座统一实现,页面零改动继承)

1. 服务端检索 loading 态与防抖全体系一致:统一防抖 300ms(pickerCore.PICKER_DEBOUNCE_MS),请求期间浮层顶部渲染 role=status 的 loading 行;竞态由 seq 状态机(pickerSearchReducer)丢弃过期响应。
2. 空结果空态文案:浮层内无匹配时展示空态行,缺省取 pages.pickers.common.empty(三语)。
3. 加载失败就地报错且可重试:触发器下方 role=alert 错误行 + 「重试」按钮(pages.pickers.common.retry);DialogPicker 错误行同款;RegionCascade 失败不缓存父码,可重拉。
4. 可空字段清空(clearable)行为一致:clearable + clearLabel 时,值非空且未禁用才出清空按钮(pickerCore.canClearValue)。
5. 键盘可达:箭头/Enter/Space 开合,上下移动跳过禁用项并循环回绕(pickerCore.moveActive),Enter/Space 选定/勾选,Esc/Tab 关闭;listbox/option 角色与 aria-activedescendant 齐全,开启即把焦点交给过滤输入。
6. 已选值不在当前结果集时回显不丢失:withPinnedValue 自动钉选合成选项(value 兼作 label);调用方可用 pinnedOptions 提供人类可读钉选(按 value 去重,先到先得)。
7. 禁用态语义清晰:触发器原生 disabled + 置灰 + cursor-not-allowed;禁用项 aria-disabled;禁用时清空按钮与 chips 移除钮隐藏。
8. 三语文案齐备:结构性词条归 pages.pickers 命名空间(common/dialog),append-only 独立小提交;组件内不硬编码文案。
9. 开合通知(W3):Dropdown 提供 onOpenChange(仅开合过渡触发,mount 不触发);SimplePicker 服务端源在 open=true 且关键字非空时自动复位检索(清关键字+重发首屏),重开浮层首屏与空输入框即时一致,不再出现「输入框已清而列表残留旧检索结果」窗口。

## 分界建议

| 场景 | 选用 | 判据 |
|---|---|---|
| 小数据量 | SimplePicker | 选项可数(经验阈值:数百条以内)、一屏可扫、单值表单字段、静态枚举或轻检索 |
| 大数据量 | DialogPicker | 千条级以上、需关键字+多维筛选、必须分页、需要单/多选批量圈定 |

经验法则:拿不准时先数数据量与筛选维度;两维以上筛选或要分页,直接 DialogPicker。

## 文案当 value 的存量调用防御(W1 裁定,2026-09-07)

W1 域审计发现至少 9 处调用点把显示文案当 value 传给 Dropdown(触发器跌回 ariaLabel,
102 实证出现「状态状态」)。基座统一防御,各域按以下口径在 Phase B 逐处修正:

- 精确 value 命中优先;文案恰为某 option 的 label 时按 label 同值兜底(resolveOptionMatch):
  勾选、键盘落点、高亮全按真实 value 项渲染,选定回写真实 value。
- 文案既不在 value 域也不在 label 域:合成钉选回显原值(withPinnedValue),不跌回占位文案。
- 两条兜底路径均 console.warn('[Dropdown] value 不在 options 值域:…') 每实例留痕一次,
  Phase B 改传真实 value 后告警自然消失;全站 grep 该前缀即可盘点存量调用点。
- MultiSelect 不做 label 兜底(多选勾选集合语义不同),values 传文案时按原值回显;
  新调用一律传真实 value,枚举以 docs/contract/fields.md 为准。

## SimplePicker(小数据量基座)

文件:components/pickers/SimplePicker.tsx。双数据源二选一:静态 options 数组(本地过滤)或 search 服务端关键字检索函数(防抖 300ms,优先级高,内部走 useServerPickerSearch)。

| prop | 必填 | 说明 |
|---|---|---|
| value / onChange | 是 | 受控值 |
| options | 二选一 | 静态全量选项(DropdownOption[]) |
| search | 二选一 | 服务端检索,返回选项列表;失败抛错或返回 null 由基座转错误态 |
| ariaLabel | 是 | 触发器 aria(调用方 i18n) |
| placeholder / searchPlaceholder / searchAriaLabel | 否 | 占位与检索输入 aria |
| clearable / clearLabel | 否 | 清空按钮及其 aria(行为口径见契约 4) |
| disabled / minWidth | 否 | 禁用、触发器最小宽(默认 220) |
| emptyLabel / pinnedOptions | 否 | 空值选项;显式钉选(已选值漏项由基座自动兜底) |
| errorText | 否 | 失败就地提示文案;缺省回退 ariaLabel |
| loadingText / emptyText / retryText | 否 | 覆盖结构性文案;缺省取 pages.pickers.common |

## DialogPicker(大数据量弹框基座)

文件:components/pickers/DialogPicker.tsx。受控 open/onPick/onClose;布局对齐附件选择器弹框(筛选区+列表区+底部确认);组件不绑定业务域。

| prop | 必填 | 说明 |
|---|---|---|
| open / onClose / onPick | 是 | 受控开关;确认回传选中实体数组 |
| mode | 否 | single(默认)/ multiple,multiple 支持跨页累选 |
| title / texts | 是 | 标题与结构文案(texts.retry 可省,缺省取 pages.pickers.common.retry) |
| columns | 是 | DialogPickerColumn[]:key/title/render |
| query | 是 | 服务端分页查询:入参 keyword+filters+page/pageSize,返回 items+total |
| rowKey / rowLabel | 是 | 行键;已选回显文案 |
| filters | 是* | 多维搜索:至少一个可配置筛选项(key/label/options);options 需自带 value='' 的「全部」项 |
| initialPageSize | 否 | 默认 10 |
| initialItems | 否 | 重开定位(W0-R4):上次 onPick 的实体数组,重开时预勾选并回显 chips,单选取首项;不传保持打开即空选 |

关键字检索防抖 300ms(与 SimplePicker 同常量);失败就地 role=alert + 重试;空态/加载态齐备;候选表行支持 Enter/Space 选中。已选回显:底部 chips(单个移除 + 清空已选);确认按钮按选中数控制可用性。

## EntityPicker(实体选择器,客户/师傅/用户)

文件:components/pickers/EntityPicker.tsx;实体三选择器(CustomerPicker/WorkerPicker/UserPicker)即注入各自域接口的实例。

- 服务端 keyword 检索 + 详情抽屉 + 前往管理页入口;SimplePicker 全部体验契约自动继承。
- **已选人类可读回显(W0-R3)**:检索结果 label 增量缓存,重hydrate/缓存漏项时自动走详情接口兜底钉选;页面无需再传 pinnedOptions,触发器恒为「姓名 · 手机号/工号」而非裸 ID。
- **清空(W0-R4)**:三选择器内置 clearable,clearLabel=pages.pickers.common.clear;清空即 onChange('')。

## ResourcePicker(兼容层,勿新增直用)

文件:components/ResourcePicker.tsx。约 20 处存量页面直用,props 契约原样保留(load/search + toOption),内部已全部转发 SimplePicker:

- search 模式 → SimplePicker 服务端模式(toOption 映射后透传),防抖/loading/重试/回显全继承;
- load 模式 → 一次性拉取 + 静态源本地过滤,失败就地 errorText + 重试;
- **clearable(W0-R4)默认开启**,clearLabel 缺省取 pages.pickers.common.clear;传 clearable={false} 可退出;
- 新旧边界:新页面禁止再直用 ResourcePicker——小数据量 SimplePicker,大数据量 DialogPicker;存量直用页在 W1-W3 波次触及时顺势迁移,不迁移也不阻碍契约继承(props 不变,行为升级);
- 回显提示:检索结果内选中的值由基座回显 label;从 URL/localStorage 重hydrate 的值若不在结果集内,仍按基座兜底回显裸 value——页面应经 pinnedOptions 提供人类可读文案(页面自身持有 label 来源)。
- 实体三选择器(UserPicker/WorkerPicker/CustomerPicker)基于 EntityPicker→SimplePicker,props 未变,另带「详情/前往管理页」入口。

## 接入指引

1. SimplePicker:表单字段处直接以 value/onChange 挂接;文案取页面 namespace 或 pages.pickers.common;需清空能力加 clearable + clearLabel。
2. DialogPicker:页面 state 控制 open;注入 columns(列头走页面 i18n)与 query(包装本域列表接口为 items/total 信封);filters 里给业务筛选项;确认回调拿实体数组自行落表单。
3. 文案登记:新文案进页面 namespace;结构性词条(loading/empty/retry/确认/取消/分页等)复用 pages.pickers,勿在页面重复定义,词条变更走 append-only 独立小提交。
4. 服务端检索函数:入参 keyword(空串=首屏全量),返回选项/列表数组,失败抛错或返回 null。

## 关键词回归锚点

- pickers/pickerCore.test.ts:双数据源合并、单/多选边界、键盘 moveActive、钉选回显、清空口径、服务端检索状态机(seq 防竞态/失败重试)、已选人类可读回显缓存(echoPinFromCache/rememberOptionLabels)、重开预选序列(pickKeysFromItems)。
- pickers/SimplePicker.test.tsx:Dropdown onOpenChange 开合过渡矩阵(点击/键盘开、Esc/外点关,mount 静默)+ 服务端源重开复位回归(重开即重发首屏检索且输入框已清)。
- i18n/locales/keys.test.ts:三语键集一致性(pickers.common.loading/empty/retry 等)。
- 选择器族全景审计(现状/修复/遗留矩阵 + 页面侧问题清单):docs/admin/selector-audit.md。
