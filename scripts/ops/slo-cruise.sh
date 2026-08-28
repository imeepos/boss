#!/usr/bin/env bash
# slo-cruise: 每日 SLO 巡航(采集+阈值判定+提醒中心告警+留痕) 与 周报汇总。
# 用法:
#   ./scripts/ops/slo-cruise.sh            # 今日巡航(默认,采集 102 并判定)
#   ./scripts/ops/slo-cruise.sh --weekly   # 汇总本月 JSONL 生成 markdown 周报
#   ./scripts/ops/slo-cruise.sh --selftest # 离线自检(不访问网络/DB)
# 依赖: 同目录 slo-collect.sh;阈值与环境见下。cron 应跑在 102 本机(双模式免 ssh)。
# 记录: 每日一条追加到 $SLO_JSONL(ts<TAB>json),周报读取全部记录聚合。
set -uo pipefail
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
BASE_URL="${BASE_URL:-http://192.168.0.102:28080}"
API="${BASE_URL}/api/admin/v1"
KEY="${ADMIN_API_KEY:-$(python3 -c "import json;print(json.load(open('$ROOT/.agents/skills/bossctl-cli/test-accounts.json'))['admin']['apiKeys'][0]['key'])" 2>/dev/null || true)}"
MONTH="$(date +%Y%m)"
SLO_JSONL="${SLO_JSONL:-/tmp/slo-cruise-$MONTH.jsonl}"
# 阈值(环境可覆盖,默认值见 docs/ops/patrol-cron.md 守夜锚点表)
SLO_TODO_ALERT="${SLO_TODO_ALERT:-1}"            # 待办超24h 数量 > 此值告警
SLO_UNBILLED_ALERT="${SLO_UNBILLED_ALERT:-100}"  # 近7d 未入账话单 > 此值告警
SLO_LAG_ALERT_SEC="${SLO_LAG_ALERT_SEC:-28800}"  # 日报滞后 > 8h 告警

# judge: stdin 读 <ts><TAB><json>,阈值判定;命中任一告警项 exit 1(供 cron 留痕)。
judge() {
  SLO_TODO_ALERT="$SLO_TODO_ALERT" SLO_UNBILLED_ALERT="$SLO_UNBILLED_ALERT" \
  SLO_LAG_ALERT_SEC="$SLO_LAG_ALERT_SEC" python3 -c '
import json, os, sys
todo_max = int(os.environ["SLO_TODO_ALERT"])
unbilled_max = int(os.environ["SLO_UNBILLED_ALERT"])
lag_max = int(os.environ["SLO_LAG_ALERT_SEC"])
alerts = []
for line in sys.stdin:
    line = line.rstrip("\n")
    if not line or "\t" not in line:
        continue
    ts, raw = line.split("\t", 1)
    try:
        d = json.loads(raw)
    except Exception as e:
        print(f"CRUISE FAIL: bad json {e}")
        sys.exit(1)
    todo_old = d.get("s5_admin_todo", {}).get("older_24h", 0)
    unbilled = d.get("s6_cdrs", {}).get("billing_unbilled", 0)
    lag = d.get("s7_reports", {}).get("lag_seconds", 0)
    ontime = d.get("s7_reports", {}).get("on_time_cnt", 0)
    s4 = d.get("s4_orders", {})
    created = s4.get("window_7d_created", 0)
    done = s4.get("done", 0)
    ratio = (done / created) if created else None
    todo_open = d.get("s5_admin_todo", {}).get("open", 0)
    cdr_total = d.get("s6_cdrs", {}).get("total", 0)
    ratio_txt = "n/a" if ratio is None else ratio
    print(f"[{ts}] s4 created={created} done={done} ratio={ratio_txt}")
    print(f"[{ts}] s5 todo_open={todo_open} todo_older_24h={todo_old}")
    print(f"[{ts}] s6 cdrs_total={cdr_total} billing_unbilled={unbilled}")
    print(f"[{ts}] s7 lag={lag}s on_time_cnt={ontime}")
    if todo_old > todo_max:
        alerts.append(f"待办积压: {todo_old} 条超 24h 未处理")
    if unbilled > unbilled_max:
        alerts.append(f"话单残留: {unbilled} 条近7d未入账")
    if lag > lag_max or ontime < 3:
        alerts.append(f"报表时效: lag={lag}s on_time_cnt={ontime}")
if alerts:
    print("CRUISE-WARN: " + "; ".join(alerts))
    sys.exit(1)
print("CRUISE-OK")
sys.exit(0)
'
}

