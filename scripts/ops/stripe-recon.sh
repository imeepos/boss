#!/usr/bin/env bash
# stripe-recon: Stripe 渠道流水与本地 payment 表逐笔对账(资金闸门第一道)。
# 用途:
#   1) 手动: 支付上线初期每日一跑,渠道账与系统账当晚对平;
#   2) cron: 稳定后挂 102 crontab(参考 patrol-cron.md 锚点表),差一分钱即红。
# 匹配键: paymentIntent.metadata.payNo ↔ payments.payNo(发起时作为幂等键回传)。
# 判定: PI status=succeeded ↔ payment status=SUCCESS 且金额分厘一致(amount*100)。
# 用法: scripts/ops/stripe-recon.sh
# 环境: BASE_URL(缺省 102:28080) / ADMIN_API_KEY(缺省取 test-accounts.json)
#       STRIPE_SK 必配(sk_test_.../sk_live_...;凭据权威在 admin 配置页 biz_params,
#       此处 env 传参仅为脚本运行,不落盘)。
# 自测: --selftest 用内置样例验证比对逻辑,不访问网络。
set -u

BASE_URL="${BASE_URL:-http://192.168.0.102:28080}"
SELFTEST=0
[ "${1:-}" = "--selftest" ] && SELFTEST=1

ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
API_KEY="${ADMIN_API_KEY:-$(python3 -c "import json;print(json.load(open('$ROOT/.agents/skills/bossctl-cli/test-accounts.json'))['admin']['apiKeys'][0]['key'])" 2>/dev/null || true)}"

# compare: 读两份 JSON 文件路径,输出对账结果;任一不一致 exit 1。
compare() {
  python3 - "$1" "$2" <<'PY'
import json, sys

def load(p):
    with open(p) as f:
        return json.load(f)

payments = load(sys.argv[1]).get("data", {}).get("items", [])
pis = load(sys.argv[2]).get("data", [])

# 口径裁定(2026-09-07 实证 PAY-20260829015320-06E1):method='card' 同时承载
# 柜面 POS 线下刷卡与 Stripe 线上卡通道两个语义;柜面动作带三要素
# (siteName/counterCode/operatorName,000167),三要素非空即剔除出 Stripe 对账范围。
counter_card = [p for p in payments
                if p.get("method") == "card"
                and (p.get("siteName") or p.get("counterCode") or p.get("operatorName"))]
# 本地侧: Stripe 线上卡通道 SUCCESS 流水(柜面 card 已剔除),键 payNo
local = {p.get("payNo"): p for p in payments
         if p.get("method") == "card" and p.get("status") == "SUCCESS"
         and not (p.get("siteName") or p.get("counterCode") or p.get("operatorName"))}
# 渠道侧: succeeded 的 PI,键 metadata.payNo
remote = {pi.get("metadata", {}).get("payNo"): pi for pi in pis
          if pi.get("status") == "succeeded" and pi.get("metadata", {}).get("payNo")}

missing_remote = sorted(set(local) - set(remote))
missing_local = sorted(set(remote) - set(local))
amount_bad = []
for k in sorted(set(local) & set(remote)):
    cents = round(local[k]["amount"] * 100)
    if cents != remote[k]["amount"]:
        amount_bad.append((k, cents, remote[k]["amount"]))

print(f"  本地 Stripe 口径 card SUCCESS: {len(local)} 条(柜面 POS card 另计 {len(counter_card)} 条,不对账) / 渠道 succeeded: {len(remote)} 条 / 对平: {len(set(local) & set(remote)) - len(amount_bad)} 条")
bad = 0
if missing_remote:
    bad += 1
    print(f"[recon] MISMATCH 本地收款渠道无流水 x{len(missing_remote)}: {missing_remote[:5]}")
if missing_local:
    bad += 1
    print(f"[recon] MISMATCH 渠道收款本地无落账 x{len(missing_local)}: {missing_local[:5]}")
if amount_bad:
    bad += 1
    print(f"[recon] MISMATCH 金额不一致 x{len(amount_bad)}: {amount_bad[:5]}")
if bad:
    sys.exit(1)
print("STRIPE-RECON OK: 渠道账与系统账逐笔对平")
PY
}

TMP="$(mktemp -d)"
trap 'rm -rf "$TMP"' EXIT
LOCAL_JSON="$TMP/local.json"
STRIPE_JSON="$TMP/stripe.json"

if [ "$SELFTEST" = 1 ]; then
  echo "[selftest] 全对平样例(含柜面 POS card 必须被剔除):"
  cat > "$LOCAL_JSON" <<'EOF'
{"code":0,"data":{"items":[
  {"payNo":"PAY-1","amount":299.00,"method":"card","status":"SUCCESS"},
  {"payNo":"PAY-2","amount":150.50,"method":"cash","status":"SUCCESS"},
  {"payNo":"PAY-3","amount":88.00,"method":"card","status":"SUCCESS",
   "siteName":"验收旗舰店","counterCode":"02","operatorName":"cashier_wang"}]}}
EOF
  cat > "$STRIPE_JSON" <<'EOF'
{"data":[
  {"id":"pi_1","amount":29900,"currency":"php","status":"succeeded","metadata":{"payNo":"PAY-1"}},
  {"id":"pi_2","amount":9900,"currency":"php","status":"canceled","metadata":{"payNo":"PAY-X"}}]}
EOF
  if compare "$LOCAL_JSON" "$STRIPE_JSON"; then echo "[selftest] pass"; else echo "[selftest] FAIL"; exit 1; fi

  echo "[selftest] 金额不一致样例(必须被拦):"
  cat > "$STRIPE_JSON" <<'EOF'
{"data":[{"id":"pi_1","amount":29000,"currency":"php","status":"succeeded","metadata":{"payNo":"PAY-1"}}]}
EOF
  if compare "$LOCAL_JSON" "$STRIPE_JSON" 2>/dev/null; then echo "[selftest] FAIL: 差额未拦截"; exit 1; fi
  echo "[selftest] pass"
  exit 0
fi

if [ -z "${STRIPE_SK:-}" ]; then
  echo "STRIPE_SK 未配置(权威凭据在 admin 配置页 biz_params;脚本运行请 export STRIPE_SK=sk_...)" >&2
  exit 2
fi

curl -sf -H "X-API-Key: $API_KEY" "$BASE_URL/api/admin/v1/payments" -o "$LOCAL_JSON" \
  || { echo "[recon] FETCH FAILED 本地 payments 拉取失败 $BASE_URL"; exit 1; }
curl -sf "https://api.stripe.com/v1/payment_intents?limit=100" -u "$STRIPE_SK:" -o "$STRIPE_JSON" \
  || { echo "[recon] FETCH FAILED Stripe payment_intents 拉取失败(检查 STRIPE_SK/网络)"; exit 1; }

compare "$LOCAL_JSON" "$STRIPE_JSON"
