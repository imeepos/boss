// L2 师傅 + L2 班组归属台账 + L5 绩效/佣金/考勤/物料/工具/评价/归还 + L5 账务与审计。
import {
  Entity, PrimaryGeneratedColumn, Column, ManyToOne, JoinColumn, Index, OneToMany, OneToOne,
} from 'typeorm';
import type { BillStatus, PaymentStatus, TaskStatus } from '../enums.js';
import { WorkerGroup } from './org.js';
import { Customer } from './customer.js';
import { DispatchTicket } from './order.js';

@Entity('workers', { comment: 'L2师傅:归属班组(公司),服务区域须落在公司经营区域' })
@Index(['staffNo'], { unique: true })
export class Worker {
  @PrimaryGeneratedColumn({ comment: '主键' })
  workerId!: number;

  @Column({ type: 'varchar', length: 32, unique: true, comment: '工号,如WK-1024' })
  staffNo!: string;

  @Column({ type: 'varchar', length: 64, comment: '师傅姓名' })
  name!: string;

  @ManyToOne(() => WorkerGroup, { nullable: false })
  @JoinColumn({ name: 'group_id' })
  group!: WorkerGroup;

  /** 班组归属台账(历史:每个时间段属于哪个班组,换班组只新增不覆盖) */
  @OneToMany(() => WorkerGroupMembership, (m) => m.worker)
  memberships!: WorkerGroupMembership[];

  /** 担任组长的班组 */
  @OneToMany(() => WorkerGroup, (g) => g.leader)
  leaderOfGroups!: WorkerGroup[];

  @Column({ type: 'integer', comment: '服务区域id(regions表):须落在班组所属公司经营区域(应用层校验)' })
  regionId!: number;

  @Column({ type: 'varchar', length: 32, comment: '联系电话(脱敏存储)' })
  phone!: string;

  @Column({ type: 'smallint', default: 1, comment: '在职状态:1在职 0离职' })
  status!: number;

  @Column({ type: 'timestamptz', comment: '入职时间' })
  joinedAt!: Date;

  @Column({ type: 'timestamptz', nullable: true, comment: '离职时间;null=在职' })
  leftAt?: Date | null;

  /** 接单设置(师傅1:1) */
  @OneToOne(() => WorkerSettings, (s) => s.worker)
  settings!: WorkerSettings;

  /** 名下派单工单(下钻:这个师傅的数据) */
  @OneToMany(() => DispatchTicket, (t) => t.worker)
  tickets!: DispatchTicket[];

  /** 绩效记录 */
  @OneToMany(() => WorkerPerformance, (p) => p.worker)
  performances!: WorkerPerformance[];

  /** 佣金计提 */
  @OneToMany(() => WorkerCommission, (c) => c.worker)
  commissions!: WorkerCommission[];

  /** 考勤记录 */
  @OneToMany(() => WorkerSchedule, (s) => s.worker)
  schedules!: WorkerSchedule[];

  /** 物料领用 */
  @OneToMany(() => WorkerMaterial, (m) => m.worker)
  materials!: WorkerMaterial[];

  /** 工具借用 */
  @OneToMany(() => WorkerTool, (t) => t.worker)
  tools!: WorkerTool[];

  /** 收到的服务评价 */
  @OneToMany(() => WorkerFeedback, (f) => f.worker)
  feedbacks!: WorkerFeedback[];

  /** 资产归还记录 */
  @OneToMany(() => AssetReturn, (r) => r.worker)
  assetReturns!: AssetReturn[];

  /** 站内消息 */
  @OneToMany(() => WorkerMessage, (m) => m.worker)
  messages!: WorkerMessage[];
}

@Entity('worker_group_memberships', { comment: 'L2师傅班组归属台账:记录师傅每次归属班组的时间段,历史归属不随当前group变更' })
@Index(['workerId', 'effectiveFrom'])
@Index(['groupId'])
export class WorkerGroupMembership {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  /** 归属师傅 */
  @ManyToOne(() => Worker, (w) => w.memberships, { nullable: false })
  @JoinColumn({ name: 'worker_id' })
  worker!: Worker;

  /** 归属班组(仅作导航;名称/公司另存快照冻结) */
  @ManyToOne(() => WorkerGroup, (g) => g.memberships, { nullable: false })
  @JoinColumn({ name: 'group_id' })
  group!: WorkerGroup;

  @Column({ type: 'varchar', length: 64, comment: '班组名称快照:班组改名不影响历史台账' })
  groupName!: string;

  @Column({ type: 'bigint', comment: '班组所属公司id(legal_entities)快照:公司级历史归属不随班组换公司而变' })
  legalEntityId!: number;

  @Column({ type: 'varchar', length: 128, comment: '班组所属公司名快照:改名不改历史台账' })
  legalEntityName!: string;

  @Column({ type: 'integer', comment: '当时服务区域id(regions表)快照:师傅换区域不改历史台账' })
  regionId!: number;

  @Column({ type: 'varchar', length: 64, nullable: true, comment: '当时服务区域名快照:改名不改历史台账' })
  regionName?: string | null;

  @Column({ type: 'varchar', length: 128, nullable: true, comment: '调组原因,如新设班组/支援/晋升' })
  reason?: string | null;

  @Column({ type: 'bigint', nullable: true, comment: '操作人账号id(accounts表):谁执行了这次调组' })
  operatorAccountId?: number | null;

  @Column({ type: 'timestamptz', comment: '归属生效时间' })
  effectiveFrom!: Date;

  @Column({ type: 'timestamptz', nullable: true, comment: '归属结束时间;null=至今(当前归属)' })
  effectiveTo?: Date | null;
}

@Entity('worker_performances', { comment: 'L5师傅绩效:按月×班组×区域统计,一行=师傅该月在该班组该区域的一段贡献,月中调组/换区拆多行' })
@Index(['workerId', 'period', 'groupId', 'regionId'], { unique: true })
export class WorkerPerformance {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  /** 归属师傅 */
  @ManyToOne(() => Worker, (w) => w.performances)
  @JoinColumn({ name: 'worker_id' })
  worker!: Worker;

  @ManyToOne(() => WorkerGroup, (g) => g.performances, { nullable: false })
  @JoinColumn({ name: 'group_id' })
  group!: WorkerGroup;

  @Column({ type: 'varchar', length: 64, comment: '班组名快照:改名不改历史' })
  groupName!: string;

  @Column({ type: 'bigint', comment: '企业归属id(legal_entities)快照:冻结事发时企业,跨企业对比锚点' })
  legalEntityId!: number;

  @Column({ type: 'varchar', length: 128, comment: '企业名快照:改名不改历史' })
  legalEntityName!: string;

  @Column({ type: 'integer', comment: '事发时服务区域id(regions表)快照:师傅换区域不改历史' })
  regionId!: number;

  @Column({ type: 'varchar', length: 64, comment: '事发时服务区域名快照:改名不改历史' })
  regionName!: string;

  @Column({ type: 'varchar', length: 16, comment: '统计月份,如2026-08' })
  period!: string;

  @Column({ type: 'integer', comment: '该月该班组完工单量' })
  finished!: number;

  @Column({ type: 'smallint', comment: '准时率百分比,如98' })
  onTimeRate!: number;

  @Column({ type: 'numeric', precision: 3, scale: 1, comment: '综合评分,如4.9' })
  score!: number;
}

@Entity('worker_commissions', { comment: 'L5师傅佣金:按月×班组×区域计提,一行=师傅该月在该班组该区域的佣金,月中调组/换区拆多行' })
@Index(['workerId', 'period', 'groupId', 'regionId'], { unique: true })
export class WorkerCommission {
  @PrimaryGeneratedColumn()
  id!: number;

