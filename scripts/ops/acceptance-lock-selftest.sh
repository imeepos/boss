#!/usr/bin/env bash
set -u
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
LOCK="${1:-/tmp/boss-102-acceptance-lock-selftest.lock}"
export ACCEPTANCE_LOCK_PATH="$LOCK"
source "$ROOT/scripts/ops/acceptance-lock.sh"
acquire_acceptance_lock || exit 1
trap release_acceptance_lock EXIT
sleep "${SELFTEST_SLEEP:-3}"
echo "SELFTEST_PASS pid=$$"
