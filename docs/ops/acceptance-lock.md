# 102 验收共享时间窗锁

## 机制

会造数并清扫的 `verify-tl1-e2e.sh`、`mainchain-acceptance.sh`、`dial-acceptance.sh` 在开始前统一获取 `/tmp/boss-102-acceptance.lock`。锁内容包含脚本标识、UTC 时间戳和 PID，TTL 为 15 分钟。

锁已被占用时脚本立即以非零退出，并输出可 grep 的 `[acceptance-lock] LOCK_BUSY holder=...`；持锁进程退出后锁会释放。进程异常退出且锁超过 TTL 时，下一实例可接管。

## 处置建议

看到 `LOCK_BUSY` 时不要删除锁或强行重跑：先记录 holder，等待当前验收正常收尾；若确认进程已死且文件已超过 15 分钟，再由新实例自动接管。不要在 102 并行运行这些脚本。

## 轻量自测

自测不访问 102 业务和数据库：

```bash
LOCK=/tmp/acceptance-lock-test.$$
ACCEPTANCE_LOCK_PATH="$LOCK" SELFTEST_SLEEP=3 scripts/ops/acceptance-lock-selftest.sh & p=$!
ACCEPTANCE_LOCK_PATH="$LOCK" scripts/ops/acceptance-lock-selftest.sh; test $? -ne 0
wait "$p"
ACCEPTANCE_LOCK_PATH="$LOCK" scripts/ops/acceptance-lock-selftest.sh
rm -f "$LOCK"
```
