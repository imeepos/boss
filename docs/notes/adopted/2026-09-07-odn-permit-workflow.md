# ROW 路权与 PECE 许可单工作流模型裁定

- 日期:2026-09-07
- 场景:P-INFRA-1 W4(审查 F3 阻塞提级):ROW/PECE 从资源链导入列文本升为独立工作流,施工开工挂许可前置。
- 需求源:docs/reviews/2026-09-07-invest-build-phase-design-review.md F3;数据基础 odn_resource_chain.row_status/pece_status(000210)。

## 决策

1. 新实体 odn_permits(000211)承载两类许可,kind 分域共用一表:ROW(路权)状态机 NOT_STARTED/PENDING/APPROVED/EXPIRED/NA;
   PECE 状态机 PENDING_SIGN/SIGNED/STAMPED/NA。字典与资源链导入列(000210)逐字对齐,不另造词。
2. 状态转移表(terms.md 4 权威):ROW 未开始-待处理-已批准(批复号+有效期止必填)/未开始(驳回,原因必填);
   已批准-已过期(手动或门控自动);已过期-待处理(过期复验重走批准);未开始与 NA 互转。
   PECE 待签署-已签署-已盖章;已签署-待签署(退回补正,原因必填);待签署与 NA 互转。驳回/补件路径如上,全部写操作入审计。
3. 关联面:project_id FK construction_projects(单号快照)、facility_code FK odn_facility(code)、chain_id 软引用 odn_resource_chain(无 FK);
   证照档案要素(批复号/管辖机构/有效期/附件引用 attachments BIGINT[])任何状态可补录(补件语义),状态与关联不经补录口。
4. 开工前置(F3):BOSS_ODN_PERMIT_GATE=on 灰度(默认关,与 BOSS_ODN_COVERAGE_GATE 同模式)时,项目 PENDING→BUILDING 前置校验
   每类存在达标许可或 NA;不满足 40900+缺失明细(ErrPermitRequired 透传);APPROVED 且 valid_until<当日自动回写 EXPIRED 并留
   [odn-permit] AUTO EXPIRED 可 grep 日志;另设 POST /odn/constructions/{id}/permits-check 干跑预检供前端/验收。
5. 单号 PM-YYYYMMDD-NNNNN(纳秒兜底,口径同结算单号);初始状态随 kind(ROW=未开始,PECE=待签署)。

## why

- 一表两域而非两表:两类许可的档案要素/关联/流转操作同构,kind CHECK 分域承载状态字典差异,导入口径与 chain 列对齐成本最低;
  独立权限码不新设(perm 复用 menu:odn),许可本就是 ODN 域能力,避免权限表迁移与菜单注册面扩大。
- 过期判定放门控数据面而非定时器:102 无秒级调度承诺,门控即时判定+自动回写保证「过期必阻断」无时间窗;手动标记过期保留给运营主动操作。
- NA 语义:全室内敷设等无路权场景必须能开工,否则门控会把存量合法项目全部堵死;NA 由人显式标记并审计,系统不猜。

## 放弃了什么(被否决项)

- 双表(row_permits/pece_permits):档案要素重复,列表页需 UNION,放弃。
- 审批流引擎(多级会签/节点配置):当前业务只有「批准/驳回」两步,引擎是过度设计;后续若需多级,在 transition 上加层即可,不改表。
- 定时任务扫描过期:见上,时间窗+调度依赖双输;复验(EXPIRED→PENDING)后必须重走批准,不复用旧批复号。
- 许可单挂资源链必填:导入链 row_status 已表达链级 ROW/PECE 状态,单据级关联做成可选软引用,不强绑。

## 关联

- 迁移 000211;契约 terms.md 4/fields.md 1.5.13/domain-map.md ODN 行;实现 internal/domain/odn/permit*.go、pg_construction_flow.go、internal/httpapi/admin/odn_permit.go。
- F6 竣工覆盖联动另见 adopted 2026-09-07-accept-coverage-linkage。