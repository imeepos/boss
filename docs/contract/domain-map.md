# 能力域四套命名映射表（contract/domain-map）

> 版本 V1.0（2026-08-17）｜权威源：《BOSS综合业务支撑平台-需求全案》V1.2 第二/四/七篇
> 定位：AI 开发时消除"四套命名体系"歧义的唯一对齐表。任何 Agent 开工前必须先读本表，
> 确认自己实现的是"哪个能力域 / 落哪个 internal 包 / 对应哪些页面 / 引用哪些 REQ"。

## 0. 四套命名体系是什么

| # | 体系 | 出处 | 粒度 | 用途 |
|:-:|:-----|:-----|:-----|:-----|
| A | 能力域（21 域） | 全案 2.3/4.1 | 逻辑域 | 需求/规则/验收的权威边界 |
| B | Agent（AG-01~18 + XG-01~04 + 1 集成） | 全案 7.2 | 执行任务 | 开发排期/分包 |
| C | internal 包（17 域） | 当前 Go 骨架 `internal/domain/*` | 代码包 | 落地位置 |
| D | admin 页面分组（13 组 49 页） | `docs/admin/menu.js` | UI 分组 | 前端落地 |

> 关键事实：A/B 是"菲律宾 BOSS"权威体系（21 域），C/D 是"中国 9 阶段"视角的子集（17 域/13 组）。
> **C 与 D 尚未覆盖 A 的部分域**（如防欺诈/忠诚度/批发/结算互连/门户/消息/渠道/应收信用）。
> 本期冻结只对齐已存在部分，未落地的域标注"待建"。

## 1. 主映射表（以能力域 A 为行主键）

| 能力域(A) | 简称 | Agent(B) | internal 包(C) | 阶段(C) | admin 分组(D) | 关键页面(D) |
|:----------|:-----|:---------|:---------------|:--------|:--------------|:------------|
| 系统管理与安全 | SYS | AG-01 | `user` | 阶段1 | base + org + 部分 base | account/address/settings/audit/company/department/post/menuperm/datascope |
| 免登录 API key | — | — | `apikey` | 阶段1 | org（web/admin；docs/admin menu.js 未列） | apikey（权限 menu:apikey，绑定 account/worker/customer 三主体） |
| 客户关系管理 CRM | CRM | AG-02 | `customer` | 阶段2 | bss | customer |
| 产品与资费 | PROD | AG-02 | `customer`(产品部分) | 阶段2 | bss | product |
| 订单与销售 | ORD | AG-02 | `order` | 阶段5 | boss | order/dispatch/dismantle |
| 计费与账务 | BIL | AG-03 | `billing` | 阶段5 | billing | billing/payment/arrears/stopsrv/paycheck |
| 支付与收款 | PAY | AG-04 | `billing`(收款部分) | 阶段5 | billing | payment/paycheck |
| 发票与税务 | TAX | AG-04 | `billing`(TAX 部分) | 阶段5 | billing | （无专用页，入 billing；发票列表/作废入 billing.html） |
| 认证授权 AAA | AAA | AG-05 | `aaa` | 阶段7 | aaa | aaalog |
| 资产管理 AMS | AMS | AG-06 | `asset` | 阶段3 | ams | asset/tag/stock/replace |
| 网络资源 OSS | OSS | AG-06 | `resource` | 阶段4 | oss | resource/reserve/transfer/expand |
| 设备与 CPE | DEV | AG-08 | `device` | 阶段7 | oss | device |
| 网络监控告警 | MON | AG-08 | `device`(告警部分) | 阶段7 | alarm | alarm |
| 配置下发 | PROV | （并入 device/provision）| `provision` | 阶段7 | provision | provision/template/provlog |
| 四码合一/一致性 | QUAD | （横切 CONS） | `quadlink` | 阶段6 | quad | quadlink/check/scanlog |
| GIS | GIS | AG-11 | `gis` | 阶段8 | intel | gis |
| 国际地理基础数据 | — | — | `geo` | 阶段1 | base（web/admin `/base/geo`；docs/admin menu.js 未列） | geo（国家/行政区划/译名，ISO 3166；服务 addresses 国际化，与 gis 分立见 note 2026-08-18-geo-vs-gis-split） |
| 经营分析 BI | BI | AG-12 | `analytics`(+`report`) | 阶段9 | intel | analytics/report |
| 客服工单 | CS | AG-07 | 待建 | — | boss(报障) | complaint |
| 应收信用 | AR | AG-09 | 待建 | — | billing(欠费) | arrears |
| 渠道经销商 | CH | AG-10 | 待建 | — | 待建 | 待建 |
| 批发结算 | WHO | AG-13 | 待建 | — | 待建 | 待建 |
| 品牌区域 | BRAND | AG-14 | `user`(Region) | 阶段1 | org | region/company |
| 营收保障 RA | RA | AG-15 | 待建 | — | 待建 | 待建 |
| 防欺诈 FMS | FMS | AG-16 | 待建 | — | 待建 | 待建 |
| 结算互连 SET | SET | AG-17 | 待建 | — | 待建 | 待建 |
| 忠诚度积分 LOY | LOY | AG-18 | 待建 | — | 待建 | 待建 |
| 客户门户 | PORT | XG-04(组装) | 待建(前端) | — | user端(非admin) | — |
| 消息通知 | NOT | XG-04(组装) | `worker`(师傅侧消息/公告)、`notify`(admin 侧提醒/待办,迁移 000090) | 阶段2 | boss | message(后台提醒=第三页签);推送通道配置 push.*(pkg/push,迁移 000094,页面 /base/pushconfig) |
| AI 能力网关 | AI | （横切，平台级，非 21 域） | `ai` | 增量 | 无专用页（复用 base/settings 参数页） | ai.openai.* 配置经 /params 或 /ai/openai/config 热更 |

