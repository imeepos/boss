#!/usr/bin/env bash
# 隧道 URL 上报 + 存活探测:把 cloudflared 快速隧道当前 URL 写入 stripe.webhookUrl 配置,
# 并检查容器存活;容器 DOWN 或 URL 无法解析时经 /api/admin/v1/ops/notify-emit 推告警到提醒中心。
# 隧道重启 URL 变化后由 cron 周期拉平,消除静默失效;容器进程 DOWN 也不再静默。
# 用法(102 上 cron 每 2~5 分钟): scripts/ops/stripe-tunnel-url.sh
# 环境:BASE_URL / ADMIN_API_KEY / STRIPE_TUNNEL_CONTAINER(缺省 cf-stripe-boss)可覆盖。
set -u

BASE_URL="${BASE_URL:-http://192.168.0.102:28080}"
API="${BASE_URL}/api/admin/v1"
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
KEY="${ADMIN_API_KEY:-$(python3 -c "import json;print(json.load(open('$ROOT/.agents/skills/bossctl-cli/test-accounts.json'))['admin']['apiKeys'][0]['key']") 2>/dev/null || true)}"
CONTAINER="${STRIPE_TUNNEL_CONTAINER:-cf-stripe-boss}"

if [ -z "${KEY:-}" ]; then
  echo "stripe-tunnel-url: ADMIN_API_KEY empty, skip"; exit 0
fi

# 告警通道:复用 remind 中心(ops/notify-emit),refType=stripe_tunnel。
# 上报脚本/容器身份用固定 refID,告警同源不重复入库(提醒中心按 refType+refID 幂等)。
emit_alert() {
  local title="$1" content="$2" level="${3:-WARN}"
  curl -sS -m 10 -X POST "$API/ops/notify-emit" \
    -H "X-API-Key: $KEY" -H "Content-Type: application/json" \
    -d "{\"refType\":\"stripe_tunnel\",\"refID\":\"$CONTAINER\",\"level\":\"$level\",\"title\":\"$title\",\"content\":\"$content\",\"link\":\"/base/stripeconfig\"}" \
    >/dev/null 2>&1 || echo "alert emit failed: $title"
}

# 1) 容器存活探测(进程 DOWN 直接告警,不再依赖日志)。
if ! docker inspect -f '{{.State.Running}}' "$CONTAINER" 2>/dev/null | grep -q '^true$'; then
  STATE=$(docker inspect -f '{{.State.Status}}' "$CONTAINER" 2>/dev/null || echo "missing")
  emit_alert "Stripe 隧道容器 DOWN" "容器 $CONTAINER 状态=$STATE,webhook 回调将静默失效" URGENT
  echo "tunnel container $CONTAINER state=$STATE"; exit 1
fi

# 2) 取隧道当前 URL(102 本机 docker 日志里的 trycloudflare 地址;cron 跑在 102)。
URL="$(docker logs "$CONTAINER" 2>&1 | grep -oE 'https://[a-z0-9-]+\.trycloudflare\.com' | tail -1 | tr -d '\r')"
if [ -z "$URL" ]; then
  emit_alert "Stripe 隧道 URL 解析失败" "容器 $CONTAINER 存活但日志无 trycloudflare 地址(隧道未就绪?)" WARN
  echo "tunnel url not found in $CONTAINER logs"; exit 1
fi
WANT="$URL/api/user/v1/webhooks/stripe"

# 3) 若已是最新则跳过。
CUR="$(curl -sS -m 15 "$API/stripe-config" -H "X-API-Key: $KEY" \
  | python3 -c "import json,sys
try: print(json.load(sys.stdin)['data']['fields'].get('stripe.webhookUrl',{}).get('value',''))
except Exception: print('')")"
if [ "$CUR" = "$WANT" ]; then
  echo "already up to date: $WANT"; exit 0
fi

# 4) 写入 webhook 组(webhookUrl 非 secret,直接 PUT)。
BODY="$(python3 -c "import json;print(json.dumps({'values':{'stripe.webhookUrl':'$WANT'}}))")"
RESP="$(curl -sS -m 15 -X PUT "$API/stripe-config/webhook" \
  -H "X-API-Key: $KEY" -H "Content-Type: application/json" -d "$BODY")"
if echo "$RESP" | grep -q '"code":0\|"ok":true'; then
  echo "stripe.webhookUrl -> $WANT"
else
  emit_alert "Stripe 隧道 URL 上报失败" "PUT stripe-config/webhook 返回异常: $RESP" WARN
  echo "update failed: $RESP"; exit 1
fi