# 新环境部署与初始化清单(Q3 验收③)

> 目标:新环境(单节点或三节点)按本清单从零完成部署、初始化和验收,不依赖口头经验。
> 配套:`docs/deploy/production-cluster.md`(集群架构)、`docs/deploy/troubleshooting.md`(排障)。
> 纪律:每步有验证命令与通过标准;任何一步不过不进入下一步;验收签核表全部勾完才算交付。

## 0. 前置检查(节点就绪)

| # | 项 | 验证 | 通过标准 |
|---|---|---|---|
| 0.1 | Docker/compose | `docker version && docker compose version` | 均有输出 |
| 0.2 | 私有镜像仓库可达 | `curl -sf http://192.168.0.102:5000/v2/_catalog` | 返回 JSON(或离线包已 load) |
| 0.3 | 端口空闲 | `ss -ltnp \| grep -E ':(25432\|28080\|29090)'` | 无输出 |
| 0.4 | 数据目录 | `ls /data/boss` 或规划卷 | 已建且属主正确 |

## 1. 基础设施

```bash
docker compose -f deployments/docker-compose.102.yml up -d        # PG/Redis/Kafka/Nacos/MinIO/Temporal/VM
docker compose -f deployments/docker-compose.102.extend.yml up -d # 可观测/网关(可选,验收可后置)
```

验证:

```bash
docker exec boss-infra-postgres-1 pg_isready -U boss   # accepting connections
```

## 2. 应用(自动迁移 + 超管引导)

**机制**(不要手工跑迁移):

- server 启动即按镜像内 `migrations/` 幂等补迁(`internal/pkg/database.Migrate`,已应用版本记入 `schema_migrations`)。
- `BOSS_ADMIN_USERNAME/BOSS_ADMIN_PASSWORD` 经 bootstrap 幂等创建 sysadmin(已存在不覆盖)。
- JWT secret 必须固定;轮换会使全部 token 失效。

```bash
docker compose -f deployments/docker-compose.102.app.yml up -d
```

验证(等待循环,直至通过):

```bash
until curl -sf http://<HOST>:28080/healthz; do sleep 5; done   # {"status":"ok"}
```

## 3. 数据库迁移核验(关键)

```bash
docker exec <pg> psql -U boss -d boss -Atc "SELECT count(*) FROM schema_migrations"
ls migrations/*.up.sql | wc -l   # 与部署镜像内编号数一致
```

通过标准:两侧数量相等;不等=迁移漏跑(403/缺表首要嫌疑),查 server 日志后重起 server 自动补迁。

## 4. 超管与权限模板核验

```bash
TOKEN=$(curl -s -X POST http://<HOST>:28080/api/admin/v1/auth/login \
  -H 'Content-Type: application/json' \
  -d '{"username":"<admin>","password":"<pwd>"}' | python3 -c 'import json,sys;print(json.load(sys.stdin)["data"]["token"])')
curl -s http://<HOST>:28080/api/admin/v1/auth/me -H "Authorization: Bearer $TOKEN"
```

通过标准:

- `code=0`,`roleCode=sysadmin`
- `permissionCodes` 非空(sysadmin 应持有全量;菜单权限由迁移种子授予)
- `SELECT count(*) FROM permissions WHERE code LIKE 'menu:%'` ≥ 60(随菜单数增长;102 现网 62)

## 5. 基础数据初始化(L0→L1→L2,按 data-layers.md 层序)

按需三选一,禁止跳层:

1. **空环境**:Admin UI 逐层录入——regions(经营区域)→ legal_entities → departments/posts → product_offers → resources/ports;或走 数据导入中心(`/base/importer`,Excel)。
2. **从存量环境克隆**:源环境 `POST /api/admin/v1/backup`(gzip JSONL 归档)→ 新环境恢复(迁移 000095,ON CONFLICT 追加)。
3. **国际属地**:geo 国家/行政区划经 `/geo` 导入(ISO 3166)。

验证(层序完整性):

```sql
SELECT 'regions' t, count(*) FROM regions
UNION ALL SELECT 'legal_entities', count(*) FROM legal_entities
UNION ALL SELECT 'product_offers', count(*) FROM product_offers
UNION ALL SELECT 'resources', count(*) FROM resources;
```

通过标准:每层非空;上层数量 ≥ 下层引用需求(无孤儿 FK)。

## 6. 数据隔离模板(租户/法人/区域)

- 每法人(legal_entity)独立 `legal_entity_id`;业务事实冗余 `legal_entity_name` 快照。
- 账号数据范围:`region_scope`(ltree 子树)+ 公司;Admin AAA/账实核对列表已在 SQL 层裁剪(勿依赖前端过滤)。
- 验证:建一个受限账号(region_scope=某子树)登录,`/lo-accounts` 只返回该子树账号。

## 7. 业务冒烟(真实接口,禁 mock)

最小集(全部期望 `code=0`):

```bash
curl -s http://<HOST>:28080/api/admin/v1/aaa/summary -H "Authorization: Bearer $TOKEN"
curl -s "http://<HOST>:28080/api/admin/v1/billing/ledger-recon?period=$(date +%Y-%m)" -H "Authorization: Bearer $TOKEN"
curl -s http://<HOST>:28080/api/admin/v1/reconciliations -H "Authorization: Bearer $TOKEN"
```

主链路验收:跑 `scripts/` 主链路验收脚本(见 docs/plan/q1-mainchain-acceptance 报告)或手工过 下单→收费→派单→扫码→激活。

## 8. 备份/恢复演练

```bash
curl -X POST http://<HOST>:28080/api/admin/v1/backup -H "Authorization: Bearer $TOKEN"   # 生成归档
```

通过标准:归档可下载、恢复到空库后步骤 3/5 的核验复跑通过。

## 9. 验收签核表

| 阶段 | 项 | 结果 | 执行人/日期 |
|---|---|---|---|
| 0 | 前置检查 | ☐ | |
| 1 | 基础设施健康 | ☐ | |
| 2 | healthz ok | ☐ | |
| 3 | 迁移数对齐 | ☐ | |
| 4 | 超管登录+权限码 | ☐ | |
| 5 | 基础数据层序核验 | ☐ | |
| 6 | 数据隔离验证 | ☐ | |
| 7 | 冒烟三接口+主链路 | ☐ | |
| 8 | 备份恢复演练 | ☐ | |

全部勾选后,本环境方可进入交付(交付物:签核表+每步验证输出留档)。

## 已知边界

- devseed(开发种子)只写 admin 口令,生产禁用;生产超管只走 env 引导。
- CI 连续 push 会并发触发部署撞容器名;验收前确认无进行中的部署(容器 Status 非 Created/Restarting)。
- 属地税局网关(CN/PH)凭据经环境变量注入;未配置=人工回填通道,不阻塞初始化。
