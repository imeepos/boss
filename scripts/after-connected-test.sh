#!/usr/bin/env bash
# after-connected-test.sh — connectedDebugAndroidTest 后断言 tests 数>0,防静默空跑。
# 用法: 在 connectedDebugAndroidTest 后调用:
#   scripts/after-connected-test.sh [mobile/user/android|mobile/worker/android]
# 缺省检测 mobile/user/android。
set -euo pipefail

DIR="${1:-mobile/user/android}"
cd "$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"

RESULT_DIR="$DIR/app/build/outputs/androidTest-results/connected"
if [ ! -d "$RESULT_DIR" ]; then
  echo "错误: 未找到测试结果目录 $RESULT_DIR" >&2
  echo "请先执行 connectedDebugAndroidTest" >&2
  exit 1
fi

TOTAL_TESTS=0
TOTAL_FAILURES=0
for xml in "$RESULT_DIR"/*.xml; do
  [ -f "$xml" ] || continue
  tests=$(grep -o 'tests="[0-9]*"' "$xml" | head -1 | grep -o '[0-9]*')
  failures=$(grep -o 'failures="[0-9]*"' "$xml" | head -1 | grep -o '[0-9]*')
  TOTAL_TESTS=$((TOTAL_TESTS + tests))
  TOTAL_FAILURES=$((TOTAL_FAILURES + failures))
  echo "  $xml: tests=$tests failures=$failures" >&2
done

echo "==> 汇总: tests=$TOTAL_TESTS failures=$TOTAL_FAILURES" >&2

if [ "$TOTAL_TESTS" -eq 0 ]; then
  echo "错误: tests=0 — 可能 testInstrumentationRunner 未正确声明或测试被静默跳过" >&2
  echo "请检查 build.gradle.kts defaultConfig 的 testInstrumentationRunner" >&2
  exit 1
fi

if [ "$TOTAL_FAILURES" -gt 0 ]; then
  echo "警告: 检测到 $TOTAL_FAILURES 个失败用例,请查看 XML 详情" >&2
  exit 1
fi

echo "OK: $TOTAL_TESTS 个用例全部通过" >&2