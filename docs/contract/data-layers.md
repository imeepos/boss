# 数据分层依赖模型（contract/data-layers）

> 版本 V1.0（2026-08-17）｜依据：fields.md、data-relations.md + mock 数据实测审计（33 处矛盾，见附录）
> 定位：回答"哪些是基础数据、谁依赖谁"。构建模拟数据、写 seed、定初始化顺序一律按本层序。

## 0. 判定规则

1. **基础数据(L0)**：不依赖任何其他业务数据即可直接增删改查（如地理位置、角色、产品资费）。
2. 上层只能引用**更低层**的 id，禁止同层互引、禁止逆向引用（流水引用单据可以，单据引用流水不行）。
3. 每层可独立向下测试：删掉 L2+ 重造，L0/L1 不动。
4. **企业锚点**：归属到企业的业务事实/主单冗余 `legal_entity_id` + `legal_entity_name` 快照（跨企业对比锚点）；集团共享基础数据(L0)与纯时间轴子记录不冗余。

## 1. 分层总表

### L0 基础数据（零依赖，真正集团共享且语义一致）

| 实体 | 主键 | 说明 |
|:-----|:-----|:-----|
| regions | id, path(LTREE) | 1集团/2大区/3省/4城市 |
| addresses | id, path(LTREE) | 1市/2区/3街道/4小区/5楼栋（客观地理事实，多公司共享） |
| legal_entities | id, code | 运营主体（与 regions 同层软关联 crossRegionIds=经营区域） |
| roles / permissions | id/code | 权限体系（集团统一） |
| biz_params | key | 全局参数 |

### L1 依赖运营主体（公司自定义数据）

| 实体 | 依赖 | 说明 |
|:-----|:-----|:-----|
| product_offers | ▲legal_entity + name(必填) + bandwidth + 基础价 | 各公司自己的产品(名称/带宽/价格全属公司) |
| worker_groups | ▲legal_entity | 班组（运营主体自定义组织数据） |
| provision_templates / qos_templates | ▲legal_entity | 下发/服务模板（各公司设备与策略不同） |
| resources(OLT/分光器) | ▲legal_entity ▲address(楼栋/小区) ▲parent(树内上下级) | 各公司建设的网络设备树 |

### L2 依赖 L0/L1

| 实体 | 依赖 | 说明 |
|:-----|:-----|:-----|
| workers | ▲worker_group ▲region(须落在班组公司经营区域) | 师傅 |
| departments | ▲legal_entity | 部门 |
| region_offers | ▲offer ▲region_path + name?(区域名)+monthlyFee | 区域运营包(名称/价格均可覆盖;展示名回退: 区域名→公司名) |
| tags | ▲legal_entity, boundAssetId 留空 | 标签池（公司库存，预绑定后才指向资产） |
| asset_batches | ▲legal_entity | 入库批次（公司采购行为） |

### L3 依赖 L0~L2

| 实体 | 依赖 | 说明 |
|:-----|:-----|:-----|
| posts | ▲dept, ◆roles(post_roles) | 岗位 |
| accounts | ▲role ▲legal_entity? ▲dept? ▲post? ▲region_scope | 员工账号（权限主体） |
| customers | ▲legal_entity(归属公司) ▲address(必须楼栋级) | 客户档案 |

### L4 业务主单（依赖主体+资源）

| 实体 | 依赖 | 说明 |
|:-----|:-----|:-----|
| assets | ▲batch ▲tag? ▲address(部署位置) | 资产台账 |
| lo_accounts | ▲customer ▲offer ▲qos_template | 认证账号（客户 1:1） |
| orders | ▲customer ▲offer(产品目录) ▲address ▲region_path + price_snapshot | 订单（12环节）；成交价快照，调价不影响历史 |
| expansions | ▲legal_entity ▲region | 扩容单（公司行为） |
| transfers | ▲resource ▲from_region ▲to_region | 调拨单 |

### L5 单据派生与流水（只读/后置）

