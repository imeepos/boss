#!/usr/bin/env bash
# T18: TL1 login DELAY 追帧修复机械自测(单元证据 + 仿真器豁免移除静态断言 + go build/vet)。
# 用法: bash scripts/ops/tl1-login-delay-selftest.sh;任一 FAIL 退出码 1。
set -o pipefail
cd "$(dirname "$0")/../.." || exit 1
FAILS=0
LOG=/tmp/tl1-login-delay-selftest.log
: > "$LOG"

ok() { echo "PASS: $1"; }
bad() { echo "FAIL: $1"; tail -20 "$LOG"; FAILS=$((FAILS+1)); }

echo "== TL1 login DELAY selftest =="

# 1 单元证据: login 遇 DELAY 进入追帧而非报错;异 ctag 残帧丢弃;DELAY 后 DENY 仍 ErrAuth。
if go test -count=1 -run "TestDialLoginDelay|TestDialLoginStray" ./internal/domain/provision/tl1/ >>"$LOG" 2>&1; then
  ok "unit: login consumes DELAY frames instead of failing"
else
  bad "unit: login consumes DELAY frames instead of failing"
fi

# 2 静态断言: 仿真器对 LOGIN 的 DELAY 注入豁免已移除(state.go 不再区分 LOGIN)。
if grep -q "verb != .LOGIN." cmd/tl1sim/sim/state.go; then
  bad "static: sim LOGIN delay-exemption removed"
else
  ok "static: sim LOGIN delay-exemption removed"
fi

# 3 静态断言: session.login 含与 Do 同款 DELAY 追帧(case DELAY: continue)。
if sed -n "/func (s .Session. login/,/^}/p" internal/domain/provision/tl1/session.go | grep -q "case .DELAY.:"; then
  ok "static: session.login has Do-style DELAY follow-frame loop"
else
  bad "static: session.login has Do-style DELAY follow-frame loop"
fi

# 4 编译与静态检查。
if go build ./... >>"$LOG" 2>&1; then
  ok "go build ./..."
else
  bad "go build ./..."
fi
if go vet ./internal/domain/provision/tl1/... ./cmd/tl1sim/... >>"$LOG" 2>&1; then
  ok "go vet tl1 + tl1sim"
else
  bad "go vet tl1 + tl1sim"
fi

echo "== FAILS=$FAILS =="
if [ "$FAILS" -ne 0 ]; then
  echo "[tl1-login-delay] SELFTEST FAILED"
  exit 1
fi
echo "[tl1-login-delay] SELFTEST PASSED"
exit 0
