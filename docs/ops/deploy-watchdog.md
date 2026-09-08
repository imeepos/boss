# 102 部署通道看门狗方案（C1，2026-09-08）

> 状态：**已获负责人书面批准（session_link 往返，2026-09-08）**。批复要点：
> ①组合方案（看门狗先行+runner 升级另立任务）通过；②workflow_dispatch 触发器
> （yml 增行）授权同意；③watchdog-token 文件路径（0600）授权同意，token 负责人侧
> 探查后补，缺失先跑纯告警态。附加要求（已落实）：六闸门参数以配置常量呈现
> （DEPLOY_WATCHDOG_* 环境可覆盖）；[deploy-watchdog] 告警行必含 task_id 与所采取动作。
> 缺陷登记：ISSUE.md「CI/deploy-102」——act_runner 0.2.11 网络抖动时拉到 deploy 任务
> 却无 job 容器、无错误行，任务卡死需人工推空提交 retrigger；近两周两次阻断交付
> （W4 手动等价部署一次、W7 阻断一次）。

## 0. 结论先行

推荐 **组合：看门狗先行（本方案，立即落地） + runner 升级另立任务（不捆绑本批）**。

依据（一句话版）：上游 release notes 无一条明确声明修复本缺陷的精确症状
（任务已 pickup 但 job 容器未创建、无错误行、任务挂死），升级 0.2.11→3.x 属
大跨度迁移（改镜像名、配置文件代次、breaking changes 三档），而看门狗直接命中
观测到的症状且零侵入；两者回滚方式独立，不捆绑才能各自 revert。

## 1. 实锤事实（2026-09-08 现场核实）

- 102 gitea-runner = `docker.m.daocloud.io/gitea/act_runner:0.2.11`（上游 2024-09-24
  发布，距今 23 个月）；Gitea 服务端 1.24.7；runner 容器 compose 管理
  （/home/imeepos/gitea/compose.yml），卷挂 docker-config.json + docker.sock。
- 上游仓库已更名 `gitea/act_runner` → `gitea/runner`（v1.0.0, 2026-05-05），
  最新 v3.4.1（2026-09-08）。版本谱系与相关修复（均引自上游 release notes）：
  - v0.2.12（2025-06）：#645「Report errors by setting raw_output when it's error
    level」——治「出错不留痕」，与"无错误行"症状同源但非同一症状。
  - v0.2.13（2025-08）：#741「Timeout to wait for and optionally require docker
    always」——docker daemon 不可达时不再无限等待，最接近"抖动后挂死"类缺陷。
  - v0.5.0（2026-04）：#852 Heartbeat ReportState for long-running silent jobs。
  - v2.0.0（2026-06）：完整 runner 侧取消处理、取消时杀进程组防 hang、防步骤日志丢失。
  - v3.4.0（2026-09-07）：job 结束时清理其创建的容器/网络/卷（**对我们是风险**，见 §2）。
- **没有任何一版 release note 写明修复「picked up but no job container」精确症状**，
  升级收益是间接证据（risk reduction），不是实证修复。
- 镜像可拉性实测（daocloud 镜像源）：0.2.12 / 0.2.13 / 1.0.8 / 2.0.1 / 3.4.1 全部
  `docker pull` 成功。
- 缺陷诱因实证：102 gitea-runner 日志 2026-09-08 当日就有两段 `failed to fetch task`
  报错（db recovery mode / connection reset by peer，源于当日 /srv/fast 磁满事故），
  网络与依赖服务抖动是常态，缺陷潜伏真实存在。
- gitea 数据可从 102 宿主直读：`docker exec gitea-postgres psql`（免 API token）；
  job 容器命名固定 `GITEA-ACTIONS-TASK-<task_id>`（0.2.11 二进制 strings 实证格式串
  `GITEA-ACTIONS-TASK-%d` ×2）。
- 状态码实证（action_run_job.status / action_task.status 同枚举）：
  0=unknown 1=success 2=failure 3=cancelled 4=skipped 5=waiting 6=running 7=blocked。
  误报样本已找到：repo4 长期 waiting(5)×527 无 runner 认领、repo83 历史 blocked(7)
  残留（run 已 failure）——看门狗必须带反误报闸门（§3.4）。