  /** 归属师傅 */
  @ManyToOne(() => Worker, (w) => w.commissions)
  @JoinColumn({ name: 'worker_id' })
  worker!: Worker;

  @ManyToOne(() => WorkerGroup, (g) => g.commissions, { nullable: false })
  @JoinColumn({ name: 'group_id' })
  group!: WorkerGroup;

  @Column({ type: 'varchar', length: 64, comment: '班组名快照:改名不改历史' })
  groupName!: string;

  @Column({ type: 'bigint', comment: '企业归属id(legal_entities)快照:冻结事发时企业,跨企业对比锚点' })
  legalEntityId!: number;

  @Column({ type: 'varchar', length: 128, comment: '企业名快照:改名不改历史' })
  legalEntityName!: string;

  @Column({ type: 'integer', comment: '事发时服务区域id(regions表)快照:师傅换区域不改历史' })
  regionId!: number;

  @Column({ type: 'varchar', length: 64, comment: '事发时服务区域名快照:改名不改历史' })
  regionName!: string;

  @Column({ type: 'varchar', length: 16, comment: '计提月份,如2026-08' })
  period!: string;

  @Column({ type: 'varchar', length: 64, comment: '计提公式,如新装×40' })
  formula!: string;

  @Column({ type: 'numeric', precision: 12, scale: 2, comment: '该月该班组佣金金额' })
  amount!: number;
}

@Entity('worker_schedules', { comment: 'L5师傅考勤:按月×班组×区域统计,一行=师傅该月在该班组该区域的忙日,月中调组/换区拆多行' })
@Index(['workerId', 'month', 'groupId', 'regionId'], { unique: true })
export class WorkerSchedule {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  /** 归属师傅 */
  @ManyToOne(() => Worker, (w) => w.schedules)
  @JoinColumn({ name: 'worker_id' })
  worker!: Worker;

  @ManyToOne(() => WorkerGroup, (g) => g.schedules, { nullable: false })
  @JoinColumn({ name: 'group_id' })
  group!: WorkerGroup;

  @Column({ type: 'varchar', length: 64, comment: '班组名快照:改名不改历史' })
  groupName!: string;

  @Column({ type: 'bigint', comment: '企业归属id(legal_entities)快照:冻结事发时企业,跨企业对比锚点' })
  legalEntityId!: number;

  @Column({ type: 'varchar', length: 128, comment: '企业名快照:改名不改历史' })
  legalEntityName!: string;

  @Column({ type: 'integer', comment: '事发时服务区域id(regions表)快照:师傅换区域不改历史' })
  regionId!: number;

  @Column({ type: 'varchar', length: 64, comment: '事发时服务区域名快照:改名不改历史' })
  regionName!: string;

  @Column({ type: 'varchar', length: 16, comment: '考勤月份,如2026-08' })
  month!: string;

  @Column({ type: 'smallint', comment: '该月该班组忙日天数' })
  busyDays!: number;
}

@Entity('worker_materials', { comment: 'L5物料领用:师傅领用耗材记录' })
export class WorkerMaterial {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  /** 领用师傅 */
  @ManyToOne(() => Worker, (w) => w.materials)
  @JoinColumn({ name: 'worker_id' })
  worker!: Worker;

  @ManyToOne(() => WorkerGroup, (g) => g.materials, { nullable: false })
  @JoinColumn({ name: 'group_id' })
  group!: WorkerGroup;

  @Column({ type: 'varchar', length: 64, comment: '班组名快照:改名不改历史' })
  groupName!: string;

  @Column({ type: 'bigint', comment: '企业归属id(legal_entities)快照:冻结事发时企业,跨企业对比锚点' })
  legalEntityId!: number;

  @Column({ type: 'varchar', length: 128, comment: '企业名快照:改名不改历史' })
  legalEntityName!: string;

  @Column({ type: 'integer', comment: '事发时服务区域id(regions表)快照:师傅换区域不改历史' })
  regionId!: number;

