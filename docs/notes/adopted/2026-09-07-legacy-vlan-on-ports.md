# 存量开户数据字段归属裁定（VLAN 挂端口 / contract_months 挂 LO 账号 / legacy_path 承接 ODB 层级）

- 日期：2026-09-07
- 决策：
  1. 线路 VLAN 四元组（外层 svlan/内层 cvlan/internet_cvlan/tr069_cvlan）落 ports（迁移 000202，全部 SMALLINT 可空）。VLAN 属物理线路属性而非订购属性：换端口重装时 VLAN 随新端口重新生效，TL1 下发取数链（provision_tl1 resolver 读端口侧）无需跨表。
  2. OCC/ODB/OBD/Port 四级光缆段层级以 ports.legacy_path 单列 TEXT 无损承接（格式 OCC06/ODB040/OBD01/P05）。Excel 三位编码不合 ODN 五位编码规范（Suniway 规范红线），不入 odn_facility；ODN 正式建模列 P2 待办。
  3. lo_accounts 加 contract_months SMALLINT 可空承接存量"月数"（orders.buy_months 只服务订单态，存量不建历史订单）。
  4. 存量导入缺省归属：legal_entity=平台总公司、region=0（不限）、地址=单一占位节点（needs_review=true，治理队列消化），ports.address_id NOT NULL 约束由占位节点满足。
  5. 不为 347 条存量开户建历史订单；拆机 1 笔以 lo_accounts.status=CLOSED 表达。历史订单档案化（orders DONE 直插）列 P2 待办，含 buy_months 回填。

- why：VLAN 挂端口使"线路=端口=PON 定位=VLAN"同表可查，避免账号-端口二跳 join；legacy_path 以最小代价保住光纤层级可追溯性，不与 ODN 编码契约冲突；存量数据以"在服档案"口径入账（lo_accounts），不伪造订单状态机轨迹。
- 放弃了什么（被否决项）：VLAN 挂 lo_accounts（换端口语义错）；OCC/ODB 拆成 4 列（污染 ports 主档）；借用 odn_facility 强行编码（违反 Suniway 编码规范）；全量直插历史 DONE 订单（污染订单列表与统计，且 Excel 无地址无法满足订单归属推导）。
- 关联：docs/design/2026-09-07-legacy-kaihu-import.md；000178 PON 四维列；2026-09-06-odn-business-linkage；2026-08-21 business-timezone；2026-09-03-import-task-idempotency。
