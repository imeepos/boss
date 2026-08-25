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

前提: 102 上有 boss 仓库 checkout(含脚本与 test-accounts.json)。
安装后验证 `ssh imeepos@192.168.0.102 'crontab -l | grep patrol'`。

## 与 CI 门禁的分工

- cron(每日): 环境侧兜底,捕获非部署通道的造数(手工脚本/联调);
- deploy-102 workflow(每次部署): 部署通道门禁,泄漏进 main 即红;
- mainchain-acceptance.sh(每次验收): 造数自清理 + 收尾自检,源头不泄漏。

三者共用同一脚本 `scripts/ops/db-patrol-gate.sh`(支持 `--selftest` 离线自检)。
