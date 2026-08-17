// L1 网络设备/模板 + L4 端口/LO账号/扩容/调拨 + L5 预占/下发任务。
import {
  Entity, PrimaryGeneratedColumn, Column, ManyToOne, JoinColumn, Index, OneToMany, OneToOne,
} from 'typeorm';
import type { PortStatus, ResourceStatus, LoAccountStatus, TaskStatus } from '../enums.js';
import { LegalEntity } from './org.js';
import { Address } from './geo.js';
import { Customer, ProductOffer } from './customer.js';

@Entity('resources', { comment: 'L1网络设备树:各公司建设的OLT→分光器,挂小区/楼栋地址' })
export class Resource {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @ManyToOne(() => LegalEntity, { nullable: false })
  @JoinColumn({ name: 'legal_entity_id' })
  legalEntity!: LegalEntity;

  @Column({ type: 'varchar', length: 32, unique: true, comment: '设备编码,如OLT-01/SPL-01' })
  code!: string;

  @Column({ type: 'varchar', length: 64, comment: '设备名称,如望京OLT-01' })
  name!: string;

  @Column({ type: 'varchar', length: 16, comment: '设备类型:OLT/SPLITTER(分光器)' })
  type!: string;

  /** 上级设备(树内上下级:分光器→OLT,OLT为空) */
  @ManyToOne(() => Resource, (r) => r.children, { nullable: true })
  @JoinColumn({ name: 'parent_id' })
  parent?: Resource | null;

  /** 下级设备(反向) */
  @OneToMany(() => Resource, (r) => r.parent)
  children!: Resource[];

  @ManyToOne(() => Address, { nullable: false })
  @JoinColumn({ name: 'address_id' })
  address!: Address;

  @Column({ type: 'varchar', length: 16, default: 'ONLINE', comment: '状态:ONLINE在线/OFFLINE离线/FAULT故障' })
  status!: ResourceStatus;

  /** 下挂端口(分光器的端口) */
  @OneToMany(() => Port, (p) => p.resource)
  ports!: Port[];

  /** 归属台账(历史:每次调拨/归属变更) */
  @OneToMany(() => ResourceAssignment, (a) => a.resource)
  assignments!: ResourceAssignment[];
}

@Entity('resource_assignments', { comment: 'L1设备归属台账:记录设备每次归属公司/区域/地址的时间段,历史归属不随调拨漂移' })
@Index(['resourceId', 'effectiveFrom'])
export class ResourceAssignment {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @ManyToOne(() => Resource, (r) => r.assignments, { nullable: false })
  @JoinColumn({ name: 'resource_id' })
  resource!: Resource;

  @ManyToOne(() => LegalEntity, { nullable: false })
  @JoinColumn({ name: 'legal_entity_id' })
  legalEntity!: LegalEntity;

  @Column({ type: 'varchar', length: 128, comment: '公司名快照:改名不改历史台账' })
  legalEntityName!: string;

  @ManyToOne(() => Address, { nullable: true })
  @JoinColumn({ name: 'address_id' })
  address?: Address | null;

  @Column({ type: 'varchar', length: 64, nullable: true, comment: '地址名快照:改名不改历史台账' })
  addressName?: string | null;

  @Column({ type: 'integer', nullable: true, comment: '区域id快照(regions表)' })
  regionId?: number | null;

  @Column({ type: 'varchar', length: 64, nullable: true, comment: '区域名快照' })
  regionName?: string | null;

  @Column({ type: 'varchar', length: 128, nullable: true, comment: '变更原因,如调拨/扩容' })
  reason?: string | null;

  @Column({ type: 'bigint', nullable: true, comment: '操作人账号id' })
  operatorAccountId?: number | null;

  @Column({ type: 'timestamptz', comment: '归属生效时间' })
  effectiveFrom!: Date;

  @Column({ type: 'timestamptz', nullable: true, comment: '归属结束时间;null=至今' })
  effectiveTo?: Date | null;
}

@Entity('ports', { comment: 'L4端口:挂分光器,占用态必带订单(数据权限与四码的物理锚点)' })
@Index(['portCode'], { unique: true })
export class Port {
  @PrimaryGeneratedColumn({ comment: '主键' })
  portId!: number;

  @Column({ type: 'varchar', length: 32, unique: true, comment: '端口编码,如P-SPL01-01' })
  portCode!: string;

  @Column({ type: 'varchar', length: 32, comment: '四码端口码(与quad_code体系对齐)' })
  quadCode!: string;

