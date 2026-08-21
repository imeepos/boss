# 选择器对接审计（2026-08-21）

> **状态:清单 1-10 已全部落地(提交 b89b873);#11 payment 账单号保留手填(账单号语义强)。**
> 真实 DOM 验证:billing 客户选择器(选中→trigger 显示→GET /bills?customerId=213 网络断言)、
> provlog 任务选择器(本地关键字过滤)均通过;其余页面同构替换。

> 目标:所有"手填实体 ID"的页面一律换选择器。基座组件 `web/admin/src/components/pickers/`
> (EntityPicker:服务端 keyword 检索 + DetailDrawer 详情 + 前往管理页);用户/师傅/客户三域
> 已有现成封装,其余域用 ResourcePicker(load/search 模式)。

## 一、已对接(无需处理)

| 页面 | 选择对象 |
|---|---|
| ams/replace 设备更换单 | 订单/资产/端口(ResourcePicker) |
| boss/dismantle 拆机管理 | 订单/资产/端口(ResourcePicker) |
| oss/transfer 跨区域调配 | 资源(ResourcePicker) |
| boss/dispatch 派单(抽屉) | 师傅(富 WorkerPicker,含评分/在途) |
| ams/stock、boss/worker | 班组(Dropdown,数据源 /worker-groups) |

## 二、需要对接(手填 ID,按优先级)

| # | 页面:行 | 手填字段 | 实体 | 数据源(已存在) | 建议组件 |
|---|---|---|---|---|---|
| 1 | boss/dispatch/index.tsx:175 | 指派/转派师傅 ID | 师傅 | /workers?keyword | 现有富 WorkerPicker 已在下方,删手填框或收紧为兜底 |
| 2 | boss/dispatch/index.tsx:99 | 列表筛选师傅 ID | 师傅 | /workers?keyword | WorkerPicker(筛选版) |
| 3 | boss/message/WorkerMessagesTab.tsx:91,98 | 查询/发送师傅 ID | 师傅 | /workers?keyword | WorkerPicker |
| 4 | billing/billing/index.tsx:44、invoices.tsx:57 | 筛选客户 ID | 客户 | /customers?keyword | CustomerPicker |
| 5 | billing/stopsrv/index.tsx:55 | 筛选客户 ID | 客户 | /customers?keyword | CustomerPicker |
| 6 | quad/scanlog/index.tsx:37 | 筛选订单 ID | 订单 | /orders | ResourcePicker(本地过滤够用) |
| 7 | oss/reserve/index.tsx:54 | 筛选端口 ID | 端口 | /ports | ResourcePicker |
| 8 | alarm/index.tsx:75 | 筛选资源 ID | 资源 | /resources | ResourcePicker |
| 9 | provision/provlog/index.tsx:42 | 筛选任务 ID | 下发任务 | /provision-tasks | ResourcePicker |
| 10 | boss/worker-registration/index.tsx:142,144 | 审核改派班组/区域 ID | 班组/区域 | /worker-groups、/regions | ResourcePicker×2 |
| 11 | billing/payment/index.tsx:40 | 筛选账单 ID | 账单 | /bills | 可选:账单号语义强,保留手填亦可 |

## 三、暂不处理(说明理由)

- intel/gis parentId/resourceId:GIS 资源树调试性质,手填 ID 即语义本身
- base/geo geonameId:外部标准数据(Geonames),无站内检索源
- base/smsconfig senderId、realidconfig accessKeyId/testIdNo:第三方平台凭证字段,非站内实体
- oss/expand expectedPorts:数字量,非实体 ID

## 权限注意

/workers 门禁 menu:dispatch、/customers 门禁 menu:customer、/users 门禁 menu:user——
选择器跨页复用时,当前角色缺对应菜单权限会 403(ResourcePicker 就地降级提示,不阻塞表单)。
