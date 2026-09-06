# ODN 无源网络升级为业务基础（覆盖关联→端口占用→逻辑物理绑定）

日期：2026-09-06

## 决策

ODN 无源物理层（000075/78/79/81，八表）从"登记型孤岛"升级为业务流程的基础数据，分三阶段推进：

1. **P1 覆盖关联**：新增 address_coverage 关联（地址 → 服务设施/设备 + 可装状态），提供可装性查询接口与 admin 展示；本期只"可查可判"，不做下单硬校验。
2. **P2 物理端口占用态**：ODN 设备端口状态机（IDLE/RESERVED/IN_SERVICE），订单预占优先在覆盖设施内分配物理口。
3. **P3 逻辑-物理绑定**：激活时写逻辑端口 ↔ ODN 设备/端口绑定，GIS 点位反查在用订单，报障按物理拓扑算影响面。

编码命名空间裁定不变：E10/E11 的"ODN 规范码独立命名空间"继续有效，桥接靠**关系表**（coverage/binding），不改双方编码，不做外键直穿业务主单。

## why

- 用户裁定（2026-09-06 会话）："应该是业务关联的，网络规划应该是基础"——即电信 OSS/BSS 标准的资源驱动开通模型：地址覆盖判定 → 订单 → 预占物理口 → 装机 → 激活绑定。
- 现状证据（102 实测 + 代码扫描）：odn 八表中业务主体五表（grid/facility/cable_segment/fiber/device）零行；order/provision/resource 三域对 "odn" 零引用；ODN 与核心业务链无任何 FK（契约 data-relations §2.12 单向挂 geo_subdivision）。
- 两套体系各自成熟但无桥：逻辑资源层已有订单预占端口完整机制（ports RESERVED/order_id、reserve_records、PON 端到端链路反查 /ports/:id/path，账本 P5-W2 已验收），ODN 物理层有规范编码全量台账模型 + GIS 图层——缺的只是中间的覆盖判定与绑定关系。
- 登记型能力空转无业务价值兑现路径：NRM 对标（nrm-benchmark-gap-analysis C1/C2）与三年路线图的覆盖分析/容量预警/投资分析都以"物理台账被业务消费"为前提。

## 放弃了什么（被否决项）

- **订单直接绑 ODN、废弃逻辑资源层**：改造面覆盖下单/预占/派单/激活全链，风险不可控；否决，改为桥接后逐步收敛。
- **odn 并入 resource 域统一建模**：推翻 E10/E11 编码命名空间裁定，牵动已对账的甲方规范；否决。
- **P1 即做下单硬校验**：覆盖数据未录入时全量拒单会阻断现网流程；否决，P1 只读展示，硬校验随 P2 端口状态机一起上。
- **ODN 点位并入 GIS 八级 drill**：2026-08-25 已裁定独立图层（level 9~11 正交），维持不动；P3 只给点位补"点击反查订单"，不改层级语义。

## 关联

- 契约：docs/contract/data-relations.md §2.12、docs/contract/fields.md §1.5、docs/contract/alignment-audit.md §10（E4-E11）
- 前置决策：adopted/2026-08-25-odn-gis-coords-link.md（GIS 图层关联）、adopted/2026-08-28-port-reserved-installing-inflight.md（端口预占状态机，P2 复用）
- 对标：docs/research/nrm-benchmark-gap-analysis.md（C1/C2/C4）
- 开发计划：docs/plan/odn-business-linkage-p1-plan.md（P1 任务账本 T4-T7）
- 巡检底稿：docs/review/orphan-tables-2026-09-05.md（odn 两码表"留存未用"标注）
