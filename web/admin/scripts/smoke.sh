#!/usr/bin/env bash
# admin 前端冒烟:直连 Go server 断言 envelope 形状(login→me)。
# 前置: go run ./scripts/devseed && BOSS_HTTP_ADDR=:18080 go run ./cmd/server
# 用法: API=http://127.0.0.1:18080 web/admin/scripts/smoke.sh
set -euo pipefail
API="${API:-http://127.0.0.1:18080}"

login=$(curl -sf -X POST "$API/api/v1/auth/login" -H 'Content-Type: application/json' \
  -d '{"username":"admin","password":"admin123"}')
echo "$login" | grep -q '"code":0' || { echo "FAIL: login envelope code!=0"; exit 1; }
echo "$login" | grep -q '"token"' || { echo "FAIL: login 缺 token"; exit 1; }
echo "login envelope: OK"

token=$(echo "$login" | python3 -c 'import json,sys; print(json.load(sys.stdin)["data"]["token"])')
me=$(curl -sf "$API/api/v1/auth/me" -H "Authorization: Bearer $token")
for field in accountId roleCode realName; do
  echo "$me" | grep -q "\"$field\"" || { echo "FAIL: /auth/me 缺 $field"; exit 1; }
done
echo "me envelope: OK"
