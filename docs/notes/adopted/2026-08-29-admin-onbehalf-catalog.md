# 裁定：代客受理目录端点跟随受理域本域权限（2026-08-29）

> 背景：目标"后台能够操作用户完成开户下单所有操作"。开户（POST /customers）、
> 代客下单（POST /orders）、注册审核（/customer-registrations）端点早已存在，
> 但受理表单的选项数据源被别的域菜单把着门——ops 凭本域权限填不完一张下单单。

## 事实（102 实测，2026-08-29 之前）

| 下单/开户表单需要的目录 | 唯一既有端点 | 门禁 | ops（受理角色）持有? |
|:--|:--|:--|:--|
| 渠道目录 | GET /provision/channels | menu:provision | ✗ |
| 经营区域 | GET /regions | menu:region | ✗ |
| 运营主体 | GET /legal-entities | menu:company | ✗ |
| 地址检索 | GET /addresses/search | menu:address | ✗ |
| 在售产品 | GET /products | menu:product | ✓（000169 读/写拆分后保留读） |

## 方案对比

1. **给 ops 补授 menu:address/region/company/provision**（迁移只加 grant）——否决：
   这些是整页菜单码，补授等于把地址树管理、组织编制、配置下发四个管理页开给客服，
   权限面膨胀远超"能填表"的需要，且 000169 刚做过反方向的收敛。
2. **前端 403 降级为手填 ID**——否决：把"操作员查不到 ID"转嫁给线下抄 ID，可用性伪满足。
3. **受理目录端点（采纳）**：新增两个只读端点，门禁跟随受理域自身权限——
   - `GET /orders/catalog`（menu:order）：在售产品（服务端过滤 PUBLISHED）+ 渠道；`q` 非空附地址检索。
   - `GET /customers/onboarding-catalog`（menu:customer）：经营区域 + 运营主体；`q` 非空附地址检索。
   域口全部复用既有 ListProducts/ListChannels/ListRegions/ListLegalEntities/SearchAddresses。

## 放弃了什么 / 代价

- 契约面 +2 端点（order.yaml / customer.yaml 已同步，bossctl 随契约重生成）；
  若走方案 1 则零契约变更。换来的是权限模型不破窗、ops 零新授权即可走完代客全程。
- 地址搜索在两个目录端点各暴露一次（同一 SearchAddresses 复用），非单点；可接受。

## 附带裁定：注册审核不做独立菜单页

注册审核抽屉挂在客户档案页（menu:customer）而非新建 /bss/registration 页：
门禁与页面可见性天然同码，免去"新页面 key→权限码登记→迁移→menu-sync 快照跨环境刷新"
的登记链；注册队列与客户档案同属开户语境，客服工作面不增页。
师傅注册审核（worker-reg 独立页 + menu:worker-reg 000139）是既有先例，若日后
客户注册量级需要独立队列页，按同一模式补即可。

## 验证（2026-08-29，102）

- make check 全绿（含契约对账 A-E、bossctl 路由一致性）+ web 三门禁（typecheck/test/build）。
- 102 全链路：目录 → 代客建址/建客户（customerId=346）→ 实名代提交+核验 PASS →
  代客下单（ORD-20260829-000581）→ 核查/预占/收费 → 自动段 5-8 → INSTALLING@stage8 →
  注册审核队列端点 200 → 地址检索命中 → 订单取消造数回收 → 孤儿巡检 OK。
- UI（cdp DOM 断言，102:5180）：客户页「新建客户/注册审核」、订单页「代客下单」入口在位；
  代客开户抽屉七字段齐全、运营主体下拉载入 12 主体；代客下单抽屉产品下拉载入 22 在售套餐。
