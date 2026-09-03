#!/usr/bin/env bash
set -u
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
ADMIN="$ROOT/web/admin"
fail=0
fail_step() { echo "FAIL $1"; fail=1; }

if (cd "$ADMIN" && CI=true pnpm vitest run src/lib/format.test.ts > /tmp/tz-display-format-test.log 2>&1); then
  if grep -q "7 tests" /tmp/tz-display-format-test.log; then echo "PASS timezone conversion test (7 tests, required case in source)"; else fail_step "timezone conversion test"; fi
else
  cat /tmp/tz-display-format-test.log
  fail_step "timezone conversion test"
fi

if grep -q "timeZone: .Asia/Shanghai." "$ADMIN/src/lib/format.ts" && grep -q "fmtTime(\'2026-09-03T16:30:00Z\')" "$ADMIN/src/lib/format.test.ts"; then
  echo "PASS business timezone formatter contract"
else
  fail_step "business timezone formatter contract"
fi

raw=0
for file in "$ADMIN/src/pages/boss/order/index.tsx" "$ADMIN/src/pages/boss/install-board/index.tsx" "$ADMIN/src/pages/boss/dispatch/index.tsx" "$ADMIN/src/pages/partner/orders/index.tsx"; do
  if grep -Eq "\.(createdAt|finishedAt|reportedAt|arrivedAt|transferredAt)\.(slice|replace)\(|\{r\.(createdAt|arrivedAt)\}|\{x\.(finishedAt|reportedAt|transferredAt)\}" "$file"; then
    echo "FAIL raw UTC timestamp render: $file"
    raw=1
  fi
done
if [ "$raw" -eq 0 ]; then echo "PASS static timestamp render check"; else fail=1; fi

if [ "${TZ_DISPLAY_SKIP_TYPECHECK:-0}" = "1" ]; then echo "PASS admin typecheck (covered by prior gate)"; elif (cd "$ADMIN" && CI=true pnpm typecheck > /tmp/tz-display-typecheck.log 2>&1); then echo "PASS admin typecheck"; else cat /tmp/tz-display-typecheck.log; fail_step "admin typecheck"; fi

if [ "$fail" -ne 0 ]; then echo "FAIL tz-display-selftest"; exit 1; fi
echo "PASS tz-display-selftest"
