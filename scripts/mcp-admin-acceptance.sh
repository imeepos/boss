#!/usr/bin/env bash
# admin MCP 真实环境验收(102 部署,无 mock;全部经 bossmcp stdio 协议,不经 LLM):
#   1. 受权业务账号(调度/财务/网维)发现并调用权限内接口,拿到真实业务数据
#   2. 权限外接口 403 且拒绝信息可读;目录面不含账号无权的接口
#   3. 数据范围:两个不同组织账号互查,互相看不到对方数据(成对留证)
#   4. 对比验证:同一操作,有权账号成功/无权账号被拒,成对归档
#   sysadmin 不参与本验收(其成功不作为通过依据);全程只读,零造数。
# 用法: bash scripts/mcp-admin-acceptance.sh [二进制路径](缺省按平台选 assets 产物)
# 注: macOS bash 3.2 不支持过程替换内嵌 heredoc,断言脚本先落盘再引用。
set -uo pipefail

ROOT=$(cd "$(dirname "$0")/.." && pwd)
SERVER="${BOSS_SERVER:-http://192.168.0.102:28080}"
KEYS="$ROOT/.agents/skills/bossctl-cli/test-accounts.json"
case "$(uname -s)-$(uname -m)" in
  Darwin-arm64) DEFAULT_BIN="$ROOT/.agents/skills/bossctl-cli/assets/bossmcp-darwin-arm64" ;;
  *)            DEFAULT_BIN="$ROOT/.agents/skills/bossctl-cli/assets/bossmcp-linux-amd64" ;;
esac
BIN="${1:-$DEFAULT_BIN}"
PASS_CNT=0; FAIL_CNT=0; LOG="/tmp/mcp-admin-acc-$(date +%s).log"
MARKERS="/tmp/mcp-admin-acc-markers.json"
ASSERT="/tmp/mcp-admin-acc-assert.py"

log() { echo "[$(date +%H:%M:%S)] $*" | tee -a "$LOG"; }
ok()  { echo "  PASS $*" | tee -a "$LOG"; PASS_CNT=$((PASS_CNT+1)); }
bad() { echo "  FAIL $*" | tee -a "$LOG"; FAIL_CNT=$((FAIL_CNT+1)); }
key() { python3 -c "import json;print(json.load(open('$KEYS'))['$1']['apiKeys'][0]['key'])"; }

# judge <账号名>:断言行经过程替换喂入,计数器留在当前 shell(管道会进子shell 丢计数)
judge() {
  local who="$1"
  while read -r status tag detail; do
    case "$status" in
      PASS) ok "[$who] $tag $detail" ;;
      *)    bad "[$who] $tag $detail" ;;
    esac
  done
}

# batch <账号名> <协议文件>:起 bossmcp 会话,输出存 BATCH_OUT
batch() {
  BATCH_OUT=$(mktemp)
  BOSS_SERVER="$SERVER" BOSS_ADMIN_API_KEY="$(key "$1")" "$BIN" <"$2" >"$BATCH_OUT" 2>/dev/null
}

# proto <文件>:收协议行(stdin)
proto() { cat >"$1"; }

[ -x "$BIN" ] || { log "二进制不存在: $BIN(先 make bossmcp)"; exit 1; }
[ -f "$KEYS" ] || { log "密钥文件不存在: $KEYS"; exit 1; }

# ---------- 断言脚本(模式参数: A/B/C/D/E) ----------
cat >"$ASSERT" <<'PYEOF'
import json, re, sys

mode, path = sys.argv[1], sys.argv[2]
lines = [json.loads(x) for x in open(path) if x.strip()]
resp = {m.get("id"): m for m in lines}


def res(i):
    r = resp.get(i, {}).get("result") or {}
    return r, (r.get("content") or [{}])[0].get("text", "")


def emit(tag, passed, detail):
    print(("PASS" if passed else "FAIL"), tag, detail.replace("\n", " "))


if mode == "A":
    r, t = res(1)
    emit("WHOAMI", not r.get("isError") and "dispatch_li" in t, t[:60])
    r, t = res(2)
    has_order = "GET    /orders" in t
    forbidden = {"GET    /bills": "menu:billing", "GET    /ports": "menu:resource",
                 "POST   /api-keys": "menu:apikey", "GET    /accounts": "menu:account"}
    leak = [p for p in forbidden if p in t]
    emit("CATALOG", not r.get("isError") and has_order and not leak,
         f"含/orders={has_order} 越权泄漏={leak or '无'}")
    r, t = res(3)
    emit("POS_ORDERS", not r.get("isError") and "code=0" in t and "orderNo" in t, t[:70])
    r, t = res(4)
    emit("NEG_BILLS_403", r.get("isError") and "403" in t and "menu:billing" in t, t[:70])
    r, t = res(5)
    emit("NEG_PORTS_403", r.get("isError") and "403" in t and "menu:resource" in t, t[:70])

elif mode == "B":
    r, t = res(1)
    emit("POS_BILLS", not r.get("isError") and "code=0" in t, t[:70])
    r, t = res(2)
    emit("NEG_PORTS_403", r.get("isError") and "403" in t and "menu:resource" in t, t[:70])
    r, t = res(3)
    emit("NEG_APIKEYS_403", r.get("isError") and "403" in t and "menu:apikey" in t, t[:70])

elif mode == "C":
    r, t = res(1)
    emit("POS_PORTS", not r.get("isError") and "code=0" in t, t[:70])
    r, t = res(2)
    emit("NEG_BILLS_403", r.get("isError") and "403" in t and "menu:billing" in t, t[:70])

