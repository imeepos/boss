// L1 产品目录(公司级) + L2 区域运营包(属地覆盖) + L3 客户档案。
// 计价: 生效价 = 区域价(前缀匹配) ?? 公司基础价; 订单只存快照。
import {
  Entity, PrimaryGeneratedColumn, Column, ManyToOne, JoinColumn, Index,
  OneToMany, OneToOne,
} from 'typeorm';
import type { ProductStatus, RealNameStatus, ServiceStatus } from '../enums.js';
import { LegalEntity } from './org.js';
import { Address } from './geo.js';
import { Order, Complaint } from './order.js';
import { LoAccount } from './oss.js';
import { Bill, Arrears } from './worker.js';

@Entity('product_offers', { comment: 'L1产品目录:名称/带宽/价格全属公司,各公司叫法不同(无集团规格层)' })
@Index(['legalEntityId', 'name'], { unique: true })
export class ProductOffer {
  @PrimaryGeneratedColumn({ comment: '主键' })
  offerId!: number;

  @ManyToOne(() => LegalEntity, { nullable: false })
  @JoinColumn({ name: 'legal_entity_id' })
  legalEntity!: LegalEntity;

  @Column({ type: 'varchar', length: 128, comment: '产品名称(公司级必填,展示名回退:区域名→公司名)' })
  name!: string;

  @Column({ type: 'varchar', length: 16, comment: '带宽档位,如300M/500M/1000M(公司自定)' })
  bandwidth!: string;

  @Column({ type: 'numeric', precision: 10, scale: 2, comment: '基础月费;生效价=区域价(前缀匹配)??此基础价' })
  monthlyFee!: number;

  @Column({ type: 'timestamptz', comment: '上架/调价生效时间' })
  effectiveAt!: Date;

  @Column({ type: 'varchar', length: 16, default: 'DRAFT', comment: '状态:DRAFT草稿/PUBLISHED在售/OFFLINE下架' })
  status!: ProductStatus;

  /** 区域运营包(属地名称/价格覆盖) */
  @OneToMany(() => RegionOffer, (r) => r.offer)
  regionOffers!: RegionOffer[];

  /** 按此产品下的订单 */
  @OneToMany(() => Order, (o) => o.offer)
  orders!: Order[];

  /** 调价台账(历史:每次基础月费调整) */
  @OneToMany(() => ProductPriceHistory, (h) => h.offer)
  priceHistory!: ProductPriceHistory[];

  /** 按此产品开通的LO账号 */
  @OneToMany(() => LoAccount, (l) => l.offer)
  loAccounts!: LoAccount[];
}

@Entity('region_offers', { comment: 'L2区域运营包:同一产品在不同区域的名称与价格覆盖' })
@Index(['offerId', 'regionPath'], { unique: true })
export class RegionOffer {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @ManyToOne(() => ProductOffer, { nullable: false })
  @JoinColumn({ name: 'offer_id' })
  offer!: ProductOffer;

  @Column({ type: 'varchar', length: 128, comment: '区域LTREE路径:须落在offer所属公司经营区域(应用层校验)' })
  regionPath!: string;

  @Column({ type: 'varchar', length: 128, nullable: true, comment: '区域级产品名,可空;空则回退公司名' })
  name?: string | null;

  @Column({ type: 'numeric', precision: 10, scale: 2, comment: '区域月费,覆盖公司基础价' })
  monthlyFee!: number;

  @Column({ type: 'varchar', length: 128, nullable: true, comment: '调价原因,如城市促销/新装立减' })
  reason?: string | null;

  /** 调价台账(历史:每次区域月费调整) */
  @OneToMany(() => RegionPriceHistory, (h) => h.regionOffer)
  priceHistory!: RegionPriceHistory[];
}

@Entity('customers', { comment: 'L3客户档案:归属运营主体,地址必须挂楼栋级' })
export class Customer {
  @PrimaryGeneratedColumn({ comment: '主键' })
  customerId!: number;

  @ManyToOne(() => LegalEntity, { nullable: false })
  @JoinColumn({ name: 'legal_entity_id' })
  legalEntity!: LegalEntity;

  @ManyToOne(() => Address, { nullable: false })
  @JoinColumn({ name: 'address_id' })
  address!: Address;

  @Column({ type: 'integer', comment: '经营区域id(regions表):地址所在经营区域,按地区统计锚点' })
  regionId!: number;

  @Column({ type: 'varchar', length: 64, comment: '经营区域名快照:改名不改历史' })
  regionName!: string;

  @Column({ type: 'varchar', length: 64, comment: '客户姓名/企业名' })
  name!: string;

  @Column({ type: 'varchar', length: 32, comment: '联系电话' })
  phone!: string;

  @Column({ type: 'varchar', length: 16, comment: '证件类型:身份证/护照/营业执照/无' })
  idType!: string;

  @Column({ type: 'varchar', length: 64, select: false, comment: '证件号码,敏感字段默认不查询' })
  idNo!: string;

  @Column({ type: 'varchar', length: 16, default: 'PENDING', comment: '实名状态:VERIFIED已实名/PENDING待补登' })
  realNameStatus!: RealNameStatus;

  @Column({ type: 'varchar', length: 16, default: 'ACTIVE', comment: '服务状态:ACTIVE在网/ARREARS欠费/SUSPENDED停机' })
  serviceStatus!: ServiceStatus;

  /** 归属台账(历史:转品牌/搬家) */
  @OneToMany(() => CustomerHistory, (h) => h.customer)
  history!: CustomerHistory[];

  /** 客户的订单(下钻:这个客户的数据) */
  @OneToMany(() => Order, (o) => o.customer)
  orders!: Order[];

  /** 客户的账单 */
  @OneToMany(() => Bill, (b) => b.customer)
  bills!: Bill[];

  /** 客户的报障工单 */
  @OneToMany(() => Complaint, (c) => c.customer)
  complaints!: Complaint[];

  /** LO认证账号(客户1:1) */
  @OneToOne(() => LoAccount, (l) => l.customer)
  loAccount!: LoAccount;

  /** 欠费态(客户1:1) */
  @OneToOne(() => Arrears, (a) => a.customer)
  arrears!: Arrears;
}

@Entity('customer_histories', { comment: 'L3客户归属台账:记录客户转品牌/搬家的时间段,历史归属不随当前值漂移' })
@Index(['customerId', 'effectiveFrom'])
export class CustomerHistory {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @ManyToOne(() => Customer, (c) => c.history, { nullable: false })
  @JoinColumn({ name: 'customer_id' })
  customer!: Customer;

  @ManyToOne(() => LegalEntity, { nullable: false })
  @JoinColumn({ name: 'legal_entity_id' })
  legalEntity!: LegalEntity;

  @Column({ type: 'varchar', length: 128, comment: '公司名快照:改名不改历史台账' })
  legalEntityName!: string;

  @ManyToOne(() => Address, { nullable: false })
  @JoinColumn({ name: 'address_id' })
  address!: Address;

  @Column({ type: 'varchar', length: 64, comment: '地址名快照:改名不改历史台账' })
  addressName!: string;

  @Column({ type: 'integer', comment: '事发时经营区域id(regions表)快照:客户搬家/转区域不改历史台账' })
  regionId!: number;

  @Column({ type: 'varchar', length: 64, comment: '事发时经营区域名快照:改名不改历史台账' })
  regionName!: string;

  @Column({ type: 'varchar', length: 128, nullable: true, comment: '变更原因,如搬家/转品牌' })
  reason?: string | null;

  @Column({ type: 'bigint', nullable: true, comment: '操作人账号id' })
  operatorAccountId?: number | null;

  @Column({ type: 'timestamptz', comment: '归属生效时间' })
  effectiveFrom!: Date;

  @Column({ type: 'timestamptz', nullable: true, comment: '归属结束时间;null=至今' })
  effectiveTo?: Date | null;
}

@Entity('product_price_histories', { comment: 'L1产品调价台账:记录产品基础月费每次调整,历史价不随当前价漂移' })
@Index(['offerId', 'effectiveAt'])
export class ProductPriceHistory {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @ManyToOne(() => ProductOffer, (o) => o.priceHistory, { nullable: false })
  @JoinColumn({ name: 'offer_id' })
  offer!: ProductOffer;

  @Column({ type: 'numeric', precision: 10, scale: 2, comment: '调价前基础月费' })
  oldMonthlyFee!: number;

  @Column({ type: 'numeric', precision: 10, scale: 2, comment: '调价后基础月费' })
  newMonthlyFee!: number;

  @Column({ type: 'timestamptz', comment: '调价生效时间' })
  effectiveAt!: Date;

  @Column({ type: 'varchar', length: 128, nullable: true, comment: '调价原因,如成本调整/促销' })
  reason?: string | null;

  @Column({ type: 'bigint', nullable: true, comment: '操作人账号id' })
  operatorAccountId?: number | null;
}

@Entity('region_price_histories', { comment: 'L2区域调价台账:记录区域运营包每次月费调整,历史价不随当前价漂移' })
@Index(['regionOfferId', 'effectiveAt'])
export class RegionPriceHistory {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @ManyToOne(() => RegionOffer, (r) => r.priceHistory, { nullable: false })
  @JoinColumn({ name: 'region_offer_id' })
  regionOffer!: RegionOffer;

  @Column({ type: 'numeric', precision: 10, scale: 2, comment: '调价前区域月费' })
  oldMonthlyFee!: number;

  @Column({ type: 'numeric', precision: 10, scale: 2, comment: '调价后区域月费' })
  newMonthlyFee!: number;

  @Column({ type: 'timestamptz', comment: '调价生效时间' })
  effectiveAt!: Date;

  @Column({ type: 'varchar', length: 128, nullable: true, comment: '调价原因,如促销/新装立减' })
  reason?: string | null;

  @Column({ type: 'bigint', nullable: true, comment: '操作人账号id' })
  operatorAccountId?: number | null;
}

@Entity('real_name_verifications', { comment: 'L3客户实名核验记录:核验方式/时间/结果审计轨迹(承接customer.html实名核验表)' })
@Index(['customerId', 'verifiedAt'])
export class RealNameVerification {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @ManyToOne(() => Customer, { nullable: false })
  @JoinColumn({ name: 'customer_id' })
  customer!: Customer;

  @Column({ type: 'varchar', length: 32, comment: '核验方式:人脸/证件OCR/人工/第三方' })
  method!: string;

  @Column({ type: 'timestamptz', comment: '核验时间' })
  verifiedAt!: Date;

  @Column({ type: 'varchar', length: 16, comment: '结果:PASS通过/FAIL不通过' })
  result!: string;

  @Column({ type: 'bigint', nullable: true, comment: '操作人账号id(accounts表)' })
  operatorAccountId?: number | null;

  @Column({ type: 'varchar', length: 64, nullable: true, comment: '操作人姓名快照:改名不改历史' })
  operatorName?: string | null;
}

@Entity('channels', { comment: 'L1渠道目录:下单来源(营业厅/线上/代理商),REQ-ORD-006 必填不可改' })
@Index(['code'], { unique: true })
export class Channel {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @Column({ type: 'varchar', length: 32, unique: true, comment: '渠道编码,如HALL营业厅/ONLINE线上/AGENT代理商' })
  code!: string;

  @Column({ type: 'varchar', length: 64, comment: '渠道名称' })
  name!: string;

  @Column({ type: 'varchar', length: 16, default: 'ACTIVE', comment: '状态:ACTIVE启用/DISABLED停用' })
  status!: string;
}
