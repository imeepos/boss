# TL1 E2E 证据链修复 runbook(record 留痕断裂/清理 SQL/退出码假绿)

> 背景:2026-09-04 失败轮 prefix=acc_tl1_1788458558(订单 ORD-20260904-000640,
> 任务 260/261,模板 179,资源 529)。本文记录三问题根因、修复与处置留痕,
> 供复跑 scripts/verify-tl1-e2e.sh 前对照。参考格式:docs/ops/tl1-driver-cutover.md。

## 1. R1 设备侧证据断裂(正向 SUCCESS 但 tl1sim record 空)

- 现象:provision_logs result=SUCCESS(driver=tl1),而
  /tmp/verify-tl1-e2e-1788458558/tl1sim-positive-a.jsonl 为空。
- 根因(102 现场取证,双层):
  1. 脚本 mkdir -p 只在本机执行,102 上 record 父目录不存在;tl1sim 的
     os.OpenFile(O_CREATE)不建父目录,sim.New 报错走 log.Fatalf,进程启动即死
     (/tmp/tl1sim-mainchain.log 见 START FAILED: open ...: no such file or directory);
  2. 演练端口 13027 同时被一个无 pidfile 的野 tl1sim 实例持续占位(Sep03 起的
     cli 调试遗留,-record /tmp/tl1sim-cli.jsonl),provisioner 连到它完成任务,
     指令留痕全部落在野实例的 record——成功与证据脱钩。
- 排除项:102 上 /home/imeepos/bin/tl1sim 与仓库 cmd/tl1sim 行为一致
  (-record/-deny-next/-delay-ms 旗标齐全),非陈旧二进制,无需替换。
- 修复(scripts/verify-tl1-e2e.sh):
  - start_sim 先在 102 mkdir -p record 目录再拉起 sim;
  - 启动后断言 pid 存活且 ss -tlnpt 端口 LISTEN 归属本 pid(防顶替答话);
  - precheck 起自检 sim,102 本机 /dev/tcp 发标准 LOGIN 帧,验证 record 真落盘,
    证据链不通即早失败,拒绝注入任务。
- 野实例处置留痕:pid 2630952(cmdline 含 tl1sim -addr 0.0.0.0:13027,record 为
  cli 调试路径且停机前仅 provisioner 心跳在写)于 2026-09-04 E2E 修复窗口内处置:
  经同工作区 5 个并行会话协调(2 轮带等待问询+4 路信使送达,均无回复)按预先
  声明的兜底语义(SIGTERM 优雅停,可随时重启)停机,其 record(/tmp/tl1sim-cli.jsonl)
  含失败轮 LOGIN/ADD-ONU/ADD-PONVLAN 各条,已闭环证据链;处置过程全程留痕本文。
  处置红线延续:只动协调确认无人认领的孤儿,pidfile+cmdline 双重校验不碰活人进程。

## 2. R2 清理段 SQL 错误归零

- 三处实测错:1) notifications.ref_id 为 text,与任务 id(bigint)直接比较报错;
  2) orders 删除被 order_stages 外键挡;3) provision_templates 被 provision_tasks 挡。
  2)/3) 与 1) 同串连锁:psql ON_ERROR_STOP=1 下首错即中止多语句串,后续清理全没跑。
- 修复口径:
  - ref_id 按文本比较(ref_id IN (SELECT id::text FROM provision_tasks ...));
  - orders 外键子表共 8 张,全数先删:order_stages/dispatch_tickets/scan_logs/
    activation_callbacks/complaints/dismantles/install_logs/partner_commission_ledger;
  - 模板先删任务链(logs→tasks)再删自身,另防 offer_provision_bindings;
  - 资源路:pon_onu_alloc/resource_assignments/ports/parent_id 子资源→resources;
  - 清理从按本轮 ID 改为 acc_tl1_ 前缀全量清扫(资源 code LIKE OLT-acc_tl1_%、
    模板 code LIKE TPL-acc_tl1_% 反查订单/任务集合),历史遗留一并归零;
  - 多语句打包改逐条独立执行(sqln),失败输出 FAIL 行计非零,不中断收尾。
- 复核口径(A2):清扫后 resources/templates 及其级联 tasks/logs/orders/notifications
  按 acc_tl1_ 前缀计数全为 0,脚本 assert_clean 硬断言。

## 3. R3 退出码保真

- 实测:positive-a 断言失败后续场景未跑,进程仍 exit 0——验收假绿。
- 修复:EXIT trap 把原始退出码传参入 cleanup;cleanup 内禁用 fail(不再二次 exit),
  清理段自身故障置 1,原始失败码优先透传:失败必非零,成功且清扫复核通过才 0。

## 4. 复跑指引

```bash
bash scripts/verify-tl1-e2e.sh   # 默认 http://192.168.0.102:28080
# 前置:boss-provisioner running 且 driver=tl1;provision_nms 法人 1 无表行;
#       队列 PENDING/DOING=0;演练端口(provisioner TL1_ADDR 端口段)无他人 LISTEN。
# 负向自证:ADMIN_API_KEY=写错的 key 重跑,进程须非零退出(验证后不留任何配置)。
```
