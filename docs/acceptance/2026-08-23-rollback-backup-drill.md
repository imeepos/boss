# 迁移回滚与备份恢复演练记录 2026-08-23

> Q1 交付物:验证"发布具备可验证回滚路径"。环境 = 102 生产部署库。

## 演练内容

工具(随本记录同提交):

- `scripts/ops/backup-102.sh` —— pg_dump -Fc + pg_restore --list 完整性校验 + sha256 + 保留 14 份
- `scripts/ops/migrate-102.sh` —— status / up [N] / down <VERSION>(down 前强制备份)

## 过程与结果

| 步骤 | 命令 | 结果 |
|------|------|------|
| 备份 | backup-102.sh | OK boss_20260823_035255.dump (961K) |
| 应用待迁移 | migrate-102.sh up | apply 000112_invoice_tax_events + 000113_payment_refund |
| 回滚 | migrate-102.sh down 000111_cdr_kafka_status | 自动备份(035312.dump)→ revert 000113 → revert 000112 |
| 重放 | migrate-102.sh up | 两个迁移重新应用,与下次部署等价 |
| 恢复演练 | 备份恢复到临时库 boss_restore_drill | pg_restore 成功,orders=151 可读,临时库已清理 |

## 结论

- 迁移 up/down/up 闭环与记账(schema_migrations)一致,发布前可回滚、回滚后可重放。
- 备份可恢复、可读,恢复演练用时 <1min(961K 小库;大库需重测并记录时长)。
- 值班例行/事故动作固化于 docs/deploy/oncall-102.md。
