# TL1 驱动切换 runbook(生产从仿真链切真实 U2000 北向)

> 目标读者:执行 provisioner 驱动从 telnet 切换到 tl1 的运维/交付人员。
> 设计事实源:`docs/design/tl1-integration-design.md`(§7 配置开关/§5 编排/§6 错误分类);
> 旧链路回归门:`docs/ops/oltsim-telnet-regression.md`;TL1 e2e:`scripts/verify-tl1-e2e.sh`。
> 本文只改运行时开关与数据准备,不改代码/配置文件(切换动作本身除外)。

## 1. 适用范围与现状风险

- **适用范围**:已拿到局方 U2000 北向访问权(host:13027、TL1 账号),要把 provisioner 执行器
  从 telnet 切到 tl1 的正式开通环境;也适用于 102 预发上用 tl1sim 演练本流程。
- **现状**:provisioner 当前以 `BOSS_PROVISION_DRIVER=telnet` 运行,指令发给 cmd/oltsim 仿真器
  (启动日志 `provisioner: telnet executor -> <addr>`,且每次启动必打
  `[provisioner] ALERT driver=telnet simulation/oltsim 仿真链路:非真实 TL1 下发,生产环境请切换 BOSS_PROVISION_DRIVER=tl1`)。
- **风险一(语义)**:telnet/oltsim 链上的 provision_logs.SUCCESS **只代表仿真器认可了
  `provision apply template=...` 这条伪指令**,不等于设备真实开通。禁止把仿真链 SUCCESS
  当生产开通成功向客户/局方交差。
- **风险二(静默失效)**:三态开关缺省是 log 桩——log 桩执行器**不失败**,直接把任务落 DONE
  (2026-09-02 事故:容器缺 `BOSS_PROVISION_DRIVER`,部署 latest 后旧链路静默 noop,一条存量
  PENDING 任务被 noop 落 DONE,详见 oltsim-telnet-regression.md)。切换期任何重启都必须
  回读启动日志确认生效驱动,不能只看容器 Running。
- **风险三(数据缺行)**:tl1 执行器按任务逐单解析端点与参数,缺数据会显式 FAILED
  (如 `no NMS endpoint for entity %d`/`OLT missing nms_oltid`/`port missing PON positioning`/
  `lo account missing`),不会静默;但法人/OLT 数据没铺齐就切换,会出现批量 FAILED。

## 2. 前置条件(顺序执行,任一不过即停)

1. **迁移先执行**:`000180_provision_log_driver`(provision_logs.driver 列)必须先于切换跑完,
   否则切换后留痕无驱动来源、审计无法区分通道;其前置 `000178`(provision_nms/nms_oltid/
   PON 定位列/pon_onu_alloc)与 `000179`(commands/device_response)同链已应用。核对:

   ```bash
   # 102 示例;生产同库执行
   docker exec -i boss-infra-postgres-1 psql -U boss -d boss -Atc \
     "SELECT version FROM schema_migrations WHERE version IN ('000178','000179','000180');"
   # 期望三行齐全;再确认列存在:
   docker exec -i boss-infra-postgres-1 psql -U boss -d boss -Atc \
     "SELECT column_name FROM information_schema.columns WHERE table_name='provision_logs' AND column_name='driver';"
   ```

2. **停止并排空任务队列**:切驱动须在 provisioner 停止、队列为零时进行(任务执行中换执行器
   会出现半程状态)。核对口径与 verify-tl1-e2e.sh 一致:

   ```bash
   docker exec -i boss-infra-postgres-1 psql -U boss -d boss -Atc \
     "SELECT count(*) FROM provision_tasks WHERE status IN ('PENDING','DOING');"
   # 非 0:等现有任务跑完,或按订单域流程处置(改约/关单)后再核;禁止带队列切换。
   docker stop boss-provisioner   # 排空后停容器(102 compose 副本在 /tmp,deploy-cluster.sh 会重下发)
   ```

## 3. Pre-flight(切换前检查清单)

### 3.1 核对 provision_nms 每法人实体端点与 TL1 凭证引用

- 取数优先级:**provision_nms(legal_entity_id) > 环境变量**(设计 §7)。生产必须以表行为准,
  环境变量只是 dev/102 连 tl1sim 的兜底。
