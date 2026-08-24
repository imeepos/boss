#!/usr/bin/env bash
# 构建并通过 ADB 安装 user Android 包。
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ANDROID_DIR="$ROOT_DIR/mobile/user/android"
PACKAGE_NAME="com.ymm.boss.user"
BUILD_TYPE="debug"
DEVICE=""
BASE_URL=""

usage() {
  cat <<'EOF'
用法: scripts/build-install-user-android.sh [选项]

选项:
  --release             构建 release；默认构建 debug
  --device SERIAL       指定 adb serial
  --base-url URL        覆盖 bossBaseUrl
  --connected           构建后执行 connectedDebugAndroidTest + 断言 tests 数>0
  --no-install          只构建，不安装
  -h, --help            显示帮助

示例:
  scripts/build-install-user-android.sh
  scripts/build-install-user-android.sh --device EAORGQYL45XGJJCY
  scripts/build-install-user-android.sh --release --no-install
EOF
}

find_adb() {
  if command -v adb >/dev/null 2>&1; then
    command -v adb
    return
  fi
  local candidate="$HOME/Library/Android/sdk/platform-tools/adb"
  [[ -x "$candidate" ]] || { echo "错误: 未找到 adb" >&2; exit 1; }
  echo "$candidate"
}

find_java_home() {
  if [[ -n "${JAVA_HOME:-}" && -x "$JAVA_HOME/bin/java" ]]; then
    echo "$JAVA_HOME"
    return
  fi
  local candidate
  candidate="$(find /opt/homebrew/Cellar/openjdk@17 -path '*/Contents/Home/bin/java' -print -quit 2>/dev/null || true)"
  [[ -n "$candidate" ]] || { echo "错误: 未找到 JDK 17，请设置 JAVA_HOME" >&2; exit 1; }
  dirname "$(dirname "$candidate")"
}

select_device() {
  local adb="$1"
  [[ -n "$DEVICE" ]] && return
  local devices
  devices="$($adb devices | awk 'NR > 1 && $2 == "device" {print $1}')"
  local device_list=()
  while IFS= read -r line; do
    [[ -n "$line" ]] && device_list+=("$line")
  done <<<"$devices"
  if [[ "${#device_list[@]}" -eq 1 ]]; then
    DEVICE="${device_list[0]}"
  elif [[ "${#device_list[@]}" -eq 0 ]]; then
    echo "错误: 没有可用的 adb device，请连接手机并开启 USB 调试" >&2
    exit 1
  else
    local usb_devices
    usb_devices="$($adb devices -l | awk '/usb:/ && $2 == "device" {print $1}')"
    local usb_list=()
    while IFS= read -r line; do
      [[ -n "$line" ]] && usb_list+=("$line")
    done <<<"$usb_devices"
    if [[ "${#usb_list[@]}" -eq 1 ]]; then
      DEVICE="${usb_list[0]}"
    else
      echo "错误: 检测到多个设备，请使用 --device SERIAL 指定" >&2
      $adb devices -l >&2
      exit 1
    fi
  fi
}

while [[ $# -gt 0 ]]; do
  case "$1" in
    --release) BUILD_TYPE="release"; shift ;;
    --device) [[ $# -ge 2 ]] || { echo "错误: --device 缺少 SERIAL" >&2; exit 1; }; DEVICE="$2"; shift 2 ;;
    --base-url) [[ $# -ge 2 ]] || { echo "错误: --base-url 缺少 URL" >&2; exit 1; }; BASE_URL="$2"; shift 2 ;;
    --connected) CONNECTED=1; shift ;;
    --no-install) NO_INSTALL=1; shift ;;
    -h|--help) usage; exit 0 ;;
    *) echo "错误: 未知选项 $1" >&2; usage >&2; exit 1 ;;
  esac
done

ADB="$(find_adb)"
JAVA_HOME="$(find_java_home)"
export JAVA_HOME

case "$BUILD_TYPE" in
  debug) GRADLE_TASK="assembleDebug" ;;
  release) GRADLE_TASK="assembleRelease" ;;
esac
GRADLE_ARGS=("$GRADLE_TASK")
[[ -n "$BASE_URL" ]] && GRADLE_ARGS+=("-PbossBaseUrl=$BASE_URL")
echo "==> 构建 user Android $BUILD_TYPE"
(cd "$ANDROID_DIR" && ./gradlew "${GRADLE_ARGS[@]}")

if [[ "${CONNECTED:-0}" == 1 ]]; then
  echo "==> 执行 connectedDebugAndroidTest"
  (cd "$ANDROID_DIR" && ./gradlew connectedDebugAndroidTest)
  THIS_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
  "$THIS_DIR/after-connected-test.sh" "mobile/user/android"
fi

APK="$ANDROID_DIR/app/build/outputs/apk/$BUILD_TYPE/app-$BUILD_TYPE.apk"
[[ -f "$APK" ]] || { echo "错误: APK 不存在: $APK" >&2; exit 1; }
echo "APK: $APK"

if [[ "${NO_INSTALL:-0}" == 1 ]]; then
  exit 0
fi

select_device "$ADB"
echo "==> 安装到 $DEVICE"
"$ADB" -s "$DEVICE" install -r "$APK"
echo "==> 验证 $PACKAGE_NAME"
"$ADB" -s "$DEVICE" shell pm path "$PACKAGE_NAME"
echo "完成: $DEVICE"
