# 2026-09-03 order_stages.finished_at 写入口径 + 历史回填裁定(T17)

> 任务: W-0906-T17(订单时间轴环节完成时间数据源头修复)。
> 背景: 102 order_stages 297 行,finished_at NULL 292 行,仅 5 行非空
> (全部为 2026-08-25 审计自愈 UPDATE 产物)。前端降级文案已由 T16 交付。

## 根因(实测复核)

全仓库唯一的 finished_at 写路径是 CheckResource 自愈 UPDATE
(`UPDATE order_stages SET result='DONE', finished_at=now() ... result='PENDING'`,2026-08-25 审计 §2.3.2 补)。
推进原语 appendStage 的 INSERT 只落 (order_id, stage, result),从不写 finished_at——
advance 状态机(环节3~12)/扫码绑定/自动化续推等全部推进路径经此原语,DONE 行全漏完成时间。

## 裁定一:写入口径(推进成功即写,失败/等待不写)

- appendStage 对 result='DONE'(推进成功)落 finished_at=now(),与自愈 UPDATE 同源;
  PENDING/DOING(等待/失败/进行中)保持 NULL——不再有"推进成功但无完成时间"。
- Submit 环节1 下单成功即落 DONE + finished_at(≈ created_at):下单即完成,时间轴环节1 不再永久未完成。
- memory store 同步口径(PG 单事实源,内存仅测试参考)。

## 裁定二:历史回填边界(仅权威痕迹可推导者回填,其余保持 NULL)

| 环节 | 权威痕迹 | 102 可推导行数 |
|:-----|:---------|:--------------|
| 1 下单 | orders.created_at(下单即建单,权威) | 82 |
| 2 资源核查 | 无带时间戳日志 | 0(5 行自愈已填) |
| 3 端口预占 | reserve_records.created_at / port_change_history.changed_at | 0(两表 102 空) |
| 4 合同收费 | payments.created_at(经 bill 关联,非订单直连,不精确) | 0 |
| 5 标签预绑定 | quad_links 无 created_at 列 | 0 |
| 6 创建账号 | lo_accounts 无 created_at 列 | 0 |
| 7 预下发配置 | provision_tasks.created_at(stage_event='preConfigOLT') | 4 |
| 8 派单 | dispatch_tickets 无 created_at(仅 arrived_at) | 0 |
| 9 扫码绑定 | scan_logs 无 created_at 列 | 0 |
| 10 激活 | 无痕迹表 | 0 |
| 11 激活回调 | activation_callbacks 无 created_at 列 | 0 |
| 12 更新GIS | port_change_history USED 102 空 | 0 |

合计可回填 86 行;其余 206 行保持 NULL(前端 T16 降级文案覆盖)。
回填脚本 scripts/ops/stage-finished-at-backfill.sql:全部 UPDATE 以 finished_at IS NULL
守卫,单事务(BEGIN/COMMIT),幂等可重复执行;已在 102 实际执行并记录行数。

## 否决项

- 用 now() 猜填全部 NULL(时间轴"完成时间"会伪造成部署时刻,不可信);
- 给 scan_logs/quad_links/activation_callbacks 补 created_at 列再回填(迁移+改写入方,收益低于成本,留契约后续);
- 把订单列表 DONE 单的 finishedAt 改为以 orders 终态时间派生(读路径假象,掩盖数据源头缺失)。
