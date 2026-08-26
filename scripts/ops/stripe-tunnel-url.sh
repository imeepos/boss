#!/usr/bin/env bash
# 隧道 URL 上报:把 cloudflared 快速隧道当前 URL 写入 stripe.webhookUrl 配置,
# 供 Stripe webhook 自愈循环比对。隧道重启 URL 变化后由 cron 周期拉平,消除静默失效。
# 用法(102 上 cron 每 2~5 分钟): scripts/ops/stripe-tunnel-url.sh
# 环境:BASE_URL / ADMIN_API_KEY / STRIPE_TUNNEL_CONTAINER(缺省 cf-stripe-boss)可覆盖。
set -u

BASE_URL="${BASE_URL:-http://192.168.0.102:28080}"
API="${BASE_URL}/api/admin/v1"
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
KEY="${ADMIN_API_KEY:-$(python3 -c "import json;print(json.load(open('$ROOT/.agents/skills/bossctl-cli/test-accounts.json'))['admin']['apiKeys'][0]['key']")}"
CONTAINER="${STRIPE_TUNNEL_CONTAINER:-cf-stripe-boss}"

# 1) 取隧道当前 URL(102 本机 docker 日志里的 trycloudflare 地址;cron 跑在 102)
URL="$(docker logs "$CONTAINER" 2>&1 | grep -oE 'https://[a-z0-9-]+\.trycloudflare\.com' | tail -1 | tr -d '\r')"
[ -n "$URL" ] || { echo "tunnel url not found in $CONTAINER logs"; exit 1; }
WANT="$URL/api/user/v1/webhooks/stripe"

# 2) 若已是最新则跳过
CUR="$(curl -sS -m 15 "$API/stripe-config" -H "X-API-Key: $KEY" \
  | python3 -c "import json,sys
try: print(json.load(sys.stdin)['data']['fields'].get('stripe.webhookUrl',{}).get('value',''))
except Exception: print('')")"
[ "$CUR" = "$WANT" ] && { echo "already up to date: $WANT"; exit 0; }

# 3) 写入 webhook 组(webhookUrl 非 secret,直接 PUT)
BODY="$(python3 -c "import json;print(json.dumps({'values':{'stripe.webhookUrl':'$WANT'}}))")"
RESP="$(curl -sS -m 15 -X PUT "$API/stripe-config/webhook" \
  -H "X-API-Key: $KEY" -H "Content-Type: application/json" -d "$BODY")"
echo "$RESP" | grep -q '"code":0\|"ok":true' \
  && echo "stripe.webhookUrl -> $WANT" \
  || { echo "update failed: $RESP"; exit 1; }