> 注：`internal/domain/user` 承担 A 体系的 `系统管理` + `品牌区域(Region 部分)` 两类职责；
> `internal/domain/billing` 承担 `计费账务` + `支付收款`；`internal/domain/device` 承担 `设备CPE` + `网络监控`。
> 这是"9 阶段"视角对"21 域"的合并，属历史架构决策，本期不强行拆分，但 Agent 落地时须按 A 域边界写清晰注释。

## 2. 反向索引（以 admin 分组 D 为行主键）

menu.js 共 13 分组 49 菜单页（另 `login.html` 为登录散页，不进菜单；web/admin 新前端 menu.def.ts 另含 geo/apikey/message 3 页）。表列"页面"为该分组全部 href。

| admin 分组(D) | 能力域(A) | internal 包(C) | 阶段(C) | 页面 |
|:--------------|:----------|:---------------|:--------|:-----|
| overview 运营总览 | BI(聚合) | `analytics` | 阶段9 | dashboard |
| base 基础配置 | SYS | `user` | 阶段1 | account/address/settings/audit/importer |
| org 组织与权限 | SYS + BRAND | `user` | 阶段1 | company/department/post/region/menuperm/datascope |
| bss 客户与资费 | CRM + PROD | `customer` | 阶段2 | customer/product/user/userdata |
| billing 计费与账务 | BIL + PAY + AR | `billing` | 阶段5 | billing/payment/arrears/stopsrv/paycheck |
| ams 资产与标签 | AMS | `asset` | 阶段3 | asset/tag/stock/replace |
| oss 网络资源 | OSS + DEV + AAA(认证账号) | `resource`/`device`/`aaa` | 阶段4/7 | resource/reserve/transfer/device/loaccount/expand |
| boss 订单与工单 | ORD + CS | `order` | 阶段5 | order/worker/worker-ops/dispatch/dismantle/complaint/callback |
| quad 四码合一 | QUAD | `quadlink` | 阶段6 | quadlink/check/scanlog |
| provision 配置下发 | PROV | `provision` | 阶段7 | provision/template/provlog |
| alarm 告警中心 | MON | `device` | 阶段7 | alarm |
| aaa 认证计费 | AAA | `aaa` | 阶段7 | aaalog |
| intel 数字孪生与经营 | GIS + BI | `gis`/`analytics` | 阶段8/9 | gis/analytics/report |

### 2.1 跨域归属的页面（边界标注，Agent 不得越界实现）

| 页面 | 挂靠分组(D) | 真实能力域(A) | 说明 |
|:-----|:-----------|:--------------|:-----|
| complaint.html | boss | CS(客服工单) | 报障工单，属客服域非订单域 |
| arrears.html | billing | AR(应收信用) | 欠费催收，属应收域非计费域 |
| loaccount.html | oss | AAA(认证授权) | 认证账号，属 AAA 域非资源域 |
| callback.html | boss | ORD(订单) | 激活回调，订单第 11 环节 |
| importer.html | base | 横切 | 数据导入中心，跨域公共能力 |

## 3. 阶段 vs 能力域 vs Agent 的落地顺序

需求全案的三期（收款闭环/服务闭环/经营闭环）与 internal 的 9 阶段是两套排期，不冲突：
9 阶段回答"代码何时写"，三期回答"验收何时过"。映射如下：

| 全案三期 | Agent | internal 阶段 | 依赖关系 |
|:---------|:------|:-------------|:---------|
| 一期 收款闭环 | XG + AG-01~05 + AG-14 | 阶段1~2、5 | AG-01 最先，其余依赖 |
| 二期 服务闭环 | AG-06~10/15/16/18 | 阶段3~4、6~7 | 依赖一期 |
| 三期 经营闭环 | AG-11~14/17 | 阶段8~9 | 依赖二期 |

## 4. 使用规则（写入每个 Agent 输入包）

1. 每个 Agent 开工第一行声明：`【能力域】<A 域> 【Agent】<B> 【internal 包】<C> 【阶段】<N> 【页面】<D 页面清单>`。
2. 禁止跨 A 域 import 他域 implementation（只经契约/事件，见全案 4.4 CT 契约）。
3. 字段、状态、术语一律以 `terms.md` 为准，页面列名与 struct 字段必须一一对齐。
4. 表 1 标注"待建"的域，本期不实现，Agent 不得臆造；对应能力域若有 REQ 引用，提交"待建域清单"而非空实现。
