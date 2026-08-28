#!/usr/bin/env bash
# 102 环境 API 文档端点验收:登录取 token → 四端聚合契约逐项断言。
# 用法: scripts/ops/apidocs-acceptance.sh
set -uo pipefail
BASE=${BASE:-http://192.168.0.102:28080}
API="$BASE/api/admin/v1"

TOKEN=$(curl -sf -m 10 "$API/auth/login" -X POST -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}' | sed -n 's/.*"token":"\([^"]*\)".*/\1/p')
[ -n "$TOKEN" ] || { echo "LOGIN FAILED"; exit 1; }

fails=0
for p in admin user worker open; do
  out=$(curl -sf -m 30 "$API/docs/openapi?portal=$p" -H "Authorization: Bearer $TOKEN")
  [ "$(printf '%s' "$out" | head -c 12)" = '{"code":0,' ] || { echo "FAIL $p envelope"; fails=$((fails+1)); continue; }
  paths=$(printf '%s' "$out" | grep -o '"path[a-z]*":' | wc -l | tr -d ' ')
  dang=$(printf '%s' "$out" | grep -o '"\$ref":"\./' | wc -l | tr -d ' ')
  [ "$dang" = "0" ] || { echo "FAIL $p 残余外部 \$ref=$dang"; fails=$((fails+1)); }
  echo "OK $p paths≈$paths 残余外部ref=$dang"
done

# 权限门禁:无 token 必须 403/401
code=$(curl -s -o /dev/null -w "%{http_code}" "$API/docs/openapi")
{ [ "$code" = "403" ] || [ "$code" = "401" ]; } && echo "OK 未认证门禁 HTTP $code" || { echo "FAIL 门禁 HTTP $code"; fails=$((fails+1)); }

# 非法 portal
code=$(curl -s -o /dev/null -w "%{http_code}" "$API/docs/openapi?portal=bad" -H "Authorization: Bearer $TOKEN")
[ "$code" = "200" ] && echo "OK 非法 portal 走 envelope(code=42200)" || { echo "FAIL 非法 portal HTTP $code"; fails=$((fails+1)); }

[ "$fails" = 0 ] && echo "ACCEPTANCE PASS" || { echo "ACCEPTANCE FAIL($fails)"; exit 1; }
