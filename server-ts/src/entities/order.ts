// L4 订单 + L5 环节轴/派单/扫码/回调/拆机/报障。
import {
  Entity, PrimaryGeneratedColumn, Column, ManyToOne, JoinColumn, Index,
  OneToMany, OneToOne,
} from 'typeorm';
import type { OrderStatus, TicketStatus, StageResult, TaskStatus, ComplaintStatus, ScanResult, ActivationResult } from '../enums.js';
import { Customer, ProductOffer, Channel } from './customer.js';
import { Address } from './geo.js';
import { WorkerGroup } from './org.js';
import { Worker } from './worker.js';

@Entity('orders', { comment: 'L4订单:12环节全流程,成交价快照防调价影响历史' })
@Index(['orderNo'], { unique: true })
export class Order {
  @PrimaryGeneratedColumn({ comment: '主键' })
  orderId!: number;

  @Column({ type: 'varchar', length: 32, unique: true, comment: '订单号,如ORD-20260817-001' })
  orderNo!: string;

  @ManyToOne(() => Customer, { nullable: false })
  @JoinColumn({ name: 'customer_id' })
  customer!: Customer;

  @Column({ type: 'varchar', length: 64, comment: '客户姓名快照:客户改名不改历史订单' })
  customerName!: string;

  @ManyToOne(() => ProductOffer, { nullable: false })
  @JoinColumn({ name: 'offer_id' })
  offer!: ProductOffer;

  @Column({ type: 'varchar', length: 128, comment: '产品名称快照:产品改名/调价不改历史订单' })
  offerName!: string;

  @ManyToOne(() => Channel, { nullable: false })
  @JoinColumn({ name: 'channel_id' })
  channel!: Channel;

  @Column({ type: 'varchar', length: 64, comment: '渠道名快照:渠道改名不改历史订单(REQ-ORD-006 必填不可改)' })
  channelName!: string;

  @Column({ type: 'numeric', precision: 10, scale: 2, comment: '成交价快照:下单时生效价(区域价优先),账单金额以此为准' })
  priceSnapshot!: number;

  @ManyToOne(() => Address, { nullable: false })
  @JoinColumn({ name: 'address_id' })
  address!: Address;

  @Column({ type: 'varchar', length: 128, comment: '区域LTREE路径:数据权限锚点,须被offer公司经营区域覆盖' })
  regionPath!: string;

  @Column({ type: 'bigint', comment: '企业归属id(legal_entities)快照:下单时客户所属企业,跨企业对比锚点' })
  legalEntityId!: number;

  @Column({ type: 'varchar', length: 128, comment: '企业名快照:改名不改历史订单' })
  legalEntityName!: string;

  @Column({ type: 'smallint', comment: '当前环节:1~12(下单→资源核查→端口预占→…→更新GIS,见terms.md)' })
  stage!: number;

  @Column({ type: 'varchar', length: 16, comment: '状态:PENDING待核查/RESERVED已预占/INSTALLING装维中/DONE已完成' })
  status!: OrderStatus;

  /** 12环节时间轴 */
  @OneToMany(() => OrderStage, (s) => s.order)
  stages!: OrderStage[];

  /** 派单工单(订单1:1) */
  @OneToOne(() => DispatchTicket, (t) => t.order)
  dispatchTicket!: DispatchTicket;

  /** 扫码绑定记录 */
  @OneToMany(() => ScanLog, (s) => s.order)
  scanLogs!: ScanLog[];

  /** 激活回调 */
  @OneToMany(() => ActivationCallback, (c) => c.order)
  activationCallbacks!: ActivationCallback[];

  /** 拆机单 */
  @OneToMany(() => Dismantle, (d) => d.order)
  dismantles!: Dismantle[];

  /** 关联报障工单 */
  @OneToMany(() => Complaint, (c) => c.order)
  complaints!: Complaint[];
}

@Entity('order_stages', { comment: 'L5订单环节时间轴:每订单12行,记录耗时与重试' })
@Index(['orderId', 'stage'], { unique: true })
export class OrderStage {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @ManyToOne(() => Order, { nullable: false })
  @JoinColumn({ name: 'order_id' })
  order!: Order;

