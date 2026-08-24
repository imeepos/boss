#!/usr/bin/env bash
# E2E 登录脚本:发短信码→查库取码→登录→输出 token。
# 用法: scripts/e2e-login.sh [phone] [scene]
#   缺省 phone=13900001234, scene=login
# 输出 token 到 stdout,可通过 TOKEN=$(scripts/e2e-login.sh) 捕获。
#
# 依赖:ssh 免密 imeepos@192.168.0.102, curl 到 102:28080, ssh 有 docker 权限。
set -euo pipefail

PHONE="${1:-13900001234}"
SCENE="${2:-login}"
API="http://192.168.0.102:28080/api/user/v1"
SSH_HOST="imeepos@192.168.0.102"

# 检查前置依赖
command -v curl >/dev/null 2>&1 || { echo "curl 未安装" >&2; exit 1; }
ssh -o BatchMode=yes -o ConnectTimeout=3 "$SSH_HOST" "echo ok" >/dev/null 2>&1 || { echo "ssh 免密不通" >&2; exit 1; }

# 1. 发短信码
echo "==> 发码 $PHONE ($SCENE)" >&2
curl -s -m 5 -X POST "$API/auth/sms-code" \
  -H 'Content-Type: application/json' \
  -d "{\"phone\":\"$PHONE\",\"scene\":\"$SCENE\"}" >/dev/null
sleep 1

# 2. 查库取码
echo "==> 查库取码" >&2
CODE=$(ssh -o BatchMode=yes "$SSH_HOST" \
  "docker exec boss-infra-postgres-1 psql -U \"\$(docker exec boss-infra-postgres-1 printenv POSTGRES_USER)\" \
    -d \"\$(docker exec boss-infra-postgres-1 printenv POSTGRES_DB)\" \
    -tAc \"SELECT code FROM portal_sms_codes WHERE phone='$PHONE' AND scene='$SCENE' ORDER BY issued_at DESC LIMIT 1\"")
if [ -z "$CODE" ]; then
  echo "错误:未取到验证码" >&2; exit 1
fi
echo "==> 码=$CODE" >&2

# 3. 登录
echo "==> 登录" >&2
TOKEN=$(curl -s -m 5 -X POST "$API/auth/login" \
  -H 'Content-Type: application/json' \
  -d "{\"phone\":\"$PHONE\",\"mode\":\"sms\",\"smsCode\":\"$CODE\"}" \
  | python3 -c 'import sys,json;print(json.load(sys.stdin)["data"]["token"])' 2>/dev/null)
if [ -z "$TOKEN" ]; then
  echo "错误:登录失败" >&2; exit 1
fi
echo "==> token=$TOKEN" >&2
echo "$TOKEN"