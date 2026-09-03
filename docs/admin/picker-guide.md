# 选择器选用准则(web/admin)

> 落位:components/pickers/。红线:全站禁止新增原生 select,下拉一律走本体系(components/Dropdown.tsx 视觉基座)。

## 分界建议

| 场景 | 选用 | 判据 |
|---|---|---|
| 小数据量 | SimplePicker | 选项可数(经验阈值:数百条以内)、一屏可扫、单值表单字段、静态枚举或轻检索 |
| 大数据量 | DialogPicker | 千条级以上、需关键字+多维筛选、必须分页、需要单/多选批量圈定 |

经验法则:拿不准时先数数据量与筛选维度;两维以上筛选或要分页,直接 DialogPicker。

## SimplePicker(小数据量基座)

文件:components/pickers/SimplePicker.tsx。双数据源二选一:静态 options 数组(本地过滤)或 search 服务端关键字检索函数(防抖,优先级高)。

| prop | 必填 | 说明 |
|---|---|---|
| value / onChange | 是 | 受控值 |
| options | 二选一 | 静态全量选项(DropdownOption[]) |
| search | 二选一 | 服务端检索,返回选项列表;debounceMs 默认 300 |
| ariaLabel | 是 | 触发器 aria(调用方 i18n) |
| placeholder | 否 | 值为空时占位文案 |
| clearable / clearLabel | 否 | 展示清空按钮;clearLabel 为其 aria 文案 |
| disabled / minWidth | 否 | 禁用、触发器最小宽(默认 220) |
| emptyLabel / pinnedOptions | 否 | 空值选项;已选值不在结果内时钉选回显 |
| searchPlaceholder / errorText | 否 | 浮层过滤占位;加载失败就地提示 |

## DialogPicker(大数据量弹框基座)

文件:components/pickers/DialogPicker.tsx。受控 open/onPick/onClose;布局对齐附件选择器弹框(筛选区+列表区+底部确认);组件不绑定业务域。

| prop | 必填 | 说明 |
|---|---|---|
| open / onClose / onPick | 是 | 受控开关;确认回传选中实体数组 |
| mode | 否 | single(默认)/ multiple,multiple 支持跨页累选 |
| title / texts | 是 | 标题与结构文案(调用方 i18n;canonical 词条在 pages.pickers.dialog) |
| columns | 是 | DialogPickerColumn[]:key/title/render |
| query | 是 | 服务端分页查询:入参 keyword+filters+page/pageSize,返回 items+total |
| rowKey / rowLabel | 是 | 行键;已选回显文案 |
| filters | 是* | 多维搜索:至少一个可配置筛选项(key/label/options);options 需自带 value='' 的"全部"项 |
| initialPageSize | 否 | 默认 10 |

已选回显:底部 chips(单个移除 + 清空已选);确认按钮按选中数可用性控制。

## 接入指引

1. SimplePicker:表单字段处直接以 value/onChange 挂接;文案取页面 namespace 或 pages.pickers.common;需清空能力加 clearable + clearLabel。
2. DialogPicker:页面 state 控制 open;注入 columns(列头走页面 i18n)与 query(包装本域列表接口为 items/total 信封);filters 里给业务筛选项;确认回调拿实体数组自行落表单。
3. 文案登记:新文案进页面 namespace;结构性词条(确认/取消/分页等)复用 pages.pickers.dialog,勿在页面重复定义。
4. 详情/跳转能力(实体三选择器现有):继续用 UserPicker/WorkerPicker/CustomerPicker(基于 EntityPicker→SimplePicker),props 未变。

## 存量与演进

- ResourcePicker(components/ 根):服务端检索下拉的既有实现,约 20 个页面直用,保持不动;SimplePicker 服务端模式内部复用它。
- 新页面不要再直用 ResourcePicker:小数据量走 SimplePicker,大数据量走 DialogPicker。
- 关键词回归锚点:pickers/pickerCore.test.ts(双数据源合并、单/多选边界、分页选中、筛选组装)。