# emit_alert: 复用提醒中心(与 stripe-tunnel 同源),refType=slo_cruise 按日幂等。
# 非 200(如 refType 白名单外/鉴权失败)必须留痕,禁止静默吞错。
emit_alert() {
  local title="$1" content="$2" level="${3:-WARN}" code
  [ -n "$KEY" ] || { echo "slo-cruise: no admin key, alert skipped: $title"; return 0; }
  code="$(curl -sS -m 10 -o /dev/null -w '%{http_code}' -X POST "$API/ops/notify-emit" \
    -H "X-API-Key: $KEY" -H "Content-Type: application/json" \
    -d "{\"refType\":\"slo_cruise\",\"refID\":\"$(date +%F)\",\"level\":\"$level\",\"title\":\"$title\",\"content\":\"$content\",\"link\":\"/intel/analytics\"}")"
  [ "$code" = "200" ] || echo "slo-cruise: alert emit failed http=$code: $title"
}

case "${1:-daily}" in
  --selftest)
    echo "[selftest] breach payload must WARN:"
    echo -e "$(date -u +%FT%TZ)\t{\"s4_orders\":{\"window_7d_created\":31,\"done\":4,\"cancelled\":15},\"s5_admin_todo\":{\"open\":4,\"older_24h\":3},\"s6_cdrs\":{\"total\":0,\"billing_unbilled\":0},\"s7_reports\":{\"lag_seconds\":0,\"on_time_cnt\":3}}" | judge
    if [ $? -ne 1 ]; then echo "[selftest] FAIL: breach not detected"; exit 1; fi
    echo "[selftest] clean payload must PASS:"
    echo -e "$(date -u +%FT%TZ)\t{\"s4_orders\":{\"window_7d_created\":10,\"done\":10,\"cancelled\":0},\"s5_admin_todo\":{\"open\":0,\"older_24h\":0},\"s6_cdrs\":{\"total\":5,\"billing_unbilled\":0},\"s7_reports\":{\"lag_seconds\":0,\"on_time_cnt\":3}}" | judge
    if [ $? -ne 0 ]; then echo "[selftest] FAIL: clean flagged"; exit 1; fi
    echo "[selftest] PASS"
    exit 0
    ;;
  --weekly)
    python3 -c '
import json, os, sys
path = os.environ["SLO_JSONL"]
if not os.path.exists(path) or os.path.getsize(path) == 0:
    print(f"NO-DATA: {path} 无记录"); sys.exit(1)
def num(d, keys):
    try:
        for k in keys: d = d[k]
        return float(d)
    except Exception: return None
days, alerts = [], 0
s4_ratios, lags, unbi = [], [], []
with open(path) as f:
    for line in f:
        line = line.rstrip("\n")
        if "\t" not in line: continue
        ts, raw = line.split("\t", 1)
        try: d = json.loads(raw)
        except Exception: continue
        days.append(ts[:10])
        ratio = num(d, ["s4_orders","window_7d_created"])
        done = num(d, ["s4_orders","done"])
        s4_ratios.append(round(done/ratio,3) if ratio else None)
        lags.append(num(d, ["s7_reports","lag_seconds"]))
        unbi.append(num(d, ["s6_cdrs","billing_unbilled"]))
        if d.get("s5_admin_todo",{}).get("older_24h",0) > 0: alerts += 1
print("# SLO 巡航周报")
print()
print(f"- 记录区间: {days[0]} ~ {days[-1]} (共 {len(days)} 天)")
print(f"- 告警天数(待办积压): {alerts}")
def fmt(vs): return " / ".join(str(v) for v in vs if v is not None) or "-"
print()
print("| 指标 | 各日取值 |")
print("| --- | --- |")
print(f"| S4 7d 完成率 | {fmt(s4_ratios)} |")
print(f"| S6 未入账话单 | {fmt(unbi)} |")
print(f"| S7 日报滞后 s | {fmt(lags)} |")
' 2>&1
    RC=$?
    exit $RC
    ;;
  daily|*)
    RAW="$("$(dirname "$0")/slo-collect.sh" 2>&1)"
    if [ $? -ne 0 ] || ! echo "$RAW" | python3 -c "import json,sys; json.loads(sys.stdin.read())" >/dev/null 2>&1; then
      emit_alert "SLO 采集失败" "slo-collect 异常: $RAW" URGENT
      echo "CRUISE-FAIL: collect error"; exit 1
    fi
    TS="$(date -u +%Y-%m-%dT%H:%M:%SZ)"
    printf '%s\t%s\n' "$TS" "$RAW" >> "$SLO_JSONL" || { emit_alert "SLO 留痕失败" "jsonl 写失败: $SLO_JSONL" URGENT; exit 1; }
    OUT="$(printf '%s\t%s\n' "$TS" "$RAW" | judge)"
    RC=$?
    echo "$OUT"
    if [ $RC -ne 0 ]; then
      TITLE=$(echo "$OUT" | grep '^CRUISE-WARN' | sed 's/^CRUISE-WARN: //')
      emit_alert "SLO 巡航告警" "$TITLE" WARN
      exit 1
    fi
    exit 0
    ;;
esac