  /** 所属设备(通常是分光器) */
  @ManyToOne(() => Resource, (r) => r.ports)
  @JoinColumn({ name: 'resource_id' })
  resource!: Resource;

  @Column({ type: 'bigint', comment: '企业归属id(legal_entities)快照:所属设备所属企业' })
  legalEntityId!: number;

  @Column({ type: 'varchar', length: 128, comment: '企业名快照:改名不改历史端口' })
  legalEntityName!: string;

  @ManyToOne(() => Address, { nullable: false })
  @JoinColumn({ name: 'address_id' })
  address!: Address;

  @Column({ type: 'integer', comment: '经营区域id(regions表):地址所在经营区域,按地区统计锚点' })
  regionId!: number;

  @Column({ type: 'varchar', length: 64, comment: '经营区域名快照:改名不改历史' })
  regionName!: string;

  @Column({ type: 'bigint', nullable: true, comment: '占用订单id:RESERVED/USED态非空,其余态必须为null(应用层校验)' })
  orderId?: number | null;

  @Column({ type: 'varchar', length: 16, comment: '状态:IDLE空闲/RESERVED预占/USED在用/DISABLED停用' })
  status!: PortStatus;

  /** 状态变更历史(预占/占用/释放/停用) */
  @OneToMany(() => PortChangeHistory, (h) => h.port)
  changeHistory!: PortChangeHistory[];
}

@Entity('port_change_history', { comment: 'L5端口状态变更历史:记录端口每次状态/占用变化,历史不随当前状态漂移' })
@Index(['portId', 'changedAt'])
export class PortChangeHistory {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @ManyToOne(() => Port, (p) => p.changeHistory, { nullable: false })
  @JoinColumn({ name: 'port_id' })
  port!: Port;

  @Column({ type: 'varchar', length: 16, comment: '变更后状态:IDLE/RESERVED/USED/DISABLED' })
  status!: PortStatus;

  @Column({ type: 'bigint', nullable: true, comment: '占用订单id快照:RESERVED/USED时非空' })
  orderId?: number | null;

  @Column({ type: 'timestamptz', comment: '变更时间' })
  changedAt!: Date;
}

@Entity('reserve_records', { comment: 'L5端口预占记录:订单第3环节的预占/释放流水' })
export class ReserveRecord {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @Column({ type: 'bigint', comment: '端口id' })
  portId!: number;

  @Column({ type: 'bigint', comment: '预占订单id' })
  orderId!: number;

  @Column({ type: 'varchar', length: 16, comment: '状态:HELD持有/RELEASED已释放/CONSUMED已消费' })
  status!: string;
}

@Entity('transfers', { comment: 'L4资源调拨单:设备跨区域调拨' })
@Index(['transferNo'], { unique: true })
export class Transfer {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @Column({ type: 'varchar', length: 32, unique: true, comment: '调拨单号' })
  transferNo!: string;

  @Column({ type: 'bigint', comment: '调拨设备id(resources表)' })
  resourceId!: number;

  @Column({ type: 'bigint', comment: '企业归属id(legal_entities)快照:调拨设备所属企业' })
  legalEntityId!: number;

  @Column({ type: 'varchar', length: 128, comment: '企业名快照:改名不改历史' })
  legalEntityName!: string;

  @Column({ type: 'integer', comment: '调出区域id' })
  fromRegionId!: number;

  @Column({ type: 'integer', comment: '调入区域id' })
  toRegionId!: number;

  @Column({ type: 'varchar', length: 16, default: 'PENDING', comment: '状态:PENDING待审/DOING执行中/DONE完成' })
  status!: TaskStatus;
}

@Entity('expansions', { comment: 'L4扩容单:公司对目标区域的端口扩容需求' })
@Index(['expansionNo'], { unique: true })
export class Expansion {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @ManyToOne(() => LegalEntity, { nullable: false })
  @JoinColumn({ name: 'legal_entity_id' })
  legalEntity!: LegalEntity;

  @Column({ type: 'varchar', length: 32, unique: true, comment: '扩容单号' })
  expansionNo!: string;

  @Column({ type: 'integer', comment: '扩容目标区域id' })
  regionId!: number;

  @Column({ type: 'integer', comment: '预期新增端口数' })
  expectedPorts!: number;

  @Column({ type: 'varchar', length: 16, default: 'PENDING', comment: '状态:PENDING待审/DOING执行中/DONE完成' })
  status!: TaskStatus;
}

