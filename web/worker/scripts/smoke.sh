#!/usr/bin/env bash
# 师傅端冒烟:起 vite preview(代理到 mock),断言全部页面与关键 API 均可达。
# 用法: bash scripts/smoke.sh  (BASE 可覆盖,默认 http://localhost:4174)
set -euo pipefail
cd "$(dirname "$0")/.."

BASE="${BASE:-http://localhost:4174}"
PAGES="index home login orders order hall pickup transfer reschedule checkin navi history
scan scanabnormal photo report activate sign charge replace dismantle repair complaint
maintenance safety profile performance schedule settings feedback messages notice help
service retire tool"

fail=0
for p in $PAGES; do
  code=$(curl -s -o /dev/null -w '%{http_code}' "$BASE/$p.html")
  if [ "$code" != "200" ]; then echo "PAGE FAIL $p.html -> $code"; fail=1; fi
done

for ep in home tickets hall materials profile messages notices; do
  code=$(curl -s -o /dev/null -w '%{http_code}' -H 'Authorization: Bearer smoke' \
    "$BASE/api/worker/v1/$ep")
  if [ "$code" != "200" ]; then echo "API FAIL /$ep -> $code"; fail=1; fi
done

[ "$fail" = 0 ] && echo "SMOKE OK ($BASE)"
exit $fail
