// 实体统一出口 + DataSource(全局 snake_case 命名策略对齐 fields.md 第 0 节)。
import 'reflect-metadata';
import { DataSource } from 'typeorm';
import { SnakeNamingStrategy } from 'typeorm-naming-strategies';

export * from './enums.js';
export * from './entities/geo.js';
export * from './entities/org.js';
export * from './entities/customer.js';
export * from './entities/order.js';
export * from './entities/asset.js';
export * from './entities/oss.js';
export * from './entities/worker.js';

export function buildDataSource(opts: {
  host?: string; port?: number; username?: string; password?: string; database?: string;
} = {}) {
  return new DataSource({
    type: 'postgres',
    host: opts.host ?? '127.0.0.1',
    port: opts.port ?? 5432,
    username: opts.username ?? 'boss',
    password: opts.password ?? 'boss',
    database: opts.database ?? 'boss',
    synchronize: false, // 建表走 migrations,禁止开发期自动同步
    entities: [__dirname + '/entities/*{.js,.ts}'],
    namingStrategy: new SnakeNamingStrategy(),
  });
}