  @Column({ type: 'varchar', length: 64, comment: '事发时服务区域名快照:改名不改历史' })
  regionName!: string;

  @Column({ type: 'varchar', length: 64, comment: '物料名称,如光纤跳线' })
  name!: string;

  @Column({ type: 'integer', comment: '领用数量' })
  qty!: number;
}

@Entity('worker_tools', { comment: 'L5工具借用:师傅借用工具登记' })
export class WorkerTool {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  /** 借用师傅 */
  @ManyToOne(() => Worker, (w) => w.tools)
  @JoinColumn({ name: 'worker_id' })
  worker!: Worker;

  @ManyToOne(() => WorkerGroup, (g) => g.tools, { nullable: false })
  @JoinColumn({ name: 'group_id' })
  group!: WorkerGroup;

  @Column({ type: 'varchar', length: 64, comment: '班组名快照:改名不改历史' })
  groupName!: string;

  @Column({ type: 'bigint', comment: '企业归属id(legal_entities)快照:冻结事发时企业,跨企业对比锚点' })
  legalEntityId!: number;

  @Column({ type: 'varchar', length: 128, comment: '企业名快照:改名不改历史' })
  legalEntityName!: string;

  @Column({ type: 'integer', comment: '事发时服务区域id(regions表)快照:师傅换区域不改历史' })
  regionId!: number;

  @Column({ type: 'varchar', length: 64, comment: '事发时服务区域名快照:改名不改历史' })
  regionName!: string;

  @Column({ type: 'varchar', length: 64, comment: '工具名称,如熔纤机' })
  name!: string;

  @Column({ type: 'boolean', comment: '是否在借中' })
  borrowed!: boolean;
}

@Entity('worker_feedbacks', { comment: 'L5服务评价:客户对工单师傅的评价' })
export class WorkerFeedback {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  /** 被评师傅 */
  @ManyToOne(() => Worker, (w) => w.feedbacks)
  @JoinColumn({ name: 'worker_id' })
  worker!: Worker;

  @Column({ type: 'varchar', length: 64, comment: '被评师傅姓名快照:师傅改名不改历史评价' })
  workerName!: string;

  @ManyToOne(() => WorkerGroup, (g) => g.feedbacks, { nullable: false })
  @JoinColumn({ name: 'group_id' })
  group!: WorkerGroup;

  @Column({ type: 'varchar', length: 64, comment: '班组名快照:改名不改历史' })
  groupName!: string;

  @Column({ type: 'bigint', comment: '企业归属id(legal_entities)快照:冻结事发时企业,跨企业对比锚点' })
  legalEntityId!: number;

  @Column({ type: 'varchar', length: 128, comment: '企业名快照:改名不改历史' })
  legalEntityName!: string;

  @Column({ type: 'integer', comment: '事发时服务区域id(regions表)快照:师傅换区域不改历史' })
  regionId!: number;

  @Column({ type: 'varchar', length: 64, comment: '事发时服务区域名快照:改名不改历史' })
  regionName!: string;

  @Column({ type: 'bigint', comment: '关联派单工单id' })
  ticketId!: number;

  @Column({ type: 'bigint', comment: '评价客户id' })
  customerId!: number;

  @Column({ type: 'varchar', length: 64, comment: '评价客户姓名快照:客户改名不改历史评价' })
  customerName!: string;

  @Column({ type: 'smallint', comment: '评分1~5' })
  score!: number;

  @Column({ type: 'boolean', default: false, comment: '是否需要人工复核' })
  needReview!: boolean;
}

@Entity('asset_returns', { comment: 'L5资产归还:师傅完工回收资产入库' })
export class AssetReturn {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  /** 归还师傅 */
  @ManyToOne(() => Worker, (w) => w.assetReturns)
  @JoinColumn({ name: 'worker_id' })
  worker!: Worker;

  @ManyToOne(() => WorkerGroup, (g) => g.assetReturns, { nullable: false })
  @JoinColumn({ name: 'group_id' })
  group!: WorkerGroup;

