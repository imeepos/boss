# 时间与时区审计报告（2026-08-21）

> 范围：migrations 全量时间列、DB 会话时区、Go 时间处理、前端展示链路。
> 方法：静态 grep 全仓 + pgx 直连 102 生产库实测（SHOW TimeZone / now() 回扫类型 / information_schema 列类型统计）。

## 0. 实测事实（权威）

| 项 | 实测值 | 结论 |
|:---|:------|:-----|
| DB `TimeZone` / `log_timezone` | 均为 UTC | DB 侧无时区配置问题 |
| 时间列类型 | 106/106 全部 `timestamptz`，0 个裸 `timestamp`、0 个 `DATE` | 迁移层无类型缺陷 |
| DSN | 未设置 `TimeZone` 参数 | 会话时区随服务器默认（UTC） |
| pgx 回扫 `now()` | 带进程本地时区偏移（测试机 `-0700 PDT`） | **pgx v5 把 timestamptz 解码为 Go 进程本地时区的 time.Time**，不是 UTC |
| boss-server 容器 | compose 未设 `TZ`，默认 UTC | 生产进程时区=UTC，与 DB 一致 |

## 1. 现状结论

当前"能跑对"完全依赖一个隐式巧合：**DB 会话时区 = UTC = Go 进程时区**。
任何一处单方面改变（给容器设 TZ、DSN 加 TimeZone、换机器部署）都会引发整类日期错位。
且业务面向菲律宾（UTC+8），所有"自然日/自然月"边界实际在马尼拉 08:00 翻转，属潜伏业务缺陷。

## 2. 问题清单

### P0 显示错位（用户可见）

1. **admin 前端两套时间格式化并存**：
   - `web/admin/src/lib/format.ts fmtTime`：纯字符串截断，**不转本地时区** → 菲律宾用户看到 UTC 时间，慢 8 小时；
   - `web/admin/src/pages/boss/message/logic.ts fmtTime`：`new Date()` 转浏览器本地时区。
   同一 admin 内行为不一致，且前者明显错误。
2. `internal/httpapi/admin/dashboard.go:130` 告警时间 `al.CreatedAt.Local().Format("15:04")` —— "Local" 指服务器时区而非用户时区，当前=UTC，马尼拉用户看到错位的钟点。

### P1 边界依赖隐式时区（改环境即碎）

3. `internal/httpapi/admin/billing.go:181` 对账日期 `time.ParseInLocation(..., time.Local)` —— 依赖进程时区；与 DB 侧 `date_trunc('day', $1::timestamptz)`（依赖**会话**时区 UTC）是两套时区来源，当前靠"都是 UTC"巧合对齐。
4. `internal/domain/order/pg_sub.go:151` SLA 截止 `ParseInLocation(..., time.Local)` 同上。
5. `internal/httpapi/admin/dashboard.go:153` 注释声称"DB 时间戳按 UTC 扫描"——**实测不成立**（pgx 回扫是进程本地时区）；sameDay 归一逻辑本身安全（两侧同为回扫值），但注释错误会误导后人。

### P2 自然日/月边界 = UTC 边界（马尼拉 08:00 翻转，业务语义错误）

6. 订单号 `to_char(now(),'YYYYMMDD')`（`order/pg.go:117`）——单号日期按 UTC 翻日。
7. 对账批次号 `PC-YYYYMMDD`（`recon_pull.go:116`）+ `loadDailyPayments` 的 `date_trunc('day')`（`pg_recon.go:134`）——对账"当日"范围按 UTC 切。
8. 师傅绩效月 `time.Now().Format("2006-01")`（`worker/profile.go:39,61,113`）——月初 8 小时归属错误。
9. 合约到期月 `to_char(effective_at + interval '24 months','YYYY-MM')`（migration 000061）——按 UTC 定月。
10. dashboard 7 日趋势 `now.AddDate(0,0,-i).Format("01-02")`（`dashboard.go:140`）——天分组按进程时区（当前 UTC）。
11. Stripe 日流水 `ListDayStatements` 注释明示按 UTC 自然日——与渠道对账口径(UTC)一致时无问题，但须与 P2 统一时显式声明。

### P3 其他

12. `notify/pg.go:173`、`attachment/minio.go:72`、`sms`/`realid` 签名时间戳等已显式 UTC —— 正确，作为范本。
13. worker 端 `misc.go/profile.go/ticket.go` 多处 `Format("01-02 15:04")`：回扫值为进程时区(UTC)，师傅端手机上看 UTC 钟点 —— 属 P0 同类显示错位（App 侧未经转换的字符串直出）。

## 3. 统一规划（防患未然）

### 原则（建议裁定后写 adopted note）

1. **存储层**：一律 TIMESTAMPTZ（已满足，106/106），DB/会话/容器时区固定 UTC（现状即如此，冻结不变，禁止给容器加 TZ）。
2. **绝对时刻传递**：API JSON 一律 RFC3339 带偏移（Go time.Time 默认行为，已满足）。
3. **展示时区**：用户侧时间一律由**前端/客户端转用户时区**后显示；后端禁止 `Format` 出"墙钟字符串"给三端 UI（发票 PDF 等单据属例外，单据时区随业务裁定）。
4. **业务日边界**：凡"自然日/自然月"（订单号日期、对账批次、绩效月、趋势图分组、合约月）统一引入 `internal/pkg/httpx` 或新 `internal/pkg/clock` 的单一业务时区常量（建议 `Asia/Manila`，配置中心可覆写），禁止散落 `time.Local`/裸 `now()` 格式化日期。

### 落地步骤（按优先级）

| # | 动作 | 位置 |
|:--|:-----|:-----|
| 1 | 统一前端 fmtTime：`new Date(iso)` 转浏览器本地 + 全仓替换字符串截断版 | web/admin format.ts、各端同类 |
| 2 | 新建 `internal/pkg/clock`：`BusinessTZ` 常量 + `Now()/BusinessDay(t)/BusinessMonth(t)` | internal/pkg |
| 3 | 替换 P2 全部裸日期边界为 clock 包口径；DB 侧 `date_trunc` 改 `date_trunc('day', $1 AT TIME ZONE 'Asia/Manila')` 型或由 Go 传入已切界的 [start,end) | pg.go/recon_pull/pg_recon/profile/dashboard |
| 4 | 清除 `time.Local`：billing.go / pg_sub.go 改 ParseInLocation(BusinessTZ) | httpapi/admin、order |
| 5 | 修正 dashboard.go:153 错误注释 | dashboard.go |
| 6 | 裁定后写 dated note 到 docs/notes/adopted/（业务时区=Asia/Manila、存储=UTC、展示=客户端） | docs/notes/adopted |

> 本报告只记录现状与方案，未做任何代码变更；P0/P1/P2 修复属行为变更，须分独立提交（fix 带回归测试）。
