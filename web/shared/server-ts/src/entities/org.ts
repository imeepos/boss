// L0 运营主体/角色权限 + L1 班组 + L2 部门 + L3 岗位/账号。
// 组织链只做归属展示,不参与数据权限(region_scope 才是权限入口)。
import {
  Entity, PrimaryGeneratedColumn, PrimaryColumn, Column, ManyToOne, JoinColumn,
  ManyToMany, JoinTable, Index, OneToMany,
} from 'typeorm';
import type { RoleCode, AccountStatus } from '../enums.js';
import {
  Worker, WorkerGroupMembership, WorkerPerformance, WorkerCommission,
  WorkerSchedule, WorkerMaterial, WorkerTool, WorkerFeedback, AssetReturn,
} from './worker.js';
import { DispatchTicket } from './order.js';
import { Customer, ProductOffer } from './customer.js';
import { AssetBatch, Tag, Stocktake } from './asset.js';
import { Resource, Expansion, ProvisionTemplate, QosTemplate } from './oss.js';

@Entity('legal_entities', { comment: 'L0运营主体(子公司):各域数据的公司归属挂靠根' })
export class LegalEntity {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @Column({ type: 'varchar', length: 32, unique: true, comment: '公司编码,如LEG-A/B/C' })
  code!: string;

  @Column({ type: 'varchar', length: 128, comment: '公司全称' })
  name!: string;

  @Column({ type: 'jsonb', default: [], comment: '经营区域id列表(同层软关联):未经营区域不得设区域价/下单/派师傅' })
  crossRegionIds!: number[];

  /** 下属师傅班组 */
  @OneToMany(() => WorkerGroup, (w) => w.legalEntity)
  workerGroups!: WorkerGroup[];

  /** 下属部门 */
  @OneToMany(() => Department, (d) => d.legalEntity)
  departments!: Department[];

  /** 归属员工账号 */
  @OneToMany(() => Account, (a) => a.legalEntity)
  accounts!: Account[];

  /** 在售产品目录 */
  @OneToMany(() => ProductOffer, (p) => p.legalEntity)
  productOffers!: ProductOffer[];

  /** 归属客户 */
  @OneToMany(() => Customer, (c) => c.legalEntity)
  customers!: Customer[];

  /** 资产入库批次 */
  @OneToMany(() => AssetBatch, (b) => b.legalEntity)
  assetBatches!: AssetBatch[];

  /** 电子标签库存 */
  @OneToMany(() => Tag, (t) => t.legalEntity)
  tags!: Tag[];

  /** 网络设备 */
  @OneToMany(() => Resource, (r) => r.legalEntity)
  resources!: Resource[];

  /** 扩容单 */
  @OneToMany(() => Expansion, (e) => e.legalEntity)
  expansions!: Expansion[];

  /** 盘点任务 */
  @OneToMany(() => Stocktake, (s) => s.legalEntity)
  stocktakes!: Stocktake[];

  /** 配置下发模板 */
  @OneToMany(() => ProvisionTemplate, (t) => t.legalEntity)
  provisionTemplates!: ProvisionTemplate[];

  /** QoS模板 */
  @OneToMany(() => QosTemplate, (t) => t.legalEntity)
  qosTemplates!: QosTemplate[];
}

@Entity('worker_groups', { comment: 'L1师傅班组:运营主体自定义组织数据,非集团基础数据' })
@Index(['legalEntityId', 'code'], { unique: true })
export class WorkerGroup {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @Column({ type: 'varchar', length: 32, comment: '班组编码,公司内唯一(稳定标识,name可改code不变)' })
  code!: string;

  @Column({ type: 'varchar', length: 64, comment: '班组名称' })
  name!: string;

  @ManyToOne(() => LegalEntity, { nullable: false })
  @JoinColumn({ name: 'legal_entity_id' })
  legalEntity!: LegalEntity;

  /** 组长(1个师傅,可空) */
  @ManyToOne(() => Worker, (w) => w.leaderOfGroups, { nullable: true })
  @JoinColumn({ name: 'leader_id' })
  leader?: Worker | null;

  @Column({ type: 'varchar', length: 64, nullable: true, comment: '组长姓名快照:组长改名不改历史' })
  leaderName?: string | null;

  /** 班组内师傅 */
  @OneToMany(() => Worker, (w) => w.group)
  workers!: Worker[];

  /** 归属台账(师傅进/出班组的历史,含已调出者) */
  @OneToMany(() => WorkerGroupMembership, (m) => m.group)
  memberships!: WorkerGroupMembership[];

  /** 名下派单工单(下钻:这个班组的数据) */
  @OneToMany(() => DispatchTicket, (t) => t.group)
  tickets!: DispatchTicket[];

  /** 班组绩效(月×班组) */
  @OneToMany(() => WorkerPerformance, (p) => p.group)
  performances!: WorkerPerformance[];

  /** 班组佣金(月×班组) */
  @OneToMany(() => WorkerCommission, (c) => c.group)
  commissions!: WorkerCommission[];

  /** 班组考勤(月×班组) */
  @OneToMany(() => WorkerSchedule, (s) => s.group)
  schedules!: WorkerSchedule[];

  /** 班组物料领用 */
  @OneToMany(() => WorkerMaterial, (m) => m.group)
  materials!: WorkerMaterial[];

  /** 班组工具借用 */
  @OneToMany(() => WorkerTool, (t) => t.group)
  tools!: WorkerTool[];

  /** 班组收到的服务评价 */
  @OneToMany(() => WorkerFeedback, (f) => f.group)
  feedbacks!: WorkerFeedback[];

  /** 班组资产归还 */
  @OneToMany(() => AssetReturn, (r) => r.group)
  assetReturns!: AssetReturn[];
}

@Entity('departments', { comment: 'L2部门:归属子公司,公司内名称唯一' })
@Index(['legalEntityId', 'name'], { unique: true })
export class Department {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @Column({ type: 'varchar', length: 64, comment: '部门名称(子公司内唯一)' })
  name!: string;

  @ManyToOne(() => LegalEntity, { nullable: false })
  @JoinColumn({ name: 'legal_entity_id' })
  legalEntity!: LegalEntity;

  /** 下属岗位 */
  @OneToMany(() => Post, (p) => p.dept)
  posts!: Post[];

  /** 部门内员工账号 */
  @OneToMany(() => Account, (a) => a.dept)
  accounts!: Account[];
}

@Entity('roles', { comment: 'L0角色:集团统一7角色码' })
export class Role {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @Column({ type: 'varchar', length: 32, unique: true, comment: '角色码:customer/technician/asset_admin/resource_admin/ops/analyst/sysadmin' })
  code!: RoleCode;

  @Column({ type: 'varchar', length: 64, comment: '角色中文名' })
  name!: string;

  @ManyToMany(() => Permission, (p) => p.roles)
  @JoinTable({
    name: 'role_permissions',
    joinColumn: { name: 'role_id', referencedColumnName: 'id' },
    inverseJoinColumn: { name: 'permission_id', referencedColumnName: 'id' },
  })
  permissions!: Permission[];

  /** 拥有该角色的账号 */
  @OneToMany(() => Account, (a) => a.role)
  accounts!: Account[];

  /** 配置该角色的岗位 */
  @ManyToMany(() => Post, (p) => p.roles)
  posts!: Post[];
}

@Entity('permissions', { comment: 'L0权限码:按钮层鉴权,如asset:create/menu:order' })
export class Permission {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @Column({ type: 'varchar', length: 64, unique: true, comment: '权限码,格式<域>:<动作>' })
  code!: string;

  @Column({ type: 'varchar', length: 128, comment: '权限名称' })
  name!: string;

  /** 授予该权限的角色(M2M反向) */
  @ManyToMany(() => Role, (r) => r.permissions)
  roles!: Role[];
}

@Entity('posts', { comment: 'L3岗位:归属部门,经post_roles关联角色' })
export class Post {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @Column({ type: 'varchar', length: 32, comment: '岗位编码:dispatcher/cashier/agent/field_tech(部门内唯一)' })
  code!: string;

  @Column({ type: 'varchar', length: 64, comment: '岗位中文名' })
  name!: string;

  @ManyToOne(() => Department, { nullable: false })
  @JoinColumn({ name: 'dept_id' })
  dept!: Department;

  @ManyToMany(() => Role, (r) => r.posts)
  @JoinTable({
    name: 'post_roles',
    joinColumn: { name: 'post_id', referencedColumnName: 'id' },
    inverseJoinColumn: { name: 'role_id', referencedColumnName: 'id' },
  })
  roles!: Role[];

  /** 在岗员工账号 */
  @OneToMany(() => Account, (a) => a.post)
  accounts!: Account[];
}

@Entity('accounts', { comment: 'L3员工账号(权限主体):四种能力的载体(数据/界面/按钮/接口)' })
export class Account {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @Column({ type: 'varchar', length: 64, unique: true, comment: '登录用户名' })
  username!: string;

  @Column({ type: 'varchar', length: 64, comment: '真实姓名' })
  realName!: string;

  @Column({ type: 'varchar', length: 32, nullable: true, comment: '联系电话' })
  phone?: string | null;

  @Column({ select: false, comment: '密码哈希,默认不查询' })
  passwordHash!: string;

  @ManyToOne(() => Role, { nullable: false })
  @JoinColumn({ name: 'role_id' })
  role!: Role;

  @ManyToOne(() => LegalEntity, { nullable: true })
  @JoinColumn({ name: 'legal_entity_id' })
  legalEntity?: LegalEntity | null;

  @ManyToOne(() => Department, { nullable: true })
  @JoinColumn({ name: 'dept_id' })
  dept?: Department | null;

  @ManyToOne(() => Post, { nullable: true })
  @JoinColumn({ name: 'post_id' })
  post?: Post | null;

  /** 组织归属台账(历史:调岗/调部门/调公司) */
  @OneToMany(() => AccountOrgHistory, (h) => h.account)
  orgHistory!: AccountOrgHistory[];

  @Column({ type: 'varchar', length: 128, nullable: true, comment: '数据权限范围:LTREE路径,空=全集团;数据可见性=region_path落在该子树内' })
  regionScope?: string | null;

  @Column({ type: 'smallint', default: 1, comment: '状态:1启用 0停用' })
  status!: AccountStatus;
}

@Entity('account_org_histories', { comment: 'L3账号组织归属台账:记录账号每次调岗/调部门/调公司的时间段,历史归属不随当前岗位漂移' })
@Index(['accountId', 'effectiveFrom'])
export class AccountOrgHistory {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @ManyToOne(() => Account, (a) => a.orgHistory, { nullable: false })
  @JoinColumn({ name: 'account_id' })
  account!: Account;

  @ManyToOne(() => LegalEntity, { nullable: true })
  @JoinColumn({ name: 'legal_entity_id' })
  legalEntity?: LegalEntity | null;

  @Column({ type: 'varchar', length: 128, nullable: true, comment: '公司名快照:改名不改历史台账' })
  legalEntityName?: string | null;

  @ManyToOne(() => Department, { nullable: true })
  @JoinColumn({ name: 'dept_id' })
  dept?: Department | null;

  @Column({ type: 'varchar', length: 64, nullable: true, comment: '部门名快照:改名不改历史台账' })
  deptName?: string | null;

  @ManyToOne(() => Post, { nullable: true })
  @JoinColumn({ name: 'post_id' })
  post?: Post | null;

  @Column({ type: 'varchar', length: 64, nullable: true, comment: '岗位名快照:改名不改历史台账' })
  postName?: string | null;

  @Column({ type: 'varchar', length: 128, nullable: true, comment: '调动原因,如晋升/转岗/支援' })
  reason?: string | null;

  @Column({ type: 'bigint', nullable: true, comment: '操作人账号id:谁执行的调动' })
  operatorAccountId?: number | null;

  @Column({ type: 'timestamptz', comment: '归属生效时间' })
  effectiveFrom!: Date;

  @Column({ type: 'timestamptz', nullable: true, comment: '归属结束时间;null=至今' })
  effectiveTo?: Date | null;
}

@Entity('biz_params', { comment: 'L0业务参数:全局键值配置(欠费阈值/预占有效期/核对周期等)' })
export class BizParam {
  @PrimaryColumn({ type: 'varchar', length: 128, comment: '参数键' })
  key!: string;

  @Column({ type: 'jsonb', comment: '参数值(JSON)' })
  value!: object;

  @Column({ type: 'varchar', length: 255, nullable: true, comment: '参数说明' })
  description?: string | null;

  @Column({ type: 'bigint', nullable: true, comment: '更新人账号id' })
  updatedBy?: number | null;

  @Column({ type: 'timestamptz', default: () => 'now()', comment: '更新时间' })
  updatedAt!: Date;
}