- 逐法人核对:一法人一行(UNIQUE),host/port 指向局方 U2000 北向(默认 13027),
  username 非空,pass_cipher 可解密(`provision_nms.pass` 走 config_secrets 加解密 helpers):

  ```bash
  docker exec -i boss-infra-postgres-1 psql -U boss -d boss -Atc \
    "SELECT legal_entity_id, host, port, protocol, username, length(pass_cipher) FROM provision_nms ORDER BY legal_entity_id;"
  # 期望:每个在营法人恰一行;缺行=该法人任务将 FAILED("no NMS endpoint for entity %d")
   ```

- 同步核对 OLT 侧标识与端口定位(tl1 取数链依赖,缺即 FAILED):

  ```bash
   docker exec -i boss-infra-postgres-1 psql -U boss -d boss -Atc \
     "SELECT count(*) FROM resources WHERE type='OLT' AND (nms_oltid IS NULL OR nms_oltid='');"
   # 期望 0(缺 nms_oltid 的 OLT 无法下发);端口抽查 pon_frame/slot/port 非空、onu_no 分配器就绪。
   ```

- **凭证红线**:TL1 账密只允许存在于 provision_nms.pass_cipher(加密)或部署环境变量注入,
  禁止写入代码库/文档/runbook;本文只登记"引用"(哪个法人用哪行端点),不登记值。

### 3.2 验证 BOSS_PROVISION_TL1_ADDR/USER/PASS 兜底

- 仅当某法人 provision_nms 无行时才落到环境变量;生产若依赖兜底=数据未铺齐,应回 3.1 补行。
- 检查 provisioner 容器环境(确认无冲突残留):

  ```bash
  docker inspect boss-provisioner --format '{{range .Config.Env}}{{println .}}{{end}}' | grep BOSS_PROVISION
  # 生产切法:BOSS_PROVISION_DRIVER=tl1;TL1_* 三项留空(以表为准)或与局方端点一致。
  # 102 演练切法:BOSS_PROVISION_TL1_ADDR=172.26.0.1:13027, USER/PASS=tl1sim 演练账号(verify-tl1-e2e.sh 同款 admin/admin)。
  ```

### 3.3 用 cmd/tl1sim 或既有 TL1 测试做预检(不动生产)

- 单测/进程内回归(先于一切现场动作):

  ```bash
  go test ./internal/domain/provision/tl1/ ./cmd/tl1sim/ ./internal/app/ -run 'TL1|TestCodec|TestSession|TestExecutor'
  ```

- 102 全链路演练(隔离容器,不碰生产 provisioner):`scripts/verify-tl1-e2e.sh`,
  断言含 provision_logs.SUCCESS、tl1sim 收到 ADD-ONU/ADD-PONVLAN、Power-Off 负向订单不推进。
- 可选手工探活(局方允许时,仅 LOGIN/LOGOUT,不做写操作):

  ```bash
  /path/to/tl1sim -addr 127.0.0.1:13027 -record /tmp/tl1sim.jsonl &   # 或直接对局方端口
  # 用 TL1 客户端发 LOGIN → COMPLD EN=0 即通;凭证错会 DENY(EN=76546031 一类)。
  ```

## 4. 切换步骤

1. 确认 §2/§3 全绿(迁移齐全、队列排空、provisioner 已停)。
2. 设置驱动开关并重建 provisioner(compose 环境改 `deployments/docker-compose.102.app.yml`
   同款 env,生产走对应部署面):

   ```bash
   # 102:
   docker compose -p boss-app -f /tmp/docker-compose.102.app.yml up -d provisioner
   # env 关键项:BOSS_PROVISION_DRIVER=tl1(其余 TL1_* 按 3.2 结论)
   ```

3. 启动即查生效驱动(启动日志是唯一权威,容器 Running 不算):

   ```bash
   docker logs --since 2m boss-provisioner 2>&1 | grep -E 'provisioner: (started|.*executor)'
   # 期望:provisioner: tl1 executor: endpoint resolved per task
   # 出现 'telnet executor' / 无 executor 行(log 桩) = 切换失败,按 §6 处置。
   ```

## 5. 切换后验证(以留痕为准,不信自述)

用一笔可控测试单(生产=装维实单;102=verify-tl1-e2e.sh 夹具单)走 preConfigOLT 后核对:

1. **驱动来源**:

   ```bash
   docker exec -i boss-infra-postgres-1 psql -U boss -d boss -Atc \
     "SELECT task_id, result, driver FROM provision_logs ORDER BY id DESC LIMIT 5;"
   # 期望 driver='tl1'(后台"下发日志详情"页同名展示);''/log/telnet 都不算切换成功。
   ```

