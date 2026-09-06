# deploy-102 runner/镜像宿主加固操作说明(2026-09-06,P2-B)

社区实践校准结论(调研会话回传)+ 本轮落地的仓库内自愈能力(deploy workflow 首步
runner-image-guard + 标记落盘巡检,见 ISSUE.md CI/deploy-102 已固化条目)合并考虑后,
还差两道**宿主侧**防线,须 Lead 在 102 执行。全部幂等、带备份、可一键回滚。

## 背景(事故机理,已实证)

- job 镜像本地存在时 runner 零 registry 交互(102 runner 配置 force_pull: false);
- 镜像被周清(prune -af)后,runner 侧拉取不带凭据,撞 htpasswd 401 秒取消,
  且 act_runner 0.2.11 默认日志级别无错误行——即 2026-09-06 六次 push 零部署事故;
- 周清对象是「所有未被容器引用的镜像」:deploy-runner 与基座 bookworm 每周日都危险。

## Lead 操作清单(在 102 上执行)

```bash
# 0) 取脚本(~/boss 为导出树,非 git checkout)
scp scripts/ops/deploy-102-runner-harden.sh imeepos@192.168.0.102:~/boss/scripts/ops/

# 1) 执行加固(幂等;自动备份 docker-clean.sh 并打补丁 + 装每日保活 cron)
ssh imeepos@192.168.0.102 'cd ~/boss && bash scripts/ops/deploy-102-runner-harden.sh --apply'

# 2) 验证
ssh imeepos@192.168.0.102 'cd ~/boss && bash scripts/ops/deploy-102-runner-harden.sh --check'
ssh imeepos@192.168.0.102 'crontab -l | grep keepalive; grep label!= ~/docker-clean.sh'
```

## 加固内容

1. `docker-clean.sh` 的 `docker image prune -af` 追加 `--filter "label!=ci-keep"`:
   打了 `ci-keep=true` 标签的镜像(deploy-runner,Dockerfile 已加 LABEL)不再被周清。
2. imeepos crontab 加每日 04:30 保活(周日清完次日即回填):
   `docker pull` deploy-runner + 基座 bookworm(宿主 imeepos 凭据已就位),
   本地 tag 常在 → runner/job 拉取零 registry 交互,401 鉴权墙无从触发。

注意:当前线上 deploy-runner 镜像建于打标之前,**首次重建后才带 ci-keep 标签**。
不等周清可立即重建一次(可选):

```bash
ssh imeepos@192.168.0.102 'cd ~/boss && DEPLOY_RUNNER_IMAGE=192.168.0.102:5000/boss/deploy-runner:latest bash scripts/ops/deploy-runner-guard.sh --force-rebuild'
```

(该命令为只重建不推注册表;若同时想刷新注册表副本,去掉 --no-push 由 workflow 守护自动做亦可。)

## 回滚

```bash
ssh imeepos@192.168.0.102 'cd ~/boss && bash scripts/ops/deploy-102-runner-harden.sh --rollback'
```

## 可选进阶(未默认执行,Lead 自行取舍)

- **act_runner label 改本地 tag**(社区最稳组合):config.yaml labels 改
  `deploy-102:docker://deploy-runner:local` + 宿主 `docker tag` 落地 + 重启 runner 容器。
  收益:runner 永不直连 registry,401 墙彻底消失;代价:改 runner 配置且需重启,故未脚本化。
- **runner 进程 auths 兜底**:把注册表 config.json 放进 gitea-runner 容器用户目录,
  runner 侧拉取即可带凭据;容器 FS 易逝,需挂载方案配合。
- **act_runner log level: debug**:0.2.11 秒取消无错误行,调 debug 便于下次秒级排障
  (config.yaml log.level,改后重启容器)。
- **部署镜像日期 tag + latest 双发**:回滚抓手,属 deploy workflow 既有 Push images 步骤
  的行为变更,按本轮「不改既有步骤语义」约束未做,如需可另开任务。
