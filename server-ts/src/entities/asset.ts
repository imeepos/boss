// L4 资产/标签/批次 + L5 四码/换新/盘点。
import {
  Entity, PrimaryGeneratedColumn, Column, ManyToOne, JoinColumn, Index, OneToMany,
} from 'typeorm';
import type { AssetStatus, QuadStatus, TagStatus, TaskStatus } from '../enums.js';
import { LegalEntity } from './org.js';
import { Address } from './geo.js';
import { Worker } from './worker.js';

@Entity('asset_batches', { comment: 'L2入库批次:公司采购行为,资产经此归属公司' })
export class AssetBatch {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @ManyToOne(() => LegalEntity, { nullable: false })
  @JoinColumn({ name: 'legal_entity_id' })
  legalEntity!: LegalEntity;

  @Column({ type: 'varchar', length: 32, unique: true, comment: '批次编码,如RK-202607-01' })
  code!: string;

  @Column({ type: 'varchar', length: 64, comment: '批次名称,如7月光猫批次' })
  name!: string;

  /** 批次内资产 */
  @OneToMany(() => Asset, (a) => a.batch)
  assets!: Asset[];
}

@Entity('tags', { comment: 'L2电子标签池:公司库存,预绑定后才指向资产' })
@Index(['tagNo'], { unique: true })
export class Tag {
  @PrimaryGeneratedColumn({ comment: '主键' })
  tagId!: number;

  @ManyToOne(() => LegalEntity, { nullable: false })
  @JoinColumn({ name: 'legal_entity_id' })
  legalEntity!: LegalEntity;

  @Column({ type: 'varchar', length: 32, unique: true, comment: '标签编号' })
  tagNo!: string;

  @Column({ type: 'varchar', length: 32, unique: true, comment: 'EPC码(电子标签全球码)' })
  epcCode!: string;

  @Column({ type: 'varchar', length: 8, comment: '频段,如UHF' })
  band!: string;

  @Column({ type: 'bigint', nullable: true, comment: '预绑定资产id:显式null=未绑定(禁用占位伪值),跨层回填豁免' })
  boundAssetId?: number | null;

  @Column({ type: 'varchar', length: 16, default: 'UNBOUND', comment: '状态:UNBOUND未绑定/BOUND已绑定/DISABLED禁用' })
  status!: TagStatus;

  @Column({ type: 'varchar', length: 8, comment: '电池电量,如86%' })
  battery!: string;
}

@Entity('assets', { comment: 'L4资产台账:光猫/ONU等装维物资的全生命周期' })
@Index(['assetCode'], { unique: true })
export class Asset {
  @PrimaryGeneratedColumn({ comment: '主键' })
  assetId!: number;

  @Column({ type: 'varchar', length: 32, unique: true, comment: '资产编码,如A-20260001' })
  assetCode!: string;

  @ManyToOne(() => AssetBatch, { nullable: false })
  @JoinColumn({ name: 'batch_id' })
  batch!: AssetBatch;

  @Column({ type: 'bigint', comment: '企业归属id(legal_entities)快照:入库批次所属企业' })
  legalEntityId!: number;

  @Column({ type: 'varchar', length: 128, comment: '企业名快照:改名不改历史资产' })
  legalEntityName!: string;

  @Column({ type: 'bigint', nullable: true, comment: '绑定标签id(tags表),入库未绑定时为空' })
  tagId?: number | null;

  @ManyToOne(() => Address, { nullable: true })
  @JoinColumn({ name: 'address_id' })
  address?: Address | null;

  @Column({ type: 'integer', nullable: true, comment: '经营区域id(regions表):部署地址所在区域,未部署为空,按地区统计锚点' })
  regionId?: number | null;

  @Column({ type: 'varchar', length: 64, nullable: true, comment: '经营区域名快照:改名不改历史' })
  regionName?: string | null;

  @Column({ type: 'varchar', length: 32, comment: '资产类型:光猫/ONU/路由器等' })
  type!: string;

  @Column({ type: 'varchar', length: 16, comment: '状态:IN_STOCK在库/DEPLOYED在用/MAINTENANCE维保/SCRAPPED报废' })
  status!: AssetStatus;

  /** 状态轨迹(历史:每次状态/位置变更,不随当前状态漂移) */
  @OneToMany(() => AssetLifecycle, (l) => l.asset)
  lifecycles!: AssetLifecycle[];

  /** 持有台账(历史:每次领用/部署/归还) */
  @OneToMany(() => AssetAssignment, (a) => a.asset)
  assignments!: AssetAssignment[];
}

@Entity('asset_lifecycles', { comment: 'L5资产状态轨迹:记录资产每次状态/位置变更,历史不随当前状态漂移' })
@Index(['assetId', 'changedAt'])
export class AssetLifecycle {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @ManyToOne(() => Asset, (a) => a.lifecycles, { nullable: false })
  @JoinColumn({ name: 'asset_id' })
  asset!: Asset;

  @Column({ type: 'varchar', length: 16, comment: '状态:IN_STOCK/DEPLOYED/MAINTENANCE/SCRAPPED' })
  status!: AssetStatus;

  @ManyToOne(() => Address, { nullable: true })
  @JoinColumn({ name: 'address_id' })
  address?: Address | null;

  @Column({ type: 'varchar', length: 64, nullable: true, comment: '事发时地址名快照:改名不改历史' })
  addressName?: string | null;

  @Column({ type: 'bigint', nullable: true, comment: '经办师傅id(部署/归还人)' })
  workerId?: number | null;

  @Column({ type: 'varchar', length: 64, nullable: true, comment: '经办师傅姓名快照:改名不改历史' })
  workerName?: string | null;

  @Column({ type: 'timestamptz', comment: '变更时间' })
  changedAt!: Date;
}

@Entity('asset_assignments', { comment: 'L4资产持有台账:记录资产每次被师傅持有/部署位置的时间段,历史归属不随当前值漂移' })
@Index(['assetId', 'effectiveFrom'])
export class AssetAssignment {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @ManyToOne(() => Asset, (a) => a.assignments, { nullable: false })
  @JoinColumn({ name: 'asset_id' })
  asset!: Asset;

  @ManyToOne(() => Worker, { nullable: true })
  @JoinColumn({ name: 'worker_id' })
  worker?: Worker | null;

  @Column({ type: 'varchar', length: 64, nullable: true, comment: '持有师傅姓名快照:改名不改历史台账' })
  workerName?: string | null;

  @ManyToOne(() => Address, { nullable: true })
  @JoinColumn({ name: 'address_id' })
  address?: Address | null;

  @Column({ type: 'varchar', length: 64, nullable: true, comment: '部署地址名快照:改名不改历史台账' })
  addressName?: string | null;

  @Column({ type: 'varchar', length: 128, nullable: true, comment: '变更原因,如领用/部署/归还' })
  reason?: string | null;

  @Column({ type: 'bigint', nullable: true, comment: '操作人账号id' })
  operatorAccountId?: number | null;

  @Column({ type: 'timestamptz', comment: '持有生效时间' })
  effectiveFrom!: Date;

  @Column({ type: 'timestamptz', nullable: true, comment: '持有结束时间;null=至今' })
  effectiveTo?: Date | null;
}

@Entity('quad_links', { comment: 'L5四码合一:资产-客户-端口-地址四码关联(第二码是客户,不是LOID)' })
@Index(['assetId'], { unique: true })
@Index(['customerId'], { unique: true })
@Index(['portId'], { unique: true })
@Index(['addressId'], { unique: true })
export class QuadLink {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @Column({ type: 'bigint', comment: '资产id(assets表)' })
  assetId!: number;

  @Column({ type: 'bigint', comment: '客户id(customers表)——口径裁定:四码第2项=客户' })
  customerId!: number;

  @Column({ type: 'bigint', comment: '端口id(ports表)' })
  portId!: number;

  @Column({ type: 'bigint', comment: '地址id(addresses表,楼栋级)' })
  addressId!: number;

  @Column({ type: 'bigint', comment: '企业归属id(legal_entities)快照:四码客户所属企业' })
  legalEntityId!: number;

  @Column({ type: 'varchar', length: 128, comment: '企业名快照:改名不改历史' })
  legalEntityName!: string;

  @Column({ type: 'varchar', length: 16, comment: '状态:LINKED一致/CONFLICT冲突/UNLINKED未关联' })
  status!: QuadStatus;
}

@Entity('replacements', { comment: 'L5换新单:故障资产换新流程' })
@Index(['replacementNo'], { unique: true })
export class Replacement {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @Column({ type: 'varchar', length: 32, unique: true, comment: '换新单号' })
  replacementNo!: string;

  @Column({ type: 'bigint', comment: '故障资产id' })
  assetId!: number;

  @Column({ type: 'bigint', comment: '企业归属id(legal_entities)快照:故障资产所属企业' })
  legalEntityId!: number;

  @Column({ type: 'varchar', length: 128, comment: '企业名快照:改名不改历史' })
  legalEntityName!: string;

  @Column({ type: 'varchar', length: 64, comment: '换新原因,如光猫故障' })
  reason!: string;

  @Column({ type: 'varchar', length: 8, comment: '优先级:HIGH/MEDIUM/LOW' })
  priority!: string;

  @Column({ type: 'varchar', length: 16, default: 'PENDING', comment: '状态:PENDING/DOING/DONE/FAILED' })
  status!: TaskStatus;
}

@Entity('stocktakes', { comment: 'L5盘点任务:按区域盘点资产,输出差异' })
export class Stocktake {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @ManyToOne(() => LegalEntity, { nullable: false })
  @JoinColumn({ name: 'legal_entity_id' })
  legalEntity!: LegalEntity;

  @Column({ type: 'varchar', length: 128, comment: '盘点范围:区域LTREE路径' })
  scope!: string;

  @Column({ type: 'smallint', comment: '进度百分比0~100' })
  progress!: number;

  @Column({ type: 'integer', comment: '差异条数(账实不符)' })
  diffCount!: number;

  @Column({ type: 'varchar', length: 16, comment: '状态:DOING进行中/DONE完成' })
  status!: string;
}