2. **指令形态为 TL1 而非 provision apply**:

   ```bash
   docker exec -i boss-infra-postgres-1 psql -U boss -d boss -Atc \
     "SELECT commands FROM provision_logs WHERE driver='tl1' ORDER BY id DESC LIMIT 1;"
   # 期望为 TL1 报文形态:ADD-ONU/ADD-PONVLAN/LST-ONU/LST-PONVLAN/LST-ONUSTATE,
   # 形如 ADD-ONU::OLTID=...,PONID=NA-0-7-5:TRC1::AUTHTYPE=LOID,...;(LOGIN 在会话层完成,不进该列)
   # 出现 'provision apply template=...' = 仍是 telnet 链,切换未生效。
   ```

3. **容器日志**:无 `[provisioner] ALERT driver=telnet` 告警;幂等重放可见
   `[provision-tl1] IDEMPOTENT SKIP(ONU-EXISTS|PONVLAN-EXISTS) task=...`。
4. **LOGIN 交互证据**:provision_logs.commands 只记业务指令(LOGIN 属会话层不落该列);
   演练环境以 tl1sim `-record` JSONL 首条 LOGIN 为证,生产以局方 U2000 侧登录审计为证。
5. 失败路径抽查(可选,tl1sim 演练已覆盖):任务 FAILED 时容器日志必含可 grep 的
   `[provision-tl1] EXEC FAILED task=<no> ctag=<t> cmd=<verb> EN=<code> ENDESC=<desc>`。
6. 证据留档:以上查询输出 + 启动日志存证据目录(仿 oltsim 回归的 /tmp/verify-* 目录惯例),
   注明时间戳与执行人。

## 6. 失败处置

| 现象 | 判定 | 动作 |
|:-----|:-----|:-----|
| 启动日志无 executor 行/任务秒 DONE 且 commands=noop | 落到 log 桩(开关未生效) | 立即停容器,回查 env 注入面,修正后重走 §4;被 noop 落 DONE 的任务须按订单域复核并重建任务 |
| `[provision-tl1] LOGIN FAILED addr=...` | TL1 账密错(表行或兜底) | 核对 provision_nms.pass_cipher 解密结果/局方账号;Manager 熔断 5min,修好后自动恢复,任务可重试 |
| `EXEC FAILED ... EN=<code>` DENY | 局方拒绝(License/参数/容量) | 以 ENDESC 与局方协查;订单停留可重试态,不人工改任务状态 |
| 批量 FAILED `no NMS endpoint for entity`/`OLT missing nms_oltid`/`port missing PON positioning`/`lo account missing` | 取数链数据缺行 | 回 §3.1 补数据后重试;属数据问题,不是代码问题 |
| DELAY 后超时/连接断(ErrConnBroken) | 网络或 U2000 抖动 | Manager 自动重连、executor 先查后写整段重跑(幂等);持续断连查网络/防火墙 13027 |

处置原则:TL1 链上失败**显式可见且可重试**(任务 FAILED+日志留痕),保持驱动不动、修根因;
禁止用任何方式把失败任务手工改成 SUCCESS。

## 7. 回滚与边界

- **回滚到 telnet 仅限仿真回归**:目的只有一个——重跑
  `scripts/verify-oltsim-provision-e2e.sh` 守住旧链路回归门(改回
  `BOSS_PROVISION_DRIVER=telnet` 并重启,同 §4 口径回读启动日志
  `provisioner: telnet executor -> <addr>` + ALERT 行)。跑完按切换流程切回 tl1。
- **禁止把仿真 SUCCESS 当生产成功**:telnet/oltsim 的 provision_logs.SUCCESS 语义=仿真器认可;
  生产开通成功唯一以 driver='tl1' 且 TL1 指令 COMPLD(EN=0)的留痕为准(§5 两条同时满足)。
- **生产禁止回滚到 log 桩**:log 桩会把任务 noop 落 DONE(伪成功,2026-09-02 事故机理);
  生产侧 TL1 故障期间的正确姿势是保持 driver=tl1 让任务显式 FAILED 堆积(有告警与日志),
  修根因后重试,或按订单域暂停新单进入自动化。
- 回滚动作同样要求队列排空(§2.2 口径)后执行,避免执行器中途更换。

## 8. 变更记录

| 日期 | 变更 | 备注 |
|:-----|:-----|:-----|
| 2026-09-03 | 首版(C5) | 依据 tl1-integration-design.md §5-§9、oltsim-telnet-regression.md 2026-09-02 事故、verify-tl1-e2e.sh 实跑口径 |