  @Column({ type: 'smallint', comment: '环节号:1~12' })
  stage!: number;

  @Column({ type: 'varchar', length: 32, comment: '环节名称' })
  name!: string;

  @Column({ type: 'varchar', length: 8, default: 'PENDING', comment: '结果:PENDING待执行/DOING进行中/DONE完成' })
  result!: StageResult;

  @Column({ type: 'smallint', default: 0, comment: '回退重试次数' })
  retries!: number;

  @Column({ type: 'timestamptz', nullable: true, comment: '完成时间' })
  finishedAt?: Date | null;

  @Column({ type: 'bigint', nullable: true, comment: '执行人账号id:谁完成该环节(客服/财务/调度),系统环节为空' })
  operatorAccountId?: number | null;

  @Column({ type: 'varchar', length: 64, nullable: true, comment: '执行人姓名快照:改名不改历史' })
  operatorName?: string | null;
}

@Entity('dispatch_tickets', { comment: 'L5派单工单:订单1:1,指派师傅' })
@Index(['ticketNo'], { unique: true })
export class DispatchTicket {
  @PrimaryGeneratedColumn({ comment: '主键' })
  ticketId!: number;

  @Column({ type: 'varchar', length: 32, unique: true, comment: '工单号' })
  ticketNo!: string;

  /** 订单 1:1 工单 */
  @OneToOne(() => Order, (o) => o.dispatchTicket)
  @JoinColumn({ name: 'order_id' })
  order!: Order;

  /** 指派师傅(双向:Worker.tickets) */
  @ManyToOne(() => Worker, (w) => w.tickets, { nullable: true })
  @JoinColumn({ name: 'worker_id' })
  worker?: Worker | null;

  @Column({ type: 'varchar', length: 64, nullable: true, comment: '派单时师傅姓名快照:师傅改名不改工单历史' })
  workerName?: string | null;

  @ManyToOne(() => WorkerGroup, (g) => g.tickets, { nullable: true })
  @JoinColumn({ name: 'group_id' })
  group?: WorkerGroup | null;

  @Column({ type: 'varchar', length: 64, nullable: true, comment: '派单时班组名称快照:班组改名不改工单历史' })
  groupName?: string | null;

  @Column({ type: 'integer', nullable: true, comment: '派单时师傅服务区域id快照(regions表):师傅换区域不改工单历史' })
  regionId?: number | null;

  @Column({ type: 'varchar', length: 64, nullable: true, comment: '派单时服务区域名快照' })
  regionName?: string | null;

  @Column({ type: 'bigint', comment: '企业归属id(legal_entities)快照:派单时所属企业,跨企业对比锚点' })
  legalEntityId!: number;

  @Column({ type: 'varchar', length: 128, comment: '企业名快照:改名不改历史工单' })
  legalEntityName!: string;

  @Column({ type: 'varchar', length: 16, default: 'PENDING', comment: '状态:PENDING待派/DOING进行中/DONE完成/CANCELED取消' })
  status!: TicketStatus;

  @Column({ type: 'timestamptz', nullable: true, comment: '预约上门时间(装维调度)' })
  appointmentAt?: Date | null;

  @Column({ type: 'timestamptz', nullable: true, comment: '实际完工时间(装维完成)' })
  finishedAt?: Date | null;

  /** 改派台账(历史:每次改派) */
  @OneToMany(() => DispatchTransfer, (t) => t.ticket)
  transfers!: DispatchTransfer[];
}

@Entity('dispatch_transfers', { comment: 'L5派单改派台账:记录工单师傅每次改派,历史派单归属不随当前师傅漂移' })
@Index(['ticketId', 'transferredAt'])
export class DispatchTransfer {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @ManyToOne(() => DispatchTicket, (t) => t.transfers, { nullable: false })
  @JoinColumn({ name: 'ticket_id' })
  ticket!: DispatchTicket;

  @ManyToOne(() => Worker, { nullable: true })
  @JoinColumn({ name: 'from_worker_id' })
  fromWorker?: Worker | null;

  @Column({ type: 'varchar', length: 64, nullable: true, comment: '原师傅姓名快照' })
  fromWorkerName?: string | null;

  @ManyToOne(() => Worker, { nullable: true })
  @JoinColumn({ name: 'to_worker_id' })
  toWorker?: Worker | null;

  @Column({ type: 'varchar', length: 64, nullable: true, comment: '新师傅姓名快照' })
  toWorkerName?: string | null;

  @Column({ type: 'varchar', length: 128, nullable: true, comment: '改派原因,如师傅请假/超时' })
  reason?: string | null;

  @Column({ type: 'bigint', nullable: true, comment: '操作人账号id(调度员)' })
  operatorAccountId?: number | null;

  @Column({ type: 'timestamptz', comment: '改派时间' })
  transferredAt!: Date;
}

@Entity('scan_logs', { comment: 'L5扫码绑定记录:装维扫码与预绑定标签比对' })
export class ScanLog {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @ManyToOne(() => Order, { nullable: false })
  @JoinColumn({ name: 'order_id' })
  order!: Order;

  @Column({ type: 'bigint', comment: '扫码师傅id' })
  workerId!: number;

  @Column({ type: 'varchar', length: 64, comment: '扫码师傅姓名快照:改名不改历史' })
  workerName!: string;

  @Column({ type: 'bigint', comment: '被扫标签id(tags表)' })
  tagId!: number;

  @Column({ type: 'varchar', length: 16, comment: '比对结果:MATCH一致/MISMATCH不一致/OFFLINE_CACHED离线缓存' })
  result!: ScanResult;
}

@Entity('activation_callbacks', { comment: 'L5激活回调:订单第11环节的系统间回执' })
export class ActivationCallback {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @ManyToOne(() => Order, { nullable: false })
  @JoinColumn({ name: 'order_id' })
  order!: Order;

  @Column({ type: 'varchar', length: 16, comment: '回调结果:SUCCESS成功/FAILED失败(RETRYING重试中为展示态,由FAILED+重试派生)' })
  result!: ActivationResult;

  @Column({ type: 'smallint', default: 0, comment: '重试次数' })
  retries!: number;
}

@Entity('dismantles', { comment: 'L5拆机单:回收资产释放端口' })
@Index(['dismantleNo'], { unique: true })
export class Dismantle {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @Column({ type: 'varchar', length: 32, unique: true, comment: '拆机单号' })
  dismantleNo!: string;

  @ManyToOne(() => Order, { nullable: false })
  @JoinColumn({ name: 'order_id' })
  order!: Order;

  @Column({ type: 'bigint', comment: '企业归属id(legal_entities)快照:拆机订单所属企业' })
  legalEntityId!: number;

  @Column({ type: 'varchar', length: 128, comment: '企业名快照:改名不改历史' })
  legalEntityName!: string;

  @Column({ type: 'bigint', comment: '回收资产id(assets表)' })
  assetId!: number;

  @Column({ type: 'bigint', comment: '释放端口id(ports表)' })
  portId!: number;

  @Column({ type: 'varchar', length: 16, default: 'PENDING', comment: '状态:PENDING/DOING/DONE/FAILED' })
  status!: TaskStatus;
}

@Entity('complaints', { comment: 'L5报障工单(客服域,挂靠boss分组):客户报障与处理' })
@Index(['ticketNo'], { unique: true })
export class Complaint {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @Column({ type: 'varchar', length: 32, unique: true, comment: '报障工单号' })
  ticketNo!: string;

  @ManyToOne(() => Customer, { nullable: false })
  @JoinColumn({ name: 'customer_id' })
  customer!: Customer;

  @ManyToOne(() => Order, { nullable: true })
  @JoinColumn({ name: 'order_id' })
  order?: Order | null;

  @Column({ type: 'bigint', comment: '企业归属id(legal_entities)快照:报障客户所属企业' })
  legalEntityId!: number;

  @Column({ type: 'varchar', length: 128, comment: '企业名快照:改名不改历史报障' })
  legalEntityName!: string;

  @Column({ type: 'varchar', length: 32, comment: '报障类型,如网速慢/断网' })
  type!: string;

  @Column({ type: 'varchar', length: 16, default: 'OPEN', comment: '状态:OPEN受理中/PROCESSING处理中/CLOSED已关闭' })
  status!: ComplaintStatus;
}
