#!/usr/bin/env bash
# 签发 release-platform 离线授权令牌(供 boss 部署激活/集成测试使用)。
# 用法:
#   BOSS_RELEASE_API=http://192.168.0.102:38080 \
#   ISSUER_URL=http://192.168.0.102:4862 \
#   scripts/issue-offline-token.sh <activation_code> <product_id> <device_id> <fingerprint>
# 输出:离线令牌 JSON 到 stdout(重定向到 /tmp/offline-token.json 供集成测试)。
set -euo pipefail

API="${BOSS_RELEASE_API:-http://192.168.0.102:38080}"
ISSUER="${ISSUER_URL:-http://192.168.0.102:4862}"
CLIENT_ID="${IDENT_CLIENT_ID:-smoke}"
USER="${LICENSE_ISSUER_USER:-admin}"
PASS="${LICENSE_ISSUER_PASS:-admin-pw}"

CODE="${1:?usage: issue-offline-token.sh <activation_code> <product_id> <device_id> <fingerprint>}"
PRODUCT="${2:?product_id required}"
DEVICE="${3:?device_id required}"
FP="${4:?fingerprint required}"

TOKEN=$(curl -fsS -m 10 -X POST -d "grant_type=password&username=$USER&password=$PASS&client_id=$CLIENT_ID" \
  "$ISSUER/token" | python3 -c 'import json,sys; print(json.load(sys.stdin)["access_token"])')

echo "[license] exchange activation code..." >&2
LIC=$(curl -fsS -m 10 -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d "{\"activation_code\":\"$CODE\",\"product_id\":\"$PRODUCT\",\"device_id\":\"$DEVICE\",\"fingerprint_hash\":\"$FP\"}" \
  "$API/v1/activations" | python3 -c 'import json,sys; print(json.load(sys.stdin)["license_id"])')

echo "[license] issue offline token for $LIC..." >&2
curl -fsS -m 10 -X POST -H "Authorization: Bearer $TOKEN" -H "Content-Type: application/json" \
  -d "{\"device_id\":\"$DEVICE\",\"fingerprint_hash\":\"$FP\"}" \
  "$API/v1/licenses/$LIC/offline-token" | python3 -m json.tool