elif mode == "D":
    markers = {}
    r, t = res(1)
    orders = re.findall(r'"orderNo":\s*"([^"]+)"', t)
    if orders:
        markers["xu_order"] = orders[0]
    r, t = res(2)
    customers = re.findall(r'"name":\s*"([^"]+)"', t)
    if customers:
        markers["xu_cust_sample"] = customers[:3]
    emit("POS_ORDERS", len(orders) > 0, f"本组织订单 {len(orders)} 条, 首单 {orders[0] if orders else '-'}")
    emit("POS_CUSTOMERS", bool(customers), f"本组织客户样例 {customers[:3]}")
    json.dump(markers, open(sys.argv[3], "w"))

elif mode == "E":
    markers = json.load(open(sys.argv[3]))
    r, t = res(1)
    orders = re.findall(r'"orderNo":\s*"([^"]+)"', t)
    xu_order = markers.get("xu_order", "")
    leak = bool(xu_order) and xu_order in orders
    emit("ISO_ORDERS", not r.get("isError") and not leak,
         f"对方首单 {xu_order or '-'} 泄漏={leak}(本组织可见 {len(orders)} 条)")
    r, t = res(2)
    customers = re.findall(r'"name":\s*"([^"]+)"', t)
    leak_c = [c for c in markers.get("xu_cust_sample", []) if c in customers]
    emit("ISO_CUSTOMERS", not r.get("isError") and not leak_c,
         f"对方客户样例泄漏={leak_c or '无'}(本组织可见 {len(customers)} 条)")
PYEOF

log "admin MCP 真实环境验收: bin=$(basename "$BIN") server=$SERVER"
log "[口径] sysadmin 全权账号不参与任何断言;以下全部为业务岗位受权账号"
log "[口径] 全程只读探测,零造数"

# ---------- A. 调度(dispatch_li):目录过滤 + 权限内成功 + 权限外 403 ----------
log "A 调度账号 dispatch_li(李调度)"
proto /tmp/acc-a.proto <<'PROTO'
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"boss_whoami","arguments":{"portal":"admin"}}}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"boss_routes","arguments":{"portal":"admin"}}}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"boss_call","arguments":{"portal":"admin","method":"GET","path":"/orders","query":{"page":"1"}}}}
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"boss_call","arguments":{"portal":"admin","method":"GET","path":"/bills"}}}
{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"boss_call","arguments":{"portal":"admin","method":"GET","path":"/ports"}}}
PROTO
batch dispatch_li /tmp/acc-a.proto
judge dispatch_li < <(python3 "$ASSERT" A "$BATCH_OUT")
rm -f "$BATCH_OUT" /tmp/acc-a.proto

# ---------- B. 财务(cashier_wang):计费域成功 + 对比反例 ----------
log "B 财务账号 cashier_wang(王财务)"
proto /tmp/acc-b.proto <<'PROTO'
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"boss_call","arguments":{"portal":"admin","method":"GET","path":"/bills","query":{"page":"1"}}}}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"boss_call","arguments":{"portal":"admin","method":"GET","path":"/ports"}}}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"boss_call","arguments":{"portal":"admin","method":"GET","path":"/api-keys"}}}
PROTO
batch cashier_wang /tmp/acc-b.proto
judge cashier_wang < <(python3 "$ASSERT" B "$BATCH_OUT")
rm -f "$BATCH_OUT" /tmp/acc-b.proto

# ---------- C. 网维(noc_chen):资源域成功 + 对比反例 ----------
log "C 网维账号 noc_chen(陈网维)"
proto /tmp/acc-c.proto <<'PROTO'
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"boss_call","arguments":{"portal":"admin","method":"GET","path":"/ports"}}}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"boss_call","arguments":{"portal":"admin","method":"GET","path":"/bills"}}}
PROTO
batch noc_chen /tmp/acc-c.proto
judge noc_chen < <(python3 "$ASSERT" C "$BATCH_OUT")
rm -f "$BATCH_OUT" /tmp/acc-c.proto

# ---------- D/E. 数据范围:kefu_xu(主品牌) vs kefu_zhao(家庭宽带) 互查 ----------
log "D 数据范围 kefu_xu(主品牌·企业)"
proto /tmp/acc-d.proto <<'PROTO'
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"boss_call","arguments":{"portal":"admin","method":"GET","path":"/orders","query":{"page":"1"}}}}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"boss_call","arguments":{"portal":"admin","method":"GET","path":"/customers"}}}
PROTO
batch kefu_xu /tmp/acc-d.proto
judge kefu_xu < <(python3 "$ASSERT" D "$BATCH_OUT" "$MARKERS")
rm -f "$BATCH_OUT" /tmp/acc-d.proto

log "E 数据范围 kefu_zhao(家庭宽带)互查"
proto /tmp/acc-e.proto <<'PROTO'
{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"boss_call","arguments":{"portal":"admin","method":"GET","path":"/orders","query":{"page":"1"}}}}
{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"boss_call","arguments":{"portal":"admin","method":"GET","path":"/customers"}}}
PROTO
batch kefu_zhao /tmp/acc-e.proto
judge kefu_zhao < <(python3 "$ASSERT" E "$BATCH_OUT" "$MARKERS")
rm -f "$BATCH_OUT" "$MARKERS" /tmp/acc-e.proto "$ASSERT"

log "结果: PASS=$PASS_CNT FAIL=$FAIL_CNT(日志 $LOG)"
[ "$FAIL_CNT" = "0" ] && exit 0
log "admin MCP 验收存在失败项"
exit 1
