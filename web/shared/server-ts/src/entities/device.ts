// L5 MON 域: 设备监控指标 + 设备健康观察。
import { Entity, PrimaryGeneratedColumn, Column, Index } from 'typeorm';
import type { ResourceStatus, MaintenancePriority } from '../enums.js';

@Entity('device_metrics', { comment: 'L5设备监控指标:OLT光功率/丢包率快照(MON域,承接device.html)' })
@Index(['resourceId', 'collectedAt'])
export class DeviceMetric {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @Column({ type: 'bigint', comment: '设备id(resources表,软引用)' })
  resourceId!: number;

  @Column({ type: 'numeric', precision: 5, scale: 2, nullable: true, comment: '光功率(dBm),如-18.2' })
  opticalPower?: number | null;

  @Column({ type: 'numeric', precision: 5, scale: 2, nullable: true, comment: '丢包率(%),如0.01' })
  packetLoss?: number | null;

  @Column({ type: 'varchar', length: 16, comment: '设备状态:ONLINE在线/OFFLINE离线/FAULT故障' })
  status!: ResourceStatus;

  @Column({ type: 'timestamptz', comment: '采集时间' })
  collectedAt!: Date;
}

@Entity('device_maintenances', { comment: 'L5设备健康观察:师傅端主动运维清单(MON域)' })
@Index(['deviceNo'])
export class DeviceMaintenance {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @Column({ type: 'varchar', length: 64, comment: '设备号,如OLT-01' })
  deviceNo!: string;

  @Column({ type: 'varchar', length: 32, nullable: true, comment: '设备类型,如PON 9口' })
  deviceType?: string | null;

  @Column({ type: 'smallint', comment: '健康评分(0~100)' })
  healthScore!: number;

  @Column({ type: 'integer', comment: '故障次数' })
  faultCount!: number;

  @Column({ type: 'numeric', precision: 4, scale: 1, nullable: true, comment: '服役年数' })
  ageYears?: number | null;

  @Column({ type: 'varchar', length: 128, nullable: true, comment: '观察原因,如丢包率偏高' })
  reason?: string | null;

  @Column({ type: 'varchar', length: 16, comment: '优先级:MUST_REPLACE必须更换/SUGGEST建议/WATCH观察' })
  priority!: MaintenancePriority;
}
