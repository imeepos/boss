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

# 自助改密负例:错旧口令必须回 40100 业务码(信封 HTTP 恒 200),不改任何凭据。
cpw=$(curl -s -X POST "$API/api/v1/auth/change-password" -H "Authorization: Bearer $token" \
  -H 'Content-Type: application/json' -d '{"oldPassword":"definitely-wrong","newPassword":"whatever-9x"}')
echo "$cpw" | grep -q '"code":40100' || { echo "FAIL: change-password 错旧口令未回 40100: $cpw"; exit 1; }
echo "change-password negative: OK"
