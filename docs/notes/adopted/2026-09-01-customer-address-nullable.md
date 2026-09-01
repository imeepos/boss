# 裁定:customers.address_id 放开可空——先建档后补地址(000176)

日期:2026-09-01 ｜ 状态:Adopted

## 背景

客户档案页「新建客户」弹出的却是「代客开户」抽屉,强制先选装机地址;而建址两侧入口
(POST /orders/address 内联建址、POST /user-addresses)都要求 customerId 先存在,
受理起步即死锁:开户要地址 → 建址要客户 → 客户建不出来。

## 裁定

1. 迁移 000176:customers.address_id DROP NOT NULL;读路径统一 COALESCE(address_id,0),
   API 语义 addressId=0 即「未登记」。down 迁移保留(有 NULL 存量时按预期失败,先归位再执行)。
2. POST /customers 的 addressId 从 required 改为可选(负数拒绝),运营主体+经营区域仍必填
   ——数据范围(List 按 entity+region 过滤)依赖二者,缺省会造出「建档人自己看不见」的客户,
   比多选两个下拉更危险,故不放开。
3. 客户档案页动作链改为三步:新建客户(轻量)→ 行内「地址」(AddressChainDrawer 复用开单
   内联建址,backfill=true 直接回填档案)→ 行内「开户」直达开户工作台(?customerId= 预选),
   代客开户字样只保留在工作台语境。

## 放弃了什么

- 建档占位地址(造「待落位」假地址行):污染地址库权威树,治理成本高于可空列。
- 放开 legal_entity_id/region_id 可空:破坏数据范围过滤,产生不可见客户,否决。
- 新开 POST /customers/:id/address 端点:与 POST /orders/address 语义重复,复用既有端点
  (menu:order 门禁=能开单就能建址,受理角色天然满足)。
