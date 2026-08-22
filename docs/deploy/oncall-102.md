# 值班与事故处理手册(102 环境现行版)

> Q1 交付物。三节点目标态见 production-cluster.md;本手册只写当前 102 单机
> 部署可立即执行的动作,目标态落地后合并改版。

## 环境 fact

- 服务: boss-server / boss-admin-web / boss-aaa / boss-report @ 192.168.0.102
- PG: boss-infra-postgres-1(库 boss);Redis/Kafka/MinIO 同机 compose
- 入口: http://192.168.0.102:28080(admin 前缀 /api/admin/v1)
- 部署: gitea CI(Build-Deploy-to-ECS),app 启动时自动跑迁移(database.Migrate)

## 值班例行(每日)

1. 备份: `scripts/ops/backup-102.sh`(产物 + sha256,保留 14 份,失败=事故)
2. 主链路验收: `scripts/ops/mainchain-acceptance.sh 3`(成功率 <100% 立即排查)
3. 迁移状态: `scripts/ops/migrate-102.sh status`(待应用 ≠ 0 且无部署进行中 = 排查)

## 事故分级与动作

| 级别 | 判定 | 首要动作 |
|------|------|----------|
| P1 主链路不可用 | 验收脚本全红 / 健康检查失败 | `docker ps` 看 boss-server;`docker logs boss-server --since 15m` |
| P1 数据损坏 | 写操作大面积报错/数据异常 | 停写(停 boss-server),按备份恢复(见下) |
| P2 迁移失败 | 部署后 app 起不来,日志 `database: apply` | `migrate-102.sh down <上一版本>` 回滚库,再回滚镜像 tag |
| P3 局部功能 | 单端点 5xx | 日志按 trace_id 追;立 postmortem 候选 |

## 回滚路径(可验证)

1. 库回滚: `scripts/ops/migrate-102.sh down <目标版本>`(down 前自动强制备份)
2. 镜像回滚: 102 私有仓库 `192.168.0.102:5000/boss/server` 改回上一 tag 重启
3. 整库恢复:
   ```bash
   ssh imeepos@192.168.0.102
   LATEST=$(ls -1t ~/backups/pg/boss_*.dump | head -1)
   docker exec -i boss-infra-postgres-1 psql -U boss -d postgres -c "DROP DATABASE IF EXISTS boss_restore;"
   docker exec -i boss-infra-postgres-1 createdb -U boss boss_restore
   docker exec -i boss-infra-postgres-1 pg_restore -U boss -d boss_restore --no-owner < "$LATEST"
   # 校验通过后切换或按表搬运
   ```
   演练记录见 docs/acceptance/(每季度至少重演一次)。

## 事故记录

- 当天立档 `docs/postmortem/000N-*.md`,随修复同提交(制度见 AGENTS.md);
  索引与规则见 docs/postmortem/README.md。
- 不可逆决策当天过账 `docs/notes/adopted/`。
