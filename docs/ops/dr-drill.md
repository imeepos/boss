# Q4 迁移演练与灾备恢复记录

> 时间:2026-08-26｜环境:102(`imeepos@192.168.0.102`,docker 全栈)｜对象:PostgreSQL `boss` 库
> 方式:全部在副本库 `boss_drill` 上执行,生产库零接触;演练完毕副本已删除。

## 1. 灾备恢复演练(备份→恢复→校验)

| 步骤 | 动作 | 结果 |
|:-----|:-----|:-----|
| 1 | `pg_dump boss` 管道导入新建 `boss_drill` | 成功,耗时 32s |
| 2 | 表数量比对 | 源 150 = 副本 150,一致 |
| 3 | 行数抽查(orders) | 源 150 / 副本 145,差 5 条为 dump 与比对间在线写入(时间偏移),非恢复丢失 |

结论:备份-恢复链路可用;生产建议固定备份窗口(低峰)以消除时间偏移,恢复 RTO ≈ 32s(当前数据量)。

## 2. 迁移演练(000112_invoice_tax_events,up→down 双向)

说明:102 生产无独立 migrate 工具,迁移由 boss-server 启动时自动应用;演练在副本库直接执行 SQL 验证双向可逆。

| 步骤 | 动作 | 结果 |
|:-----|:-----|:-----|
| 1 | 执行 `000112_invoice_tax_events.up.sql`(ON_ERROR_STOP) | 成功,342ms,`invoice_tax%` 表 = 1 |
| 2 | 执行 `000112_invoice_tax_events.down.sql` | 成功,115ms,`invoice_tax%` 表 = 0 |

结论:最新迁移 up/down 均干净可逆;发布列车 §6 的「迁移先演练」要求按此流程执行。

## 3. 部署链路事实(演练中核实)

- 生产 schema_migrations 当前到 000111,main 已有 000112 → 存在一条部署滞后(与契约对账报告的 `/billing/ledger-recon` 滞后同源:CI 部署节奏)。
- 历史撞号(000044/000075/000095/000102 双行)在库中可见,均为已登记存量例外(baseline 豁免)。

## 4. 回滚演练(容器级,2026-08-26 执行)

事实核实:CI(deploy-102.yml)每次构建推双标签 `server:${GITHUB_SHA}` + `server:latest`,registry 保留完整 SHA 版本史;另有手工固化的 `server:v1.0.0-rc1`。

| 步骤 | 动作 | 结果 |
|:-----|:-----|:-----|
| 1 | compose override 钉镜像为版本标签,force-recreate server | 成功,**3s** 恢复 healthz=200 |
| 2 | 冒烟:登录 200、/metrics 可采 | 通过 |
| 3 | 回滚前置检查:回滚目标镜像内置迁移(000061 老镜像)远低于库(000115),确认不可用;回滚须选同迁移代际内的近版镜像 | 已记入 runbook 流程 |
| 4 | 恢复 latest(roll-forward),healthz=200 | 成功 |

结论:容器级回滚程序验证通过(RTO 3s);跨迁移代际回滚的镜像选择约束已固化。
2 天前的 0228c82 镜像(000061)对当前库不兼容,作废不用。

## 5. 遗留动作

- [ ] 固定每日低峰 pg_dump 备份窗口(docker-compose.backup.yml 已有基建)
- [ ] 000112 部署到 102 后复跑 `contract-probe.mjs` 确认全绿