- retrigger 通道核实：102 宿主 `~/boss` 是导出树非 git checkout；102 的 gitea
  deploy key `build-host` 无 sker/boss 权限，**宿主无空提交推送通道**。
  Gitea 1.24.7 有 `POST /repos/{owner}/{repo}/actions/workflows/{workflow_id}/dispatches`
  （swagger 实证），deploy-102.yml 目前只有 `on: push` 触发器。

## 2. 备选方案评估（任务书二选一/组合）

### 方案① 升级 act_runner

- 目标版本二选一：
  - **v0.2.13**（保守档）：同仓库同镜像名，配置兼容，仅带 docker-wait timeout 等
    修复；跨度小、回滚秒级（换镜像 tag + compose up）。
  - **v3.4.1**（激进档）：跨 v1/v2/v3 三档 breaking changes：
    ①镜像名改 `gitea/runner`；②v2.0 起 `DOCKER_USERNAME/PASSWORD` 隐式拉取凭据
    移除——我们 deploy job 容器镜像（192.168.0.102:5000/boss/deploy-runner 私仓）
    依赖 runner 容器内 /root/.docker/config.json，v2+ 需确认该宿主 docker login
    路径仍生效；③v3.0 起 container.options 需 privileged、.runner 文件加进程锁；
    ④v3.4.0 起 job 结束会清理其创建的容器/网络/卷——deploy job 经挂入的 docker.sock
    在宿主拉起 boss-* 常驻容器，**若被误判为 job 资源将在 job 收尾被删，属生产事故级风险**；
    ⑤v3.x 与 Gitea 1.24.7 的兼容矩阵上游只测 latest，无背书。
- 升级收益：获得 #741/取消处理/心跳等一串防 hang 修复 + 后续维护线（0.2.x 已停更）。
- 结论：**值得做，但不与本批捆绑**。理由：无实证修复声明 + v3.4.0 资源清理风险 +
  与 Gitea 版本兼容无背书，任一条都要求灰度观察窗口；看门狗不依赖 runner 版本，
  先行落地可立即封住症状，升级留独立任务（建议先 0.2.13 灰度一周再议 3.x）。

### 方案② 非侵入看门狗（本批落地，下详）

## 3. 看门狗设计（deploy-watchdog.sh）

### 3.1 运行形态

102 宿主 crontab 每 5 分钟：

```
*/5 * * * * cd /home/imeepos/boss && ./scripts/ops/deploy-watchdog.sh >> /tmp/deploy-watchdog.log 2>&1 # deploy-watchdog
```

（口径同 db-patrol-gate/deploy-guard-alert：~/boss 导出树 + scp 同步。）

### 3.2 检测（只读）

1. gitea 侧：`docker exec -i gitea-postgres psql -U gitea -d gitea -At` 查询
   `action_run_job j JOIN action_run r ON r.id=j.run_id`，条件
   `j.repo_id=83 AND j.name='deploy' AND j.status IN (5,6) AND r.status IN (5,6)
   AND r.ref='refs/heads/main'`，取 job_id/task_id/status/created/started。
   （只看 sker/boss 的 deploy job，repo4 等噪音天然隔离。）
2. 宿主侧：`docker ps -a --format '{{.Names}}'` 收集 `GITEA-ACTIONS-TASK-*`。
3. 判卡死：age = now - coalesce(started, created) > 阈值（默认 900s，正常部署
   实测 86~172s，15 分钟无容器即高置信）**且** task_id>0 **且**
   `GITEA-ACTIONS-TASK-<task_id>` 不在容器名单——即「已领取但无 job 容器」精确症状。

### 3.3 动作（红线内）

- **retrigger**：经 Gitea API `POST .../workflows/deploy-102.yml/dispatches`
  （ref=main）触发新 run。deploy-102.yml 增补 `workflow_dispatch:` 触发器（一行）；
  concurrency `deploy-102` 的 cancel-in-progress 会自动取消卡死旧 run，无需触碰
  runner/容器。Classify 优先与「102 实际部署镜像 sha」比对（2026-09-07 已固化），
  dispatch 事件语义安全：有未部署 runtime 变更则真部署，无则明确 no-op 跳过，
  两态都不会误部署。
