#!/usr/bin/env bash
# deploy-runner 镜像守护(2026-09-06 deploy-102 事故固化,见 ISSUE.md CI/deploy-102)。
# 背景:job 容器镜像被 docker prune 清掉后,runner 拉 job 容器即失败且 runner 日志无
# 错误行(act_runner 0.2.11 缺陷),部署通道整体静默停摆;当时手工重建又缺 compose 二进制
# 与注册表凭据(原只烤在镜像里,重建即失)。本脚本由 deploy workflow 首步
# runner-image-guard job 调用,把「一次性手修」固化为流水线自愈:
#   快路径(只读,幂等零副作用):经 docker.sock 引擎 API 探本机镜像 + 注册表 v2 API 探
#     副本,双双存在 → 打 OK no-op 退出 0(自检可无限重复,A2);
#   补推路径:本机在而注册表副本缺 → 只补推,不重建;
#   重建路径:本机缺 → 用 scripts/deploy-runner.Dockerfile 就地重建(构建上下文 =
#     scripts/ops/,compose 二进制与凭据 JSON 均固化在仓库内,不再依赖 /tmp 与宿主文件)
#     并推回注册表;基座镜像缺失时用凭据自动补拉。
# 守护 job 刻意跑在 windows-latest 标签(= daocloud 公网源 node:20-bookworm,本地被
# 周清后可匿名重拉,不撞私有仓库 401 墙;见 workflow 注释),与 deploy-runner、
# runner:bookworm 双双解耦,双缺场景守护仍能拉起;镜像无 docker CLI,慢路径现装
# docker.io——与重建 Dockerfile 自身的 RUN apt-get 同一依赖面,不引入新外部依赖。
# 失败一律输出 "[deploy-guard] FAILED ..." 并退出非零(禁止静默,可 grep 留痕)。
# 用法: scripts/ops/deploy-runner-guard.sh [--force-rebuild] [--no-push]
#   --force-rebuild 跳过自检强制重建(A1 演练/测试用);--no-push 只构建不推注册表(测试用)。
# 环境: DOCKER_SOCK / DEPLOY_RUNNER_IMAGE / DEPLOY_RUNNER_BASE 可覆盖(测试用)。
set -euo pipefail

IMG=192.168.0.102:5000/boss/deploy-runner:latest
BASE=192.168.0.102:5000/runner:bookworm
REG=192.168.0.102:5000
SOCK=/var/run/docker.sock
# printenv 探测:set -u 下直接引用未设 env 会 unbound,先探测再引用
if printenv DEPLOY_RUNNER_IMAGE >/dev/null; then IMG="$DEPLOY_RUNNER_IMAGE"; fi
if printenv DEPLOY_RUNNER_BASE >/dev/null; then BASE="$DEPLOY_RUNNER_BASE"; fi
if printenv DOCKER_SOCK >/dev/null; then SOCK="$DOCKER_SOCK"; fi
# 注册表仓库路径(去 registry 前缀与 :tag),供 v2 manifest 探活
REGISTRY_REPO=$(printf '%s' "$IMG" | sed "s|^$REG/||; s|:[^:]*$||")
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
CREDS="$ROOT/scripts/ops/deploy-registry-config.json"

FORCE=0
NOPUSH=0
for a in "$@"; do
  case "$a" in
    --force-rebuild) FORCE=1 ;;
    --no-push) NOPUSH=1 ;;
    *) echo "[deploy-guard] FAILED unknown arg: $a" >&2; exit 2 ;;
  esac
done

trap 'echo "[deploy-guard] FAILED rc=$? img=$IMG (guard step aborted)" >&2' ERR

if [ ! -f "$CREDS" ]; then
  echo "[deploy-guard] FAILED creds file missing: $CREDS" >&2
  exit 1
fi
if [ ! -S "$SOCK" ]; then
  echo "[deploy-guard] FAILED docker socket not mounted at $SOCK" >&2
  exit 1
fi

# 快路径探活:200=存在 404=缺失 其他=引擎/网络异常(大声失败)
LOCAL=0
local_code=$(curl -s --unix-socket "$SOCK" -o /dev/null -w "%{http_code}" "http://d/images/$IMG/json")
case "$local_code" in
  200) LOCAL=1 ;;
  404) LOCAL=0 ;;
  *) echo "[deploy-guard] FAILED docker engine unreachable via $SOCK (http=$local_code)" >&2; exit 1 ;;
esac

# 注册表副本探活:凭据来自仓库固化文件(经 DOCKER_CONFIG 注入给 docker CLI,不碰宿主 ~/.docker)
auth=$(python3 -c "import json,sys;print(json.load(open(sys.argv[1]))['auths']['$REG']['auth'])" "$CREDS")
reg_code=$(curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Basic $auth" "http://$REG/v2/$REGISTRY_REPO/manifests/latest")
if [ "$reg_code" != "200" ] && [ "$reg_code" != "404" ]; then
  echo "[deploy-guard] FAILED registry unreachable at $REG (http=$reg_code)" >&2
  exit 1
fi
REGOK=0
if [ "$reg_code" = "200" ]; then REGOK=1; fi

if [ "$FORCE" = 0 ] && [ "$LOCAL" = 1 ] && [ "$REGOK" = 1 ]; then
  echo "[deploy-guard] OK image present locally and in registry, no-op: $IMG"
  exit 0
fi

# 慢路径前置:基座无 docker CLI 则现装(仅重建/补推时付出此成本,快路径零开销)。
# 注意:apt 不加 -qq 静默标志并配 timeout 兜底——2026-09-06 在 102 实测 -qq 模式下
# apt 安装会概率性挂在 _apt http 方法(3/3 复现,全量输出 2/2 且 <45s 完成),
# timeout 保证最坏情况是大声红而非 job 挂死到 3h 超时。
if ! command -v docker >/dev/null 2>&1; then
  echo "[deploy-guard] docker CLI missing, installing docker.io via apt (same dep as rebuild)..."
  timeout 180 apt-get update
  DEBIAN_FRONTEND=noninteractive timeout 300 apt-get install -y --no-install-recommends docker.io
fi
if ! command -v docker >/dev/null 2>&1; then
  echo "[deploy-guard] FAILED docker CLI still missing after apt install" >&2
  exit 1
fi
mkdir -p "$ROOT/.guard-docker-config"
cp "$CREDS" "$ROOT/.guard-docker-config/config.json"
export DOCKER_CONFIG="$ROOT/.guard-docker-config"

if [ "$FORCE" = 0 ] && [ "$LOCAL" = 1 ]; then
  # 本机在而注册表缺:补推即可,无需重建(--force-rebuild 时不走此捷径,走重建)
  if [ "$NOPUSH" = 0 ]; then docker push "$IMG"; fi
  echo "[deploy-guard] OK registry copy restored (local image was intact): $IMG"
  exit 0
fi

# 重建路径:确保基座在(拉取走上面已就位的 DOCKER_CONFIG 凭据),就地重建并推回注册表
if docker image inspect "$BASE" >/dev/null 2>&1; then
  echo "[deploy-guard] base image present: $BASE"
else
  echo "[deploy-guard] base image missing, pulling: $BASE"
  docker pull "$BASE"
fi
docker build -f "$ROOT/scripts/deploy-runner.Dockerfile" -t "$IMG" "$ROOT/scripts/ops"
PUSH_NOTE=skipped
if [ "$NOPUSH" = 0 ]; then
  docker push "$IMG"
  PUSH_NOTE=pushed
fi
echo "[deploy-guard] OK rebuilt from scripts/deploy-runner.Dockerfile (registry copy: $PUSH_NOTE): $IMG"
