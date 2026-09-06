# 102 孤儿巡检定时化(2026-08-29)

服务端已有 hourly patrol loop(`internal/app/patrol_loop.go` 落 report_snapshots),
但快照无人消费、泄漏无人拦截。定时化落地为 102 宿主 crontab,每日 08:10 跑
`scripts/ops/db-patrol-gate.sh`,任一孤儿类 >0 退出 1 并留痕日志。

## 安装(一次性,在 102 上)

```bash
ssh imeepos@192.168.0.102 'crontab -l 2>/dev/null | grep -v db-patrol-gate; \
  echo "10 8 * * * cd ~/boss && ./scripts/ops/db-patrol-gate.sh >> /tmp/boss-patrol-gate.log 2>&1"' \
  | ssh imeepos@192.168.0.102 'crontab -'
```

前提: 102 上 `~/boss` 为导出树(非 git checkout),脚本与 test-accounts.json
经 scp 放置(scripts/ops/db-patrol-gate.sh + .agents/skills/bossctl-cli/test-accounts.json),
脚本自包含仅依赖 curl+python3。

## 已安装(2026-08-29 实录)

102 crontab 已含 `10 8 * * * cd /home/imeepos/boss && ./scripts/ops/db-patrol-gate.sh
>> /tmp/boss-patrol-gate.log 2>&1`,102 本机实测 ORPHAN-GATE OK(11 项全 0)。
验证: `ssh imeepos@192.168.0.102 'crontab -l | grep patrol'`。
若脚本/密钥轮换,需同步 scp 更新 ~/boss 下的两份文件。

## 与 CI 门禁的分工

- cron(每日): 环境侧兜底,捕获非部署通道的造数(手工脚本/联调);
- deploy-102 workflow(每次部署): 部署通道门禁,泄漏进 main 即红;
- mainchain-acceptance.sh(每次验收): 造数自清理 + 收尾自检,源头不泄漏。

三者共用同一脚本 `scripts/ops/db-patrol-gate.sh`(支持 `--selftest` 离线自检)。

## 资产↔标签双向一致性(2026-08-27 追加)

internal/domain/report/pg_patrol.go 巡检清单新增两项:

- `assets.tag_id -> tags`:资产.tag_id 指向不存在或反向未绑的标签 → 孤儿数
- `tags.bound_asset_id -> assets`:标签.bound_asset_id 指向不存在或反向未绑的资产 → 孤儿数

历史 124 条 B 端孤儿由此类单边写入产生(internal/domain/asset/pg_write.go
双绑回填漏了 CreateTag 反向);DB 部分唯一约束 000158 已部署兜底,但应用层遗漏
或未来回归仍可能产生——本巡检作为每日 cron + db-patrol-gate 兜底,任一 >0 即
ORPHAN-GATE FAIL 拦截验收/部署。

## 资产数据质量两查(P4-T3,2026-09-06 追加,只暴露不修改)

db-patrol-gate.sh 在孤儿门禁之外新增两项资产数据质量暴露(固定前缀可 grep,
不影响退出码;`--asset` 可只跑这两项):

- `[db-patrol] ASSET-EPC-INVALID count=N`(+SAMPLE 前 20 行):tags.epc_code 非
  24-hex,口径=贴标待回填/待清理;物理 EPC 与实物一致,严禁程序生成重写;
- `[db-patrol] ASSET-TYPE-UNKNOWN count=N`(+BREAKDOWN):assets.type 非白名单
  ONU/ROUTER/OLT 计数,防新方言;白名单与 internal/domain/asset/type_whitelist.go
  同步维护。

SQL 通道:开发机/CI 经 ssh 到 102 容器;102 本机 cron 自连 ssh 无免密,脚本自动
回退本机 docker exec(imeepos 具备 docker 权限,已实测),两形态输出一致。
配套残留清理:SMOKE-P3-*/A-RK-E2E-001-MI-ONU-* 六行 e2e 造数残留由
`scripts/ops/clean-asset-type-residue.sh` 按「无引用才删、有引用只暴露」清理
(2026-09-06 已执行,6→0);验收脚本 `scripts/ops/verify-patrol-extended.sh`。

## 部署静默停摆巡检(deploy-guard-alert,2026-09-06 追加)

来源:deploy-runner job 镜像被 prune 后 6 次 push 零部署且无任何可见错误(ISSUE.md
CI/deploy-102)。机制分两半:

- 写入方(自动,随 deploy-102 workflow 生效):尾步 `scripts/ops/deploy-marker-write.sh`
  在容器/healthz/指纹/孤儿门禁全过后,把「最近成功部署 ts+sha」经 docker-run 写到 102
  宿主 `/home/imeepos/boss-deploy-state/last-success.env`;
- 比对方(本巡检):`scripts/ops/deploy-guard-alert.sh` 读标记算年龄,超 24h(默认,
  `DEPLOY_GUARD_THRESHOLD_HOURS` 可调)输出 `[deploy-guard] ALERT` 并 exit 1;标记
  缺失/不可读同样 ALERT。自测:`--selftest`(新鲜/超龄/缺失三态,不读真实标记)。

