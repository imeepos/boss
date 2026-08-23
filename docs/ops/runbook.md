# 运维手册(稳定版 1.0)

> 定位:日常运维的入口枢纽。部署细节见 `docs/deploy/production-cluster.md`,故障排查见 `docs/deploy/troubleshooting.md`,新环境初始化见 `docs/deploy/new-env-init-checklist.md`,本文不重复,只做索引+日常例行动作。

## 1. 环境清单

| 环境 | 地址 | 用途 |
|:-----|:-----|:-----|
| 102 联调/预发 | `http://192.168.0.102:28080`(ssh `imeepos@192.168.0.102`) | CI 自动部署,验收/压测/演练场地 |
| 客户生产 | 按交付清单 | 稳定版 1.0 部署 |

关键容器:`boss-server`(API,`/healthz` `/metrics`)、`boss-admin-web`、`boss-aaa`、`boss-report`、`boss-infra-postgres-1`、`boss-apisix`、监控栈(prometheus/grafana/loki/alertmanager)。

## 2. 例行动作

| 频率 | 动作 | 命令/入口 |
|:-----|:-----|:----------|
| 每日 | 检查健康与告警 | Grafana 看板;`curl :28080/healthz` |
| 每日 | SLO 采集留档 | `scripts/ops/slo-collect.sh`(结果归 `docs/ops/` 周报) |
| 每日低峰 | 数据库备份 | 102 crontab 03:30 自动调 `POST /api/admin/v1/backup/jobs`;恢复流程见 `dr-drill.md` §1、`POST /backup/restore` |
| 每周 | 契约对账复跑 | `node scripts/ops/contract-probe.mjs`(退出码非 0 即有漂移) |
| 每列车 | 发布后验证 | `release-train.md` §7 七项清单 |
| 每列车 | 主链路验收 | `scripts/ops/mainchain-acceptance.sh RUNS=10` |

## 3. 发布与回滚

- 发车流程、灰度、回滚触发条件:`docs/ops/release-train.md`(权威)。
- 回滚要点:镜像按版本保留 ≥3 个;迁移 down 仅限演练环境,生产回滚迁移须单独评审。
- 演练先例与 RTO 记录:`docs/ops/dr-drill.md`。

## 4. 监控与 SLO

- 指标定义与错误预算:`docs/ops/slo.md`;基线:`docs/ops/slo-baseline-102.md`。
- 服务端指标端点:`/metrics`(RED);存活:`/healthz`。
- 错误预算耗尽 50%/100% 的处置动作见 `slo.md` §4(降速/冻结)。

## 5. 安全例行

- API key 生命周期:`bossctl apikey list/revoke`;密钥泄漏立即 revoke(明文不可找回,只能重建)。
- 停用主体即停用其全部 key;worker/customer 密钥恒无菜单权限。
- 备份文件与 `deployments/app.env` 权限 0600,不入库。

## 6. 升级路径

- 迁移编号规则与撞号处置:AGENTS.md「迁移编号规则」+ `check-contract-sync` D 项。
- 版本发布记录:发布说明随发布提交,changelog 引用迁移号区间。
