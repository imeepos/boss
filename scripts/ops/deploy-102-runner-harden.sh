#!/usr/bin/env bash
# deploy-102 runner/镜像宿主加固(2026-09-06 P2-B,社区实践校准版)——Lead 在 102 执行。
# 背景:每周日 04:00 docker-clean.sh 的 docker image prune -af 会把未被容器引用的
# deploy-runner 与基座 bookworm 全清掉(ISSUE.md CI/deploy-102 事故根因),而
# act_runner 进程侧拉私有仓库镜像无凭据(401 秒取消且默认日志无错误行)。本脚本落地
# 两道宿主侧防线(与仓库内 workflow 守护 job 互为纵深,不影响其独立自愈能力):
#   1) docker-clean.sh 的 prune 加 --filter "label!=ci-keep":打标镜像
#      (deploy-runner,Dockerfile LABEL ci-keep=true)获得周清豁免;
#   2) imeepos crontab 每日保活:用宿主已就位的注册表凭据 pull deploy-runner 与
#      基座 bookworm,本地 tag 常在 → runner/job 拉取零 registry 交互,401 无从触发。
# 用法(在 102 上,~/boss 为导出树,先 scp 本脚本):
#   scp scripts/ops/deploy-102-runner-harden.sh imeepos@192.168.0.102:~/boss/scripts/ops/
#   cd ~/boss && bash scripts/ops/deploy-102-runner-harden.sh --apply    # 幂等加固
#   bash scripts/ops/deploy-102-runner-harden.sh --check                # 只读检查,未加固 exit 1
#   bash scripts/ops/deploy-102-runner-harden.sh --rollback             # 还原备份并摘除保活 cron
# 更进一步的容器/配置层可选项(act_runner label 改本地 tag、runner 进程 auths 兜底、
# log level: debug)不在本脚本默认范围,见 deploy-102-runner-harden.README.md。
set -u

CLEAN_SH=/home/imeepos/docker-clean.sh
KEEP_TAG=ci-keep
CRON_MARK=deploy-runner-keepalive
APPLY=0; ROLLBACK=0; CHECK=0
for a in "$@"; do
  case "$a" in
    --apply) APPLY=1 ;;
    --rollback) ROLLBACK=1 ;;
    --check) CHECK=1 ;;
    *) echo "[harden] FAILED unknown arg: $a" >&2; exit 2 ;;
  esac
done
if [ $(( APPLY + ROLLBACK + CHECK )) -ne 1 ]; then
  echo "usage: $0 --apply|--check|--rollback" >&2
  exit 2
fi

cron_ok() {
  crontab -l 2>/dev/null | grep -q 'deploy-runner-keepalive'
}

filter_ok() {
  grep -q "label!=" "$CLEAN_SH" 2>/dev/null
}

if [ "$CHECK" = 1 ]; then
  ST=0
  if filter_ok; then echo "[harden] check: prune filter present"; else echo "[harden] check: prune filter MISSING"; ST=1; fi
  if cron_ok; then echo "[harden] check: keepalive cron present"; else echo "[harden] check: keepalive cron MISSING"; ST=1; fi
  exit $ST
fi

if [ "$APPLY" = 1 ]; then
  if [ ! -f "$CLEAN_SH" ]; then
    echo "[harden] FAILED $CLEAN_SH not found" >&2
    exit 1
  fi
  if ! filter_ok; then
    cp "$CLEAN_SH" "$CLEAN_SH.bak-$(date +%Y%m%d%H%M%S)" || { echo "[harden] FAILED backup" >&2; exit 1; }
    sed -i 's|^docker image prune -af$|docker image prune -af --filter "label!=ci-keep"|' "$CLEAN_SH" || { echo "[harden] FAILED sed" >&2; exit 1; }
    if ! filter_ok; then echo "[harden] FAILED prune filter patch did not take" >&2; exit 1; fi
    echo "[harden] prune filter patched (backup kept alongside)"
  else
    echo "[harden] prune filter already present, skip"
  fi
  if cron_ok; then
    echo "[harden] keepalive cron already present, skip"
  else
    { crontab -l 2>/dev/null; echo "30 4 * * * docker pull 192.168.0.102:5000/boss/deploy-runner:latest >> /tmp/deploy-runner-keepalive.log 2>&1 && docker pull 192.168.0.102:5000/runner:bookworm >> /tmp/deploy-runner-keepalive.log 2>&1 # deploy-runner-keepalive"; } | crontab -
    if ! cron_ok; then echo "[harden] FAILED keepalive cron install" >&2; exit 1; fi
    echo "[harden] keepalive cron installed (daily 04:30)"
  fi
  echo "[harden] apply done; note current image lacks ci-keep label until next rebuild"
  echo "[harden] optional now: DEPLOY_RUNNER_IMAGE=192.168.0.102:5000/boss/deploy-runner:latest scripts/ops/deploy-runner-guard.sh --force-rebuild"
  exit 0
fi

if [ "$ROLLBACK" = 1 ]; then
  B=$(ls -1t "$CLEAN_SH".bak-* 2>/dev/null | head -1)
  if [ -n "$B" ]; then cp "$B" "$CLEAN_SH"; echo "[harden] restored $CLEAN_SH from $B"; else echo "[harden] no backup found, prune line untouched"; fi
  crontab -l 2>/dev/null | grep -v 'deploy-runner-keepalive' | crontab -
  echo "[harden] keepalive cron removed"
  exit 0
fi
