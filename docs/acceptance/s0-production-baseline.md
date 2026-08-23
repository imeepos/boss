# S0 生产基线验收记录

> 状态：代码与自动化回归已完成；102 远端部署/迁移/开放平台写路径需按本文件执行后填写证据，未连接远端前不得标记 PASS。

## 1. 版本与迁移对账

```sh
set -eu
BASE_URL=http://192.168.0.102:28080
curl -fsS "$BASE_URL/healthz"
# 需要具备只读数据库凭据后执行，禁止把密码写入脚本或日志：
psql "$DATABASE_URL" -Atc "select version, dirty from schema_migrations order by version desc limit 5"
psql "$DATABASE_URL" -Atc "select max(version) from schema_migrations"
```

验收要求：数据库版本与发布分支迁移清单一致，`dirty=false`；部署记录包含镜像 tag、commit 和回滚点。当前仓库仅核实本树与未合并分支迁移目录，未代替 102 线上证据。

## 2. 主链路端口状态

使用真实 102 账号和测试资源执行下单至激活回调（12 环节），然后查询：

```sql
select o.order_no, o.stage, o.status, p.id, p.status as port_status
from orders o join ports p on p.order_id = o.id
where o.order_no = :'ORDER_NO';
```

出口条件：订单 `stage=12,status=DONE` 且关联端口 `USED`；不得有该订单端口残留 `RESERVED`。重复调用环节12应幂等，并再次确认 `USED`。

## 3. 开放平台 102 写路径

重部署与本地 commit/迁移一致的镜像后，用 JWT 管理员执行：

```sh
BASE_URL=http://192.168.0.102:28080
curl -fsS -X POST "$BASE_URL/api/admin/v1/openplat/apps" \
  -H "Authorization: Bearer $ADMIN_JWT" -H 'Content-Type: application/json' \
  -d '{"name":"s0-sandbox","callbackUrl":"https://example.invalid/webhook"}'
# 使用返回的 app 凭据运行仓库 selftest/replay 脚本，并保存响应与投递记录
```

若仍返回旧版本校验错误，记录为 BLOCKED（代码/镜像漂移），不得伪造通过；完成重部署后补填 commit、镜像、迁移版本和 selftest/replay 响应摘要。

## 4. 已执行门禁

- `go test ./internal/domain/order`：应通过，包含 `TestPGStore_UpdateMapRepairsReservedPort`。
- 102 真实环境：待部署后执行；未执行项保持 BLOCKED。

## 5. 未完成阻塞

- 无远端数据库和发布权限时，无法安全执行迁移、重部署及真实开放平台写路径；责任人需按 §1/§3 执行并归档原始响应、SQL 输出和回滚点。
