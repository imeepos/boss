# oltsim telnet 旧链路回归验收(verify-oltsim-provision-e2e.sh)

## 目的

固化 2026-09-01 验证过的旧配置下发链路 `telnet -> cmd/oltsim -> provision_tasks` 的可复跑验收。
本期 TL1 改动(commit cbdddb22 起)把 provisioner 执行器选择从「OLT_ADDR 配置即走 telnet」
改为「显式 `BOSS_PROVISION_DRIVER` 开关(缺省 log 桩)」,本脚本是防止旧链路被再次静默破坏的回归门。
只回归旧 telnet 链路,不验证 TL1(TL1 主链路验收见 scripts/verify-tl1-e2e.sh)。

## 定位边界(2026-09-03 TL1 主链路切换后)

- 102 compose(deployments/docker-compose.102.app.yml)provisioner 已默认
  `BOSS_PROVISION_DRIVER=tl1`(TL1_* 指向 tl1sim 演练地址),TL1 是主链路。
- 本脚本自此仅是 **legacy telnet 回归门**:driver 非 telnet 时前置直接拒绝运行
  (拒绝与 TL1 主链路混用),不再以 WARN+断言失败的方式"暴露回归"——切净后的
  稳态是 tl1,继续跑只会制造混用事故。
- 如需跑本回归:按 `docs/ops/tl1-driver-cutover.md` §7 队列排空后临时切回
  telnet(回读启动日志),跑完按同文档 §4 切回 tl1。
- 本脚本全程不停/不换已部署 provisioner,不启隔离 provisioner 容器
  (与 TL1 验收脚本互斥:任一脚本运行期间不得启动另一链路的 provisioner)。

## 用法

```bash
scripts/verify-oltsim-provision-e2e.sh [BASE_URL]   # 默认 http://192.168.0.102:28080
# 可覆盖: OLTSIM_HOST / ADMIN_API_KEY / USER_API_KEY / OLTSIM_HTTP_PORT(23333) /
#         OLTSIM_TELNET_PORT(2323) / OLTSIM_EVID_DIR(默认 /tmp/verify-oltsim-e2e-<stamp>)
```

## 流程与断言

1. 前置: boss-provisioner running、无其他 provisioner 容器并跑(防 TL1 临时容器抢队列)、
   PENDING/DOING=0、boss-oltsim systemd active、telnet 2323 有 login 提示、/records 可读。
2. 夹具: acc_oltsim_<stamp> 资源/端口/模板(admin API,端口取 data.portId)+ 用户订单
   (productId=101/addressId=290/channelId=102)+ reserve;预留端口若非夹具端口则快照原值,
   清理时精确还原。
3. 驱动: 预建 PENDING preConfigOLT 任务(task_no=PRV-O<orderID>,lo 用订单客户既有 LO,
   template 用夹具模板;CreateTask 同 task_no 幂等复用保证隔离模板必达设备)后 charge 触发
   环节 7 自动化,由「既有」boss-provisioner 领取执行——不停它、不改它配置。
4. 断言: task=DONE、provision_logs result=SUCCESS、oltsim /records 出现本单 apply
   (taskNo=PRV-O<orderID>, event=preConfigOLT, result=OK, template=夹具模板)+ journal
   "apply ok" 留痕(尽力而为)。
5. 驱动门(2026-09-03 起硬化): precheck 读容器 env,driver 非 telnet 直接拒绝运行
   (仅 legacy telnet 回归,拒绝与 TL1 主链路混用;处置路径见上文「定位边界」)。
6. 清理: 按实际 ID 精确删除 订单/任务/日志/资源/端口/模板/order_stages/dispatch_tickets/
   scan_logs/activation_callbacks/admin_notifications,复核逐表归零+队列空+provisioner 仍
   running;失败路径 trap 恢复现场。oltsim 内存台账是服务自身留痕,按设计保留作证据。

## 2026-09-02 实跑结果(102):FAIL——旧链路确被本期改动破坏

- exit code 1;夹具 resource=524 port=805 template=175,订单 ORD-20260902-000629(id 663),
  预建任务 251(isolated 模式,loid=98)。
- task=251 DONE + provision_logs=SUCCESS,但 oltsim 60s 未收到 apply → FAIL:
  `provisioner: exec task 251 (template=175, noop)`(docker logs)——运行中的 boss-provisioner
  是 log 桩:容器 env 只有 OLT_ADDR/USER/PASS,无 `BOSS_PROVISION_DRIVER`,新代码缺省 log。
- oltsim journal 最后一条 apply ok 为 Sep 01 22:56:48(PRV-O637);新 provisioner 容器 Sep 02
  12:03(UTC)起以 log 桩运行——旧链路自该时点静默失效。
- 清理复核: 本单各表归零、队列空、provisioner running(证据目录 /tmp/verify-oltsim-e2e-1788356522)。

## 根因与修复指引

- 根因: cbdddb22(feat(provisioner): switch log telnet and tl1 drivers)引入三态开关并把缺省
  从「OLT_ADDR 即 telnet」改为 log 桩,但 deployments/docker-compose.102.app.yml 的 provisioner
  服务未同步加 `BOSS_PROVISION_DRIVER: "telnet"`,部署 latest 后旧链路静默变 noop;Sep 02 12:34(UTC)
  一条存量 PENDING 任务 2 已被 noop 落 DONE(日志可查)。helm values 的「配置真实 OLT 后 Telnet
  执行器生效」注释同样过期。
- 修复已执行(2026-09-02): compose 补 `BOSS_PROVISION_DRIVER: "telnet"`(6dd2fa69),helm values 注释
  同步;102 上 /tmp/docker-compose.102.app.yml 已更新并按 compose 重建
  (docker compose -p boss-app -f /tmp/docker-compose.102.app.yml up -d provisioner),
  容器收敛回 compose 管理(project=boss-app;旧手工容器配置快照存 102:/tmp/boss-provisioner.old.json)。
  注意 compose 副本在 /tmp,机器重启丢文件,后续部署经 deploy-cluster.sh 会重新下发。

## 修复后绿证(2026-09-02 22:02 +08:00)

- 重跑本脚本 exit 0:precheck driver=telnet 无 WARN;resource=525 port=806 template=176,
  订单 ORD-20260902-000630(id 664),任务 252(isolated,loid=98)。
- task=252 DONE + provision_logs=SUCCESS;oltsim records 收到
  `{"template":"176","taskNo":"PRV-O664","event":"preConfigOLT","result":"OK"}`,
  journal 同秒 `apply ok template=176 task=PRV-O664 event=preConfigOLT`。
- 清理复核: 本单各表归零、队列空、provisioner running(证据目录 /tmp/verify-oltsim-e2e-1788357750)。
- 启动日志基准: `provisioner: telnet executor -> 172.26.0.1:2323`(14:01:53 UTC)。

## 存量遗留清理(2026-09-02)

- 已按精确 ID 清理更早 acc_tl1_ 残留: pon_onu_alloc(515,0,7,5)、port 785
  (P-acc_tl1_1788347248-01,order_id=648 指向已删订单,属孤儿预留)、resource 515、
  template 166;清理后 acc_ 前缀资源/模板归零,队列空。
- 教训: 删资源会被 pon_onu_alloc 外键拦截,须先清分配行(与本脚本 cleanup 顺序一致)。