### 安装(2026-09-06 实录)

```bash
ssh imeepos@192.168.0.102 'crontab -l 2>/dev/null | grep -v "deploy-guard-alert"; echo "25 8 * * * cd ~/boss && ./scripts/ops/deploy-guard-alert.sh >> /tmp/deploy-guard-alert.log 2>&1 # deploy-guard-daily"' | ssh imeepos@192.168.0.102 'crontab -'
```

- 种子标记:安装日以当日已部署 sha(7a5bc656)手工落盘一次;此后由 workflow 尾步接管。
- 巡检只兜底「部署通道整体静默停摆」;单次 run 失败由 gitea Actions UI 侧负责。

## 放量守夜锚点总表(2026-09 M0 冻结)

> 对齐 `docs/plan/q4-launch-growth-plan.md` 辅线 4.1/4.2(每日轧账 + SLO 巡航)。
> 职责分工:业务态对账/补偿在服务端循环内,环境侧巡检/采集在 102 crontab。

| 锚点 | 载体 | 触发 | 脚本/实现 | 状态 |
|---|---|------|-----------|------|
| 五域每日对账 | 服务端循环 | 每日 03:00(30min 粒度补跑) | `internal/app/daily_recon_loop.go`(refType=daily_recon,幂等) | 已内建 |
| 话单补偿 | 服务端循环 | 周期轮询 | `internal/app/cdr_compensation_loop.go` | 已内建 |
| ETL 逾期自动派单 | 服务端循环 | 周期轮询 | `internal/app/etl_autodispatch_loop.go` | 已内建 |
| 孤儿巡检门禁 | 102 cron | 每日 08:10 | `db-patrol-gate.sh`(见上) | 已装 |
| 部署静默停摆巡检 | 102 cron | 每日 08:25 | `deploy-guard-alert.sh`(见下,新 2026-09) | 本批安装 |
| 备份 | 102 cron | 每日 03:30 | `/backup/jobs` | 已装 |
| Stripe 隧道守卫+告警 | 102 cron | 每 3 分钟 | `stripe-tunnel-url.sh`(refType=stripe_tunnel) | 已装 |
| **SLO 巡航采集+告警** | 102 cron | 每日 03:40 | `slo-cruise.sh`(refType=slo_cruise,新 2026-09) | 本批安装 |
| **SLO 周报** | 102 cron | 每周日 03:50 | `slo-cruise.sh --weekly` | 本批安装 |
| Gitea 清理 | 102 cron | 每日 03:00 | gitea cleanup.sh | 已装 |
| 市场对账 | 102 cron | 每月 1 日 04:30 | market-reconciliation-cron.sh | 已装 |

### SLO 巡航(slo-cruise.sh)要点

- 双模式:102 本机直接 docker exec;其它主机经 ssh(**slo-collect.sh 2026-09 起支持**,
  修复原脚本 102 本机自连 host key 失败坑)。
- 阈值:待办超 24h >1(SLO_TODO_ALERT)、近 7d 未入账话单 >100(SLO_UNBILLED_ALERT)、
  日报滞后 >8h(SLO_LAG_ALERT_SEC);超阈值经 `/ops/notify-emit`(refType=slo_cruise,
  按日幂等)推提醒中心,并写 `/tmp/slo-cruise-YYYYMM.jsonl` 留痕。
- S4 订单完成率为窗口口径,当前只上报数值不自动告警(等 M1 轧账口径细化,防误报)。
- 告警发出后 exit 1;采集/留痕失败发 URGENT 且 exit 1,禁止静默。

### 安装(2026-09 M0,须先 scp 新脚本到 ~/boss)

```bash
scp scripts/ops/slo-collect.sh scripts/ops/slo-cruise.sh imeepos@192.168.0.102:~/boss/scripts/ops/
ssh imeepos@192.168.0.102 'crontab -l 2>/dev/null | grep -v "slo-cruise"; \
  echo "40 3 * * * cd ~/boss && ./scripts/ops/slo-cruise.sh >> /tmp/slo-cruise.log 2>&1 # slo-cruise-daily"; \
  echo "50 3 * * 0 cd ~/boss && ./scripts/ops/slo-cruise.sh --weekly >> /tmp/slo-cruise-weekly.log 2>&1 # slo-cruise-weekly"' \
  | ssh imeepos@192.168.0.102 'crontab -'
```

### 应急口径

- 巡航 WARN 提示待办积压/话单残留/报表滞后 → 先查对应补偿任务中心队列,再查
  `/tmp/slo-cruise-YYYYMM.jsonl` 该日上下文;连续 3 天同一 WARN 且无人工处置 →
  升级 URGENT(值班红线)。
- `ops/notify-emit` refType 白名单在 `internal/httpapi/admin/ops_notify.go`
  (stripe_webhook_guard / stripe_tunnel / slo_cruise),新增脚本 refType 必须先登记。
