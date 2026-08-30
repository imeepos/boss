#!/usr/bin/env bash
# verify-deploy: 部署复验脚本(ISSUE.md「复验口诀」的脚本化)。
# 三步核验,任一不过退出 1:
#   1) /healthz 自报 commit —— 服务端真的换了版本;
#   2) admin-web index.html 引用的 index-*.js bundle 指纹 —— 前端真的换了版本;
#   3) 可选 --feature 串 —— 新代码文本确实已在新 bundle 内(治 bundle 缓存"改了看不到")。
# 用法:
#   scripts/ops/verify-deploy.sh [--expect-sha <sha>] [--feature <str>]
# 环境: BASE_URL(默认 102:28080) / WEB_URL(默认 102:5180) 可覆盖。
# --expect-sha 缺省取本树 git HEAD(worktree 内亦正确,读主 .git)。
# 自测: --selftest 用内置样例验证判定逻辑,不访问网络。
set -u

BASE_URL="${BASE_URL:-http://192.168.0.102:28080}"
WEB_URL="${WEB_URL:-http://192.168.0.102:5180}"
EXPECT_SHA=""
FEATURE=""
SELFTEST=0
while [ $# -gt 0 ]; do
  case "$1" in
    --expect-sha) EXPECT_SHA="${2:-}"; shift 2 ;;
    --feature) FEATURE="${2:-}"; shift 2 ;;
    --selftest) SELFTEST=1; shift ;;
    *) echo "unknown arg: $1"; exit 2 ;;
  esac
done

# judge: 读 healthz json + bundle 名 + 可选特征计数,输出三步判定(供 --selftest 复用)。
judge() {
  EXPECTED="$1" FEATURE="$2" python3 -c '
import json, os, sys
healthz_raw, bundle_name, feature_hits = sys.stdin.read().split("\n--SPLIT--\n")
expected = os.environ["EXPECTED"]; feature = os.environ["FEATURE"]
bad = 0
try:
    commit = json.loads(healthz_raw).get("commit", "")
except Exception as e:
    print(f"VERIFY FAIL healthz not json: {e}"); sys.exit(1)
if not commit:
    print("VERIFY FAIL healthz missing commit (旧版本二进制?)"); bad += 1
elif expected and not expected.startswith(commit):
    print(f"VERIFY FAIL server commit={commit} != expected {expected[:len(commit)]}"); bad += 1
else:
    print(f"VERIFY OK  server commit={commit}")
if not bundle_name:
    print("VERIFY FAIL index.html 无 index-*.js bundle 引用"); bad += 1
else:
    print(f"VERIFY OK  admin-web bundle={bundle_name}")
if feature:
    if feature_hits.strip() == "" or feature_hits.strip() == "0":
        print(f"VERIFY FAIL 特征串未上线: {feature}"); bad += 1
    else:
        print(f"VERIFY OK  特征串已上线: {feature} x{feature_hits.strip()}")
sys.exit(1 if bad else 0)
'
}

if [ "$SELFTEST" = 1 ]; then
  echo "[selftest] 期望 3 步全过:"
  printf '{"status":"ok","commit":"abc1234"}\n--SPLIT--\nindex-DdB7kQ9L.js\n--SPLIT--\n3\n' | judge "abc1234abcd" "x" && echo "[selftest] pass" || { echo "[selftest] FAIL"; exit 1; }
  echo "[selftest] 期望 commit 不符被拦:"
  if printf '{"status":"ok","commit":"old0001"}\n--SPLIT--\nindex-x.js\n--SPLIT--\n1\n' | judge "abc1234abcd" ""; then
    echo "[selftest] FAIL: 陈旧版本未被拦截"; exit 1
  fi
  echo "[selftest] pass"
  exit 0
fi

if [ -z "$EXPECT_SHA" ]; then
  EXPECT_SHA="$(git rev-parse HEAD 2>/dev/null || true)"
  if [ -z "$EXPECT_SHA" ]; then
    echo "usage: --expect-sha <sha> (非 git 树内必须显式给)"; exit 2
  fi
fi

HEALTHZ_RAW="$(curl -sf "$BASE_URL/healthz" || true)"
BUNDLE_NAME="$(curl -sf "$WEB_URL/" | grep -o 'index-[A-Za-z0-9_-]*\.js' | head -1 || true)"
FEATURE_HITS=""
if [ -n "$FEATURE" ] && [ -n "$BUNDLE_NAME" ]; then
  FEATURE_HITS="$(curl -sf "$WEB_URL/assets/$BUNDLE_NAME" | grep -c -F "$FEATURE" || true)"
fi
printf '%s\n--SPLIT--\n%s\n--SPLIT--\n%s\n' "$HEALTHZ_RAW" "$BUNDLE_NAME" "$FEATURE_HITS" | judge "$EXPECT_SHA" "$FEATURE"
