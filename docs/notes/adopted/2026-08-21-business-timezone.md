# 2026-08-21 业务时区裁定:Asia/Manila + 存储UTC + 展示归客户端

## 裁定

1. **业务时区 = Asia/Manila**（UTC+8，无夏令时）。依据配置证据：Stripe 记账币种 `php`（config.go 默认）、geo 域币种注释 `'PHP'` 且 migration 000038 国家表 PH 排首、库内已载入 43,869 条菲律宾 PSGC 行政区划、需求全案为"菲律宾 BOSS"。配置项 `BOSS_TIMEZONE`（默认 `Asia/Manila`）可覆写。
2. **存储/DB 会话/容器时区固定 UTC**：全部时间列为 TIMESTAMPTZ（106/106），DSN 不设 TimeZone，容器不设 TZ——冻结不变。
3. **API 传递绝对时刻**：RFC3339 带偏移。
4. **展示时区归客户端**：前端 `new Date()` 转浏览器本地；后端需要墙钟字符串处一律 `In(clock.Location())`。
5. **自然日/月边界统一走 `internal/pkg/clock`**（`Now/Location/DayBounds`），禁止 `time.Local`、禁止裸 `now()`/`time.Now()` 格式化日期。

## 放弃了什么

- **放弃"进程时区=业务时区"方案**（给容器设 TZ=Asia/Manila）：会让 timestamptz 回扫墙钟随进程变，DB 会话与 Go 双侧口径分裂，且同一二进制跑多时区场景无解。
- **放弃 DB 侧 date_trunc 切日**：date_trunc 依赖会话时区，与批次号/Go 侧口径分叉；改由 Go `clock.DayBounds` 切好 `[start,end)` 传参。
- **放弃裸 `timestamp`/`DATE` 列**：现库 0 个，维持禁令。

## 遗留（记录不阻塞）

- migration 000061 的 `contract_end` 回填按 UTC 定月（已应用的迁移不可改，影响仅历史一次性回填数据）。
- Stripe 日流水对账口径为 UTC 自然日（渠道侧事实），与系统侧业务日对账的差异在对账报告中显式标注即可。