- **告警**：全程输出 `[deploy-watchdog] ...` 可 grep 日志；判定卡死但无 token、
  dispatch 失败、task_id=0（runner 未领取，可能是 runner 忙/掉线）一律
  `[deploy-watchdog] ALERT ...` + exit 1，只响铃不动手。
- **绝对禁止**：本脚本任何分支不执行 `docker restart gitea-runner`/`docker stop`/
  容器删除/镜像操作——只允许 dispatch（写入 gitea 侧）与告警日志。
  W7 事故红线（盲目 restart 杀进行中构建）由「无容器才动作」的判定条件 +
  动作白名单双重保证。

### 3.4 防误报/防风暴闸门

1. `task_id>0` 才可能 retrigger（=0 未领取，只告警）；
2. 同一 job_id 只 retrigger 一次：状态文件
   `/home/imeepos/boss-deploy-state/watchdog-state.env` 记 `handled_job_ids`；
3. 全局冷却：两次 retrigger 间隔 ≥30 分钟（`DEPLOY_WATCHDOG_COOLDOWN_SEC` 可调）；
4. 每日上限 4 次（超限转 ALERT）；
5. r.status 必须 ∈(5,6)——run 已终态的 blocked 残留（§1 实证存在）不触发；
6. 无卡死任务时输出 `[deploy-watchdog] OK` 零动作（正常运行零打扰，验收可 grep）。

### 3.5 凭据

- 检测侧免凭据（psql 宿主直读）。
- dispatch 需 Gitea token：一次性在 102 落
  `/home/imeepos/gitea/runner/watchdog-token`（0600，scope: repo 读写 sker/boss），
  由负责人在 Gitea 后台生成后放置（脚本读文件，值不进仓库、不进日志）。
  token 缺失时自动降级为纯告警模式（安全但只响铃），落地文档写明两态口径。

### 3.6 自测（--selftest，离线零副作用）

注入夹具驱动 judge 逻辑，六个用例：
①stuck 任务+无容器 → 判定卡死、输出 RETRIGGER 意图（dry-run 不发请求）；
②同 job 已 handled → 抑制（幂等）；③运行中+容器在 → OK；
④run 终态的 blocked 残留 → OK（反误报）；⑤repo4 waiting 噪音 → OK（范围隔离）；
⑥无 token → ALERT+exit 1（降级态）。另 `--dry-run` 对真实环境只读巡检
（psql+docker ps 全走，动作只打印不执行）。

## 4. 阶段 1 落地清单（获批后执行）

1. worktree `feat/ci-deploy-watchdog`（已建）内提交：
   - `scripts/ops/deploy-watchdog.sh`（含 --selftest/--dry-run，≤300 行）；
   - `.gitea/workflows/deploy-102.yml` 增补 `workflow_dispatch:` 触发器；
   - `docs/ops/patrol-cron.md` 增补看门狗安装/两态口径/卸载方式；
   - 本文档随批注更新批准结论。
2. 合并 main → push 触发自动部署（本批改动含 .gitea/workflows，Classify 判 runtime
   会真实走部署链）：healthz 200 + verify-deploy.sh 指纹一致 + deploy-marker 写入，
   整链无人工干预（验收规则 2）。
3. scp 脚本至 102 ~/boss/scripts/ops/，装 cron；负责人放置 token 前先以纯告警态运行。
4. 验证：`--selftest` 6 用例全过；`--dry-run` 对真实环境输出 OK 零动作（无卡死时）；
   手工调 API dispatch 一次证明 retrigger 通道通（产生一次 run，docs-only 路径 no-op）。
5. 收尾四步照旧（push gitea → 主树 ff-only merge → worktree remove → branch -d），
   回报分支/提交/验证证据至 session-ba11cc82。

## 5. 回滚

- 看门狗：删 crontab 行 + 删 ~/boss/scripts/ops/deploy-watchdog.sh 即全下线；
  仓库 revert 单提交。对 runner/部署链零耦合。
- workflow_dispatch 触发器：一行 revert，无行为影响（push 触发不变）。

## 6. 遗留与边界

- runner 升级（0.2.13 灰度 → 3.x 评估）另立任务，不随本批；
- 本工作流无迁移、无前端改动；
- repo4 等其他 workflow 不在看门狗范围（如需可后续加参数）。
