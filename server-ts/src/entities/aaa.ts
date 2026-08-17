// L5 AAA 域: 话单(CDR) + 认证日志。
import { Entity, PrimaryGeneratedColumn, Column, Index } from 'typeorm';
import type { CdrBillingStatus, AuthResult } from '../enums.js';

@Entity('cdrs', { comment: 'L5话单(AAA域):RADIUS Accounting-Request 产出的计费原始记录' })
@Index(['loid', 'startedAt'])
export class CallDetailRecord {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @Column({ type: 'varchar', length: 32, comment: '认证账号LOID(lo_accounts.loid)' })
  loid!: string;

  @Column({ type: 'varchar', length: 64, nullable: true, comment: '用户名' })
  username?: string | null;

  @Column({ type: 'smallint', comment: '计费状态(rfc2866 AcctStatusType):1开始/2停止/3中间' })
  acctStatus!: number;

  @Column({ type: 'varchar', length: 64, nullable: true, comment: '会话ID' })
  sessionId?: string | null;

  @Column({ type: 'integer', comment: '会话时长(秒)' })
  sessionTime!: number;

  @Column({ type: 'bigint', comment: '上行流量(字节)' })
  inputOctets!: number;

  @Column({ type: 'bigint', comment: '下行流量(字节)' })
  outputOctets!: number;

  @Column({ type: 'varchar', length: 64, nullable: true, comment: 'NAS IP' })
  nasIp?: string | null;

  @Column({ type: 'varchar', length: 16, default: 'UNBILLED', comment: '入账状态:UNBILLED未入账/BILLED已入账' })
  billingStatus!: CdrBillingStatus;

  @Column({ type: 'timestamptz', comment: '开始时间' })
  startedAt!: Date;
}

@Entity('auth_logs', { comment: 'L5认证日志(AAA域):认证成功/失败记录' })
@Index(['loid', 'createdAt'])
export class AuthLog {
  @PrimaryGeneratedColumn({ comment: '主键' })
  id!: number;

  @Column({ type: 'varchar', length: 32, comment: '认证账号LOID' })
  loid!: string;

  @Column({ type: 'varchar', length: 16, comment: '结果:SUCCESS成功/FAILED失败' })
  result!: AuthResult;

  @Column({ type: 'timestamptz', comment: '认证时间' })
  createdAt!: Date;
}
