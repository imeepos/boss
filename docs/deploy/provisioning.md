# 生产环境初始化手册(102 现行版)

> Q1 交付物:生产初始化包。从零把一个新环境拉起到可接单,按序执行;
> 每步含验证动作,失败即停。

## 前置

- 硬件/网络:单机(102 现行)或三节点(production-cluster.md 目标态)
- Docker + docker compose、Go(构建机)、gitea runner(自动部署)

## 步骤

### 1. 基础设施(PG/Redis/Kafka/Nacos/MinIO/Temporal/VM)

```bash
make infra-up
# 验证: docker ps 全部 healthy;PG 可连:
docker exec boss-infra-postgres-1 psql -U boss -d boss -c "SELECT 1;"
```

### 2. 数据库迁移(app 启动自动执行,或手动)

```bash
# 手动(与 app 内置 Migrate 同语义):
scripts/ops/migrate-102.sh up
scripts/ops/migrate-102.sh status   # 验证: 待应用为空
```

迁移编号/撞号规则见 AGENTS.md;回滚路径见 oncall-102.md。

### 3. 应用拉起(CI 或手动)

```bash
# CI: push main → deploy-102 流水线自动构建镜像 + compose up + healthz
# 手动: deployments/docker-compose.102.app.yml up -d
# 验证: curl http://<host>:28080/healthz
```

启动引导(幂等):超管账号 sysadmin 由 EnsureSuperAdmin 自动创建。

### 4. 基础数据初始化(缺一不可接单)

| 数据 | 动作 | 验证 |
|------|------|------|
| 菲律宾行政区划 | migration 000041 内置 | SELECT count(*) FROM geo_subdivisions > 0 |
| 角色/菜单权限 | migration 000101/000102 内置 | admin 登录 me 返回菜单码 |
| 下单渠道 | POST /provision/channels(HALL/ONLINE/AGENT) | GET /provision/channels |
| 产品资费 | POST /products(status=PUBLISHED) | GET /products |
| 师傅 | 师傅注册→后台审核→实名 | GET /worker-registrations |
| 资源/端口 | POST /provision/resources + /provision/ports | GET /ports |

### 5. 冒烟验收(接单能力闭环)

```bash
scripts/ops/mainchain-acceptance.sh 3   # 3/3 = 环境可接单
```

### 6. 开通值班例行

- 备份/验收/迁移状态例行见 docs/deploy/oncall-102.md;
- 首次备份: scripts/ops/backup-102.sh。

## 常见坑

- 迁移撞号:新环境前先 `git ls-tree` 查未合并分支占号(AGENTS.md 规则)
- app.env 密钥:见 adopted/2026-08-18-app-env-in-repo(固定密钥裁定)
- 端口资源耗尽:地址下 IDLE 口耗尽 → 新增分光器/端口
