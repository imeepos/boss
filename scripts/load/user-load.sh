#!/usr/bin/env bash
# user 端压测驱动 v2: 登录时延单次真码采样 + 60VU 并发用 API key(213 客户)打
# 下单(289/291 地址, perf- 标记)/支付(stripe-intent)/订单列表/实名读。
# 真实约束: portal_accounts 仅 213 可登录; 验证码 60s 冷却 → 登录并发不可行, 不计入 60VU。
#           phoneCap=5/24h、addrCap=3 → 下单并发被风控约束, 42300 拦截率入报告。
# 用法: scripts/load/user-load.sh [TARGET]
set -uo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"
TARGET="${1:-60}"

command -v k6 >/dev/null 2>&1 || export PATH="/opt/homebrew/bin:$PATH"
command -v k6 >/dev/null 2>&1 || { echo "k6 not found"; exit 1; }
ssh -o BatchMode=yes -o ConnectTimeout=3 imeepos@192.168.0.102 "echo ok" >/dev/null 2>&1 || { echo "ssh 免密不通 102"; exit 1; }

# 1) 登录时延采样(单次真码全链路, 避免 60s 冷却循环失败)
echo "==> 1/3 真码登录时延采样(客户 213)"
t0=$(date +%s%N)
LOGIN_MS=""
TOKEN=$(scripts/e2e-login.sh 13900001234 login 2>/dev/null | tail -1)
if [ -n "$TOKEN" ]; then
  t1=$(date +%s%N)
  LOGIN_MS=$(( (t1 - t0) / 1000000 ))
  echo "  login_ms=$LOGIN_MS (含发码+取码+登录全链路)"
else
  echo "  登录采样失败(冷却?), 用 API key 兜底"
fi

# 2) 60VU 并发主体: API key(213) 直连
echo "==> 2/3 k6 60VU 并发(API key 213)"
export BASE="http://192.168.0.102:28080"
export TARGET="$TARGET"
export APIKEY="boss_a7c676b7d33083cffc00820664e1d539"
k6 run --summary-trend-stats="avg,p(95),p(99),max" scripts/load/user-load.js 2>&1 | tail -45
echo "LOGIN_MS=$LOGIN_MS  (真码登录全链路时延, 60s 冷却下每次仅能 1 次)"
echo "==> 3/3 提示: 压测后执行 scripts/load/user-cleanup.sh --apply 清理 perf- 造数"