  @Column({ type: 'varchar', length: 64, comment: '班组名快照:改名不改历史' })
  groupName!: string;

  @Column({ type: 'bigint', comment: '企业归属id(legal_entities)快照:冻结事发时企业,跨企业对比锚点' })
  legalEntityId!: number;

  @Column({ type: 'varchar', length: 128, comment: '企业名快照:改名不改历史' })
  legalEntityName!: string;

  @Column({ type: 'integer', comment: '事发时服务区域id(regions表)快照:师傅换区域不改历史' })
  regionId!: number;

  @Column({ type: 'varchar', length: 64, comment: '事发时服务区域名快照:改名不改历史' })
  regionName!: string;

  @Column({ type: 'bigint', comment: '归还资产id' })
  assetId!: number;

  @Column({ type: 'varchar', length: 128, comment: '归还原因,如完工回收' })
  reason!: string;

  @Column({ type: 'varchar', length: 16, default: 'PENDING', comment: '状态:PENDING待确认/DONE已入库' })
  status!: TaskStatus;
}

@Entity('bills', { comment: 'L5账单:客户×账期唯一,金额=订单成交价快照' })
@Index(['customerId', 'period'], { unique: true })
export class Bill {
  @PrimaryGeneratedColumn({ comment: '主键' })
  billId!: number;

  @Column({ type: 'varchar', length: 32, unique: true, comment: '账单号,如BILL-202608-201' })
  billNo!: string;

  @ManyToOne(() => Customer, { nullable: false })
  @JoinColumn({ name: 'customer_id' })
  customer!: Customer;

  @Column({ type: 'varchar', length: 64, comment: '客户姓名快照:客户改名不改历史账单' })
  customerName!: string;

  @Column({ type: 'bigint', comment: '企业归属id(legal_entities)快照:客户所属企业,跨企业对比锚点' })
  legalEntityId!: number;

  @Column({ type: 'varchar', length: 128, comment: '企业名快照:改名不改历史账单' })
  legalEntityName!: string;

  @Column({ type: 'integer', comment: '经营区域id(regions表)快照:账单时客户所在区域,按地区统计锚点' })
  regionId!: number;

  @Column({ type: 'varchar', length: 64, comment: '经营区域名快照:改名不改历史账单' })
  regionName!: string;

  @Column({ type: 'varchar', length: 16, comment: '账期,如2026-08(客户×账期唯一)' })
  period!: string;

  @Column({ type: 'numeric', precision: 12, scale: 2, comment: '账单金额:必须等于订单price_snapshot(应用层校验)' })
  amount!: number;

  @Column({ type: 'varchar', length: 16, comment: '状态:UNPAID未缴/PAID已缴/OVERDUE逾期' })
  status!: BillStatus;

  /** 该账单的缴费流水 */
  @OneToMany(() => Payment, (p) => p.bill)
  payments!: Payment[];
}

@Entity('payments', { comment: 'L5缴费流水:针对账单的收款记录' })
@Index(['payNo'], { unique: true })
export class Payment {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @Column({ type: 'varchar', length: 32, unique: true, comment: '缴费流水号,如PAY-20260820-001' })
  payNo!: string;

  @ManyToOne(() => Bill, { nullable: false })
  @JoinColumn({ name: 'bill_id' })
  bill!: Bill;

  @Column({ type: 'numeric', precision: 12, scale: 2, comment: '缴费金额' })
  amount!: number;

  @Column({ type: 'varchar', length: 16, comment: '缴费方式:微信/支付宝/现金/银行' })
  method!: string;

  @Column({ type: 'varchar', length: 16, comment: '状态:SUCCESS成功/FAILED失败/REFUNDED已退款' })
  status!: PaymentStatus;
}

@Entity('arrears', { comment: 'L5欠费态(应收信用域):客户1:1欠费快照' })
@Index(['customerId'], { unique: true })
export class Arrears {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @OneToOne(() => Customer, (c) => c.arrears)
  @JoinColumn({ name: 'customer_id' })
  customer!: Customer;

