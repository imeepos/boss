# 发布说明:v1.0.0-rc1(生产稳定版候选)

> 发布列车第一个正式版本标。按 release-train.md §2 门禁核验后打 tag;正式 1.0 待客户终验(SLO 7 天数据 + 签收)后发布。

## 版本内容

- 代码:main @ 30b781a(含 Q1-Q4 全部已合并能力)
- 迁移:000001–000116(无破坏性迁移;000112/000113 曾撞号,已按让号规则消除)
- 契约:api/openapi 三端,check-contract-sync A/B/C/D 全绿(394 条路由)

## 发车门禁核验(release-train.md §2)

| 门禁 | 结果 | 证据 |
|:-----|:-----|:-----|
| make check(contract-sync) | 绿 | A/B/C/D OK,394 条路由 |
| Go 全量测试 | 绿 | `go test ./...` exit 0(2026-08-26) |
| 契约对账无新增红项 | 绿 | contract-audit-q4.md;部署滞后项随 CI 追平 |
| SLO 错误预算 | 绿 | 监控修复后刚开始积累,无超支 |
| 本发布说明 | 本文件 | — |

## 附带报告(验收包引用)

契约对账 `contract-audit-q4.md`|容量压测 `load-test-report.md`(204rps 安全容量)|灾备迁移演练 `dr-drill.md`|SLO 定义/基线 `slo.md`/`slo-baseline-102.md`

## 发布后验证清单(§7)

- [ ] /healthz 200、/metrics 可采、Grafana boss-slo 看板有数据
- [ ] 主链路验收 10/10(mainchain-acceptance.sh)
- [ ] contract-probe.mjs 全命中(等 CI 部署追平 main)
- [ ] SLO 看板 24h 无红项

## 已知事项

- CI 部署滞后 main 数个合并(镜像迁移号 000109<000116),rc1 全量部署后复跑探针
- 客户环境首周需复采 S4/S5 真实基线
