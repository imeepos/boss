#!/usr/bin/env bash
# bossmcp(MCP server)真实环境冒烟(默认打 102 部署 http://192.168.0.102:28080):
#   资产二进制起 stdio 协议 → initialize/tools.list → 双端 whoami(真身份)
#   → 真实业务错误(42200 参数非法,错误渲染路径)→ worker /home(成功数据路径)。
# 直连 JSON-RPC 不经 LLM,零 token 可重复;密钥取 test-accounts.json 的 MCP 专用 key。
# 用法: bash scripts/mcp-smoke.sh [二进制路径](缺省按本机平台选 assets 预编译产物)
set -uo pipefail

ROOT=$(cd "$(dirname "$0")/.." && pwd)
SERVER="${BOSS_SERVER:-http://192.168.0.102:28080}"
KEYS="$ROOT/.agents/skills/bossctl-cli/test-accounts.json"
case "$(uname -s)-$(uname -m)" in
  Darwin-arm64) DEFAULT_BIN="$ROOT/.agents/skills/bossctl-cli/assets/bossmcp-darwin-arm64" ;;
  *)            DEFAULT_BIN="$ROOT/.agents/skills/bossctl-cli/assets/bossmcp-linux-amd64" ;;
esac
BIN="${1:-$DEFAULT_BIN}"
PASS_CNT=0; FAIL_CNT=0; LOG="/tmp/mcp-smoke-$(date +%s).log"

log() { echo "[$(date +%H:%M:%S)] $*" | tee -a "$LOG"; }
ok()  { echo "  PASS $*" | tee -a "$LOG"; PASS_CNT=$((PASS_CNT+1)); }
bad() { echo "  FAIL $*" | tee -a "$LOG"; FAIL_CNT=$((FAIL_CNT+1)); }

[ -x "$BIN" ] || { log "二进制不存在或不可执行: $BIN(先 make bossmcp 或用资产产物)"; exit 1; }
[ -f "$KEYS" ] || { log "密钥文件不存在: $KEYS"; exit 1; }

USER_KEY=$(python3 -c "import json;print(json.load(open('$KEYS'))['customers'][0]['apiKeys'][0]['key'])")
WORKER_KEY=$(python3 -c "import json;print(json.load(open('$KEYS'))['workers'][0]['apiKeys'][0]['key'])")

log "bossmcp 真实环境冒烟: bin=$(basename "$BIN") server=$SERVER"
OUT=$(mktemp); trap 'rm -f "$OUT"' EXIT
BOSS_SERVER="$SERVER" BOSS_USER_API_KEY="$USER_KEY" BOSS_WORKER_API_KEY="$WORKER_KEY" \
  "$BIN" >"$OUT" 2>/dev/null <<'PROTO'
{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-06-18"}}
{"jsonrpc":"2.0","id":2,"method":"tools/list"}
{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"boss_whoami","arguments":{"portal":"user"}}}
{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"boss_whoami","arguments":{"portal":"worker"}}}
{"jsonrpc":"2.0","id":5,"method":"tools/call","params":{"name":"boss_call","arguments":{"portal":"user","method":"POST","path":"/orders","body":{"productId":"0","addressId":"0","channelId":"0"}}}}
{"jsonrpc":"2.0","id":6,"method":"tools/call","params":{"name":"boss_call","arguments":{"portal":"worker","method":"GET","path":"/home"}}}
PROTO

# 断言经进程替换喂 while:计数器留在当前 shell(管道 while 进子shell 会丢计数)
while read -r status tag detail; do
  case "$status" in
    PASS) ok "$tag $detail" ;;
    *)    bad "$tag $detail" ;;
  esac
done < <(python3 - "$OUT" <<'PY'
import json, sys

lines = [json.loads(x) for x in open(sys.argv[1]) if x.strip()]
resp = {m.get("id"): m for m in lines}


def result(i):
    r = resp.get(i, {}).get("result") or {}
    return r, (r.get("content") or [{}])[0].get("text", "")


checks = []
r, t = result(1)
checks.append(("INIT", r.get("serverInfo", {}).get("name") == "bossmcp"
               and r.get("protocolVersion") == "2025-06-18", f"serverInfo={r.get('serverInfo')}"))
r2 = resp.get(2, {}).get("result") or {}
names = sorted(x.get("name") for x in r2.get("tools", []))
checks.append(("TOOLS", names == ["boss_call", "boss_routes", "boss_whoami"], f"tools={names}"))
r, t = result(3)
checks.append(("USER_ID", not r.get("isError") and "customerId" in t, t[:80]))
r, t = result(4)
checks.append(("WORKER_ID", not r.get("isError") and ("staffNo" in t or '"name"' in t), t[:80]))
r, t = result(5)
checks.append(("BIZ_ERR", r.get("isError") and "42200" in t, t[:80]))
r, t = result(6)
checks.append(("WORKER_HOME", not r.get("isError") and "code=0" in t, t[:80]))
for tag, passed, detail in checks:
    print(("PASS" if passed else "FAIL"), tag, detail.replace("\n", " "))
PY
)

log "结果: PASS=$PASS_CNT FAIL=$FAIL_CNT(日志 $LOG)"
[ "$FAIL_CNT" = "0" ] && [ "$PASS_CNT" = "6" ] || exit 1
log "bossmcp 冒烟全绿"
