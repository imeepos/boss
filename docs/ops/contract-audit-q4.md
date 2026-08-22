# Q4 全量 API 契约对账报告

> 版本 V1.0｜对账时间：2026-08-26｜契约源：`api/openapi/{admin,user,worker}/*.yaml`｜实测环境：102 `http://192.168.0.102:28080`
> 方法：`scripts/ops/contract-probe.mjs` 逐条探测契约路由（444 条），非 404 即判定已部署（401/403/4xx 均证明路由存在）。
> 代码侧路由↔契约一致性另由 `make check` 的 check-contract-sync（A 检查）机械门禁兜底。

## 1. 结论

444 条契约路由中 442 条在 102 环境实测已部署（99.5%）。发现 2 个工具级缺陷（已修复：user 端前缀硬编码、path key 引号解析）与 2 条真实契约差异（契约超前实现）。

## 2. 发现与处置

### 2.1 工具错误一：user 端前缀硬编码（已修复）

- 症状：bossctl 路由生成器把 user 端前缀写死为 `/api/v1`，实测 404（85 条误报）。
- 事实：契约 `api/openapi/user.yaml` servers 为 `/api/user/v1`，服务端 `internal/httpapi/user/auth.go` 挂载同前缀，二者一致；错在 `scripts/gen-bossctl-routes.mjs`。
- 处置：改为 `/api/user/v1` 并重新生成 `cmd/bossctl/routes_*.go`（admin 303 / user 81 / worker 60 条）。bossctl 二进制需随下个发布重建。

### 2.2 工具错误二：带引号的 path key 解析丢失（已修复）

- 症状：`geo.yaml` 两处 path key 用引号形式（`'/geo/countries/{code}/names/{locale}/{nameType}'`），解析器正则只认无引号形式，其下的 delete 方法被误挂到相邻 2 段路径，产生 2 条假差异。
- 处置：`gen-bossctl-routes.mjs` 与 `contract-probe.mjs` 的正则兼容 `'?\//`，重新生成后 geo 两条 DELETE 正确入册。

### 2.3 契约超前实现（D 类：契约有、代码无，真实差异）

| # | 路由 | 现状 | 处置建议 |
|:-:|:-----|:-----|:---------|
| Q4-1 | admin `POST /customer-registrations` | 代码仅有 GET 列表 + approve/reject，无创建 | 二选一：补创建实现，或契约降级为「由 user 端提交」 |
| Q4-2 | admin `POST /worker-registrations` | 同上 | 同上（师傅注册走 worker 端扫码，admin 创建或非必要） |

## 3. 销项状态

- [x] 工具前缀错误（随本报告提交修复）
- [x] 引号 path key 解析（随本报告提交修复）
- [ ] Q4-1/Q4-2 契约与实现对齐（进 backlog，销项后更新本表）

## 4. 复跑方式

```bash
node scripts/ops/contract-probe.mjs            # 默认打 102
node scripts/ops/contract-probe.mjs --base http://...   # 其他环境
# 退出码 0 = 全部命中；非 0 = 存在 missing/errors（JSON 明细输出）
```
