#!/usr/bin/env bash
# 构建 bossctl 二进制(BOSS API CLI)。
# 用法:
#   ./build.sh                     # 构建当前平台到 skill 的 assets/
#   GOOS=linux GOARCH=amd64 ./build.sh   # 交叉编译到指定平台
set -euo pipefail

# 定位仓库根目录(脚本位于 .agents/skills/bossctl-cli/scripts/)
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
SKILL_DIR="$(dirname "$SCRIPT_DIR")"
REPO_ROOT="$(cd "$SKILL_DIR/../../.." && pwd)"

GO="${GO:-}"
if [ -z "$GO" ]; then
  for cand in go /opt/homebrew/bin/go /usr/local/go/bin/go /usr/bin/go; do
    if command -v "$cand" >/dev/null 2>&1; then
      GO="$cand"
      break
    fi
  done
fi
if [ -z "$GO" ] || ! command -v "$GO" >/dev/null 2>&1; then
  echo "错误: 未找到 go 可执行文件,请设置 GO 环境变量" >&2
  exit 1
fi

OUT_NAME="bossctl-${GOOS:-$("$GO" env GOOS)}-${GOARCH:-$("$GO" env GOARCH)}"
OUT="${OUT:-$SKILL_DIR/assets/$OUT_NAME}"

cd "$REPO_ROOT"
VERSION="$(git describe --tags --always --dirty 2>/dev/null || echo dev)"
echo "构建 bossctl -> $OUT (GOOS=${GOOS:-$("$GO" env GOOS)} GOARCH=${GOARCH:-$("$GO" env GOARCH)} 版本=$VERSION)"
"$GO" build -ldflags="-s -w -X main.bossctlVersion=$VERSION" -o "$OUT" ./cmd/bossctl
chmod +x "$OUT"
echo "完成: $OUT"