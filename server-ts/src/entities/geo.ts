// L0 地理基础数据: regions(权限区域树) + addresses(地址挂接树)。
// LTREE 在 PG 用 ltree 类型;此处以 varchar 表达 path,迁移时替换。
import { Entity, PrimaryGeneratedColumn, Column, Index, OneToMany, ManyToOne, JoinColumn } from 'typeorm';
import type { RegionLevel, AddressLevel } from '../enums.js';
import { Customer } from './customer.js';
import { Order } from './order.js';
import { Asset } from './asset.js';
import { Resource, Port } from './oss.js';

@Entity('regions', { comment: 'L0区域树(数据权限唯一依据):1集团/2大区/3省/4城市,LTREE物化路径' })
@Index(['path'], { unique: true })
export class Region {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @Column({ type: 'varchar', length: 128, comment: 'LTREE物化路径,唯一权威,如CN.NORTH.BJ.CITY' })
  path!: string;

  @Column({ type: 'smallint', comment: '层级:1集团 2大区 3省 4城市(=nlevel(path))' })
  level!: RegionLevel;

  @Column({ type: 'varchar', length: 64, comment: '区域名称' })
  name!: string;

  /** 该经营区域下的楼栋级地址 */
  @OneToMany(() => Address, (a) => a.region)
  addresses!: Address[];
}

@Entity('addresses', { comment: 'L0地址挂接树:1市/2区/3街道/4小区/5楼栋,客户与端口必须挂楼栋级' })
@Index(['path'], { unique: true })
export class Address {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @Column({ type: 'varchar', length: 128, comment: 'LTREE物化路径,唯一权威' })
  path!: string;

  @Column({ type: 'smallint', comment: '层级:1市 2区 3街道 4小区 5楼栋' })
  level!: AddressLevel;

  @Column({ type: 'varchar', length: 64, comment: '地址名称' })
  name!: string;

  @Column({ type: 'bigint', nullable: true, comment: '父节点id(派生:反查path父节点,应用层不手填)' })
  parentId?: number | null;

  /** 所属经营区域(楼栋级挂城市区域,固化地址→区域硬关联;客户/资产/端口经此继承区域) */
  @ManyToOne(() => Region, (r) => r.addresses, { nullable: true })
  @JoinColumn({ name: 'region_id' })
  region?: Region | null;

  @Column({ type: 'varchar', length: 64, nullable: true, comment: '经营区域名快照:改名不改历史' })
  regionName?: string | null;

  /** 该楼栋下的客户(下钻:这个地址的数据) */
  @OneToMany(() => Customer, (c) => c.address)
  customers!: Customer[];

  /** 部署在该地址的资产 */
  @OneToMany(() => Asset, (a) => a.address)
  assets!: Asset[];

  /** 该地址的网络设备 */
  @OneToMany(() => Resource, (r) => r.address)
  resources!: Resource[];

  /** 该地址的端口 */
  @OneToMany(() => Port, (p) => p.address)
  ports!: Port[];

  /** 送往该地址的订单 */
  @OneToMany(() => Order, (o) => o.address)
  orders!: Order[];
}
