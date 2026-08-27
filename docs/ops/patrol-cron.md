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