| 实体 | 依赖 | 说明 |
|:-----|:-----|:-----|
| order_stages | ▲order | 12 环节时间轴 |
| dispatch_tickets | ▲order ▲worker | 派单工单（订单1:1） |
| reserves | ▲port(L4 ports 属资源 L1 子集) ▲order | 端口预占 |
| port_change_history | ▲port ▲order? | 端口状态变更历史(状态/占用时间轴) |
| quad_links | ▲asset ▲customer ▲port ▲address | 四码合一（**第二码=客户，非LOID**） |
| scan_logs | ▲order ▲worker ▲tag | 扫码绑定记录 |
| activation_callbacks | ▲order | 激活回调 |
| dismantles | ▲order ▲asset ▲port | 拆机单 |
| complaints | ▲customer | 报障工单 |
| bills | ▲customer ×账期 | 账单 |
| payments | ▲bill | 缴费流水 |
| arrears | ▲customer | 欠费态 |
| stop_resume_tasks | ▲customer ▲lo_account | 停复机任务 |
| provision_tasks/logs | ▲lo_account ▲template | 配置下发 |
| audit_logs | ▲account ▲target | 审计（分区） |
| worker_group_memberships | ▲worker ▲group | 班组归属台账(换班组只新增不覆盖,快照 group_name/legal_entity_id) |
| account_org_histories | ▲account ▲legal_entity? ▲dept? ▲post? | 账号组织归属台账(调岗/调部门/调公司) |
| customer_histories | ▲customer ▲legal_entity ▲address | 客户归属台账(转品牌/搬家) |
| product_price_histories | ▲offer | 产品调价台账(旧价/新价/生效时间) |
| region_price_histories | ▲region_offer | 区域调价台账(旧价/新价/生效时间) |
| dispatch_transfers | ▲ticket ▲from_worker ▲to_worker | 派单改派台账 |
| resource_assignments | ▲resource ▲legal_entity ▲address? | 设备归属台账(调拨) |
| asset_assignments | ▲asset ▲worker? ▲address? | 资产持有台账(领用/部署/归还) |
| worker_performances/commissions/schedules/materials/tools | ▲worker ▲group(FK, +group_name 快照) | 师傅域派生;月度三表粒度=师傅×月×班组(月中调组拆多行) |
| worker_feedbacks | ▲worker ▲ticket ▲customer | 评价 |
| asset_returns | ▲worker ▲asset | 资产归还 |
| asset_lifecycles | ▲asset ▲address? | 资产状态轨迹(每次状态/位置变更,快照地址名) |
| replacements | ▲asset | 换新单 |
| stocktakes | ▲legal_entity ▲region/scope | 盘点任务（公司行为） |

## 2. 初始化顺序（seed 顺序）

```
L0 regions→addresses→legal_entities→roles/permissions→biz_params
L1 product_offers→worker_groups→templates→resources
L2 workers→departments→region_offers→tags→batches
L3 posts→accounts→customers
L4 assets→lo_accounts→orders→expansions/transfers
L5 stages/tickets/reserves/quad/scan/callback/dismantle/bills/payments/...
```

## 附录 A. 现有 mock 数据 33 处矛盾（实测，重构依据）

| 类别 | 数量 | 典型 |
|:-----|:-----|:-----|
| 孤儿外键 | 4 | ports/reserves/callbacks 引用不存在的订单 ORD-...009/011/021 |
| 四码口径错误 | 8 | quad.customerCode 全部用 LOID(认证账号)冒充客户，违反 fields.md 5.1 裁定 | ✅ adopted 2026-08-21 落 `customers.customer_code` 修正 |
| 端口码体系分裂 | 9 | quad/dismantles 用 P-SPLxx-xx 码，ports 表里根本没有 |
| 资产码混用 | 3 | dismantles.assetCode 用 EPC(标签码)当资产码；A-/EPC-/TAG- 三套码互串 |
| 弱引用 | 2 | orders/bills/lo_accounts 用客户姓名字符串做关联；customers 缺 addressId |
| 空值语义 | 5 | tags.boundAsset 未绑定时写 "—" 而非 null |
| 实体缺失 | 2 | workers.groupName 裸字符串无班组实体；quad.addrCode 自造码无地址实体对应 |

**修复口径**：全部改 numeric id 强引用 + code 仅作展示冗余（xxx_id + xxx_name 双列，fields.md 第 0 节规则）。