@Entity('provision_templates', { comment: 'L1下发模板:各公司设备型号不同,模板挂公司' })
export class ProvisionTemplate {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @ManyToOne(() => LegalEntity, { nullable: false })
  @JoinColumn({ name: 'legal_entity_id' })
  legalEntity!: LegalEntity;

  @Column({ type: 'varchar', length: 32, unique: true, comment: '模板编码,如TPL-FTTH' })
  code!: string;

  @Column({ type: 'varchar', length: 64, comment: '模板名称,如FTTH标准开通' })
  name!: string;
}

@Entity('qos_templates', { comment: 'L1 QoS模板:各公司服务策略不同,挂公司' })
export class QosTemplate {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @ManyToOne(() => LegalEntity, { nullable: false })
  @JoinColumn({ name: 'legal_entity_id' })
  legalEntity!: LegalEntity;

  @Column({ type: 'varchar', length: 32, unique: true, comment: '模板编码,如QoS-VIP' })
  code!: string;

  @Column({ type: 'varchar', length: 64, comment: '模板名称,如VIP' })
  name!: string;

  /** 应用该模板的LO账号 */
  @OneToMany(() => LoAccount, (l) => l.qosTemplate)
  loAccounts!: LoAccount[];
}

@Entity('lo_accounts', { comment: 'L4 LO认证账号:客户1:1,宽带认证与QoS生效载体' })
@Index(['loid'], { unique: true })
export class LoAccount {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @Column({ type: 'varchar', length: 32, unique: true, comment: 'LOID认证账号,如LOID-88A1' })
  loid!: string;

  /** 客户 1:1 */
  @OneToOne(() => Customer, (c) => c.loAccount)
  @JoinColumn({ name: 'customer_id' })
  customer!: Customer;

  @Column({ type: 'bigint', comment: '企业归属id(legal_entities)快照:客户所属企业' })
  legalEntityId!: number;

  @Column({ type: 'varchar', length: 128, comment: '企业名快照:改名不改历史LO账号' })
  legalEntityName!: string;

  @Column({ type: 'integer', comment: '经营区域id(regions表)快照:客户所在经营区域,按地区统计锚点' })
  regionId!: number;

  @Column({ type: 'varchar', length: 64, comment: '经营区域名快照:改名不改历史' })
  regionName!: string;

  @ManyToOne(() => ProductOffer, { nullable: false })
  @JoinColumn({ name: 'offer_id' })
  offer!: ProductOffer;

  @ManyToOne(() => QosTemplate, { nullable: false })
  @JoinColumn({ name: 'qos_template_id' })
  qosTemplate!: QosTemplate;

  @Column({ type: 'varchar', length: 16, comment: '状态:ACTIVE在服/SUSPENDED停服/CLOSED注销;QoS模板公司须与产品公司一致' })
  status!: LoAccountStatus;
}

@Entity('provision_tasks', { comment: 'L5配置下发任务:按模板向LO账号下发配置' })
export class ProvisionTask {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @Column({ type: 'bigint', comment: 'LO账号id;模板公司须与账号公司一致(应用层校验)' })
  loAccountId!: number;

  @Column({ type: 'bigint', comment: '下发模板id(provision_templates)' })
  templateId!: number;

  @Column({ type: 'varchar', length: 16, default: 'PENDING', comment: '状态:PENDING/DOING/DONE/FAILED' })
  status!: TaskStatus;
}

@Entity('provision_logs', { comment: 'L5配置下发日志:下发任务对目标设备的结果/重试流水(承接provlog.html)' })
@Index(['taskId', 'createdAt'])
export class ProvisionLog {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @Column({ type: 'bigint', comment: '下发任务id(provision_tasks表,软引用)' })
  taskId!: number;

  @Column({ type: 'bigint', comment: '目标设备id(resources表,软引用)' })
  resourceId!: number;

  @Column({ type: 'varchar', length: 64, nullable: true, comment: '设备编码快照,如OLT-01' })
  resourceCode?: string | null;

  @Column({ type: 'bigint', comment: '下发模板id(provision_templates表,软引用)' })
  templateId!: number;

  @Column({ type: 'varchar', length: 64, nullable: true, comment: '模板编码快照,如GPON-1000M-v3' })
  templateCode?: string | null;

  @Column({ type: 'varchar', length: 16, comment: '结果:SUCCESS成功/FAILED失败' })
  result!: string;

  @Column({ type: 'smallint', default: 0, comment: '重试次数' })
  retries!: number;

  @Column({ type: 'timestamptz', comment: '下发时间' })
  createdAt!: Date;
}
