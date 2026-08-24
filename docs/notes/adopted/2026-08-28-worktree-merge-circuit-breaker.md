# 2026-08-28 worktree 合并协议止损线

## 背景

2026-08-24 生产验证收尾时段,多个并行会话(worker-android round3、cms-posts 等)以分钟级频率向 main 推送。
单次收尾(merge main → 门禁 → ff-merge → push)需数分钟,三次在 ff-merge 或 push 环节因 main 前进而失败,
同一分支连续 re-sync 3 次。协议原文只规定"失败回 worktree rebase/merge 后重试",未设上限:
极端情况下会活锁——每次重试都被下一次并行推送打断,收尾永不能完成,且每次重跑全量门禁成本不低。

## 裁定

在合并协议收尾步骤增加止损线:

- 同一 feature 分支连续 re-sync(merge main + 门禁 + ff 尝试)**达 3 次仍未合入**时,停止第 4 次自动重试。
- 止损后动作:① `git push gitea <分支>`(commit 安全已在远端);② 检查 `git log` 确认冲突来源会话是否仍在活跃推送
  (`docker logs gitea-runner` / 最近 main 提交时间);③ 与活跃会话错峰(等其收尾完成后单次 re-sync 合入),
  而不是与其竞速。
- 判断依据:重试失败原因均为 "diverging branches"/non-ff,而非冲突未解决;真正的代码冲突不属于止损线范畴,
  仍按协议在 feature 侧消化。

## 放弃了什么

- 放弃"无限重试直到合入":并行度高的时段这会退化为活锁+重复门禁开销。
- 不引入中央锁文件/合入队列:当前并行会话数(≤4)与频率下,错峰的成本低于引入协调机制的复杂度;
  若并行度继续上升再议(本文可被 Amended)。

## 关联

- 协议全文:docs/notes/adopted/2026-08-22-worktree-merge-protocol.md(本文是其增补,不推翻原文)
- 实例:2026-08-24 fix/loop-stagger-dashboard-isolation 收尾,3 次 re-sync 后第四次成功(未触发止损,但已到边界)。