  @Column({ type: 'numeric', precision: 12, scale: 2, comment: '欠费金额' })
  amount!: number;

  @Column({ type: 'integer', comment: '欠费天数' })
  days!: number;

  @Column({ type: 'varchar', length: 16, comment: '状态:催收中/已停机等' })
  status!: string;
}

@Entity('stop_resume_tasks', { comment: 'L5停复机任务:欠费停机/缴费复机的执行流水' })
export class StopResumeTask {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @Column({ type: 'bigint', comment: '客户id' })
  customerId!: number;

  @Column({ type: 'bigint', comment: '目标LO账号id' })
  loAccountId!: number;

  @Column({ type: 'varchar', length: 8, comment: '动作:STOP停机/RESUME复机' })
  action!: string;

  @Column({ type: 'varchar', length: 16, comment: '状态:PENDING/DOING/DONE/FAILED' })
  status!: TaskStatus;
}

@Entity('audit_logs', { comment: 'L5审计日志(分区表):谁在何时对什么对象做了什么' })
export class AuditLog {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @Column({ type: 'bigint', comment: '操作人账号id(accounts表):弱引用,日志只读不FK,账号删除不影响日志' })
  accountId!: number;

  @Column({ type: 'varchar', length: 64, comment: '操作人姓名快照:账号改名不改历史日志' })
  accountName!: string;

  @Column({ type: 'varchar', length: 64, nullable: true, comment: '事发时部门名快照:调部门不改历史日志' })
  deptName?: string | null;

  @Column({ type: 'varchar', length: 128, nullable: true, comment: '事发时公司名快照:调公司不改历史日志' })
  legalEntityName?: string | null;

  @Column({ type: 'varchar', length: 32, comment: '操作类型:数据变更/状态变更/权限变更' })
  action!: string;

  @Column({ type: 'varchar', length: 32, comment: '目标对象类型,如order' })
  targetType!: string;

  @Column({ type: 'varchar', length: 64, comment: '目标对象id' })
  targetId!: string;

  @Column({ type: 'jsonb', default: {}, comment: '操作详情(变更前后值)' })
  detail!: object;

  @Column({ type: 'varchar', length: 64, comment: '来源IP' })
  ip!: string;
}

@Entity('worker_settings', { comment: 'L5师傅接单设置(师傅1:1):接单开关/半径/类型偏好' })
export class WorkerSettings {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  /** 师傅1:1 */
  @OneToOne(() => Worker, (w) => w.settings)
  @JoinColumn({ name: 'worker_id' })
  worker!: Worker;

  @Column({ type: 'boolean', default: true, comment: '是否接单中' })
  accepting!: boolean;

  @Column({ type: 'smallint', comment: '接单半径(公里)' })
  radiusKm!: number;

  @Column({ type: 'varchar', length: 255, comment: '接单类型偏好,如:新装宽带/宽带变更/拆机' })
  acceptTypes!: string;

  @Column({ type: 'timestamptz', nullable: true, comment: '设置最近更新时间' })
  updatedAt?: Date | null;
}

@Entity('worker_messages', { comment: 'L5师傅站内消息:系统/调度下发给师傅的通知' })
export class WorkerMessage {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  /** 接收师傅 */
  @ManyToOne(() => Worker, (w) => w.messages)
  @JoinColumn({ name: 'worker_id' })
  worker!: Worker;

  @Column({ type: 'varchar', length: 8, comment: '级别:INFO/WARN/URGENT' })
  level!: string;

  @Column({ type: 'varchar', length: 128, comment: '消息标题' })
  title!: string;

  @Column({ type: 'text', comment: '消息内容' })
  content!: string;

  @Column({ type: 'timestamptz', comment: '发送时间' })
  sentAt!: Date;

  @Column({ type: 'boolean', default: false, comment: '是否已读' })
  read!: boolean;
}
