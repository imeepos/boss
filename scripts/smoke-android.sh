#!/usr/bin/env bash
# Android 双端真机冒烟:安装(可选)+启动+UI dump+崩溃扫描,固化人工验证序列。
# 用法:
#   scripts/smoke-android.sh [--serial S] [--app user|worker|both] [--skip-install]
#   scripts/smoke-android.sh --app user --serial ee9999eb
# 前置: 设备已连接 adb;APK 已构建(或省略 --skip-install 由脚本直接安装已有产物)。
# 产物: /tmp/smoke-android/<app>-launch.png + <app>-dump.xml;崩溃即 exit 1。
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ADB="${ANDROID_HOME:-$HOME/Library/Android/sdk}/platform-tools/adb"
SERIAL=""
APPS="both"
SKIP_INSTALL=0

usage() { sed -n '2,8p' "$0"; exit 0; }
while [ $# -gt 0 ]; do
  case "$1" in
    --serial) SERIAL="$2"; shift 2;;
    --app) APPS="$2"; shift 2;;
    --skip-install) SKIP_INSTALL=1; shift;;
    -h|--help) usage;;
    *) echo "未知参数: $1" >&2; exit 1;;
  esac
done

pkg_of() { case "$1" in user) echo "com.ymm.boss.user";; worker) echo "com.ymm.boss.worker";; esac; }
act_of() { case "$1" in user) echo "com.ymm.boss.user/.MainActivity";; worker) echo "com.ymm.boss.worker/.MainActivity";; esac; }
apk_of() { case "$1" in user) echo "$ROOT_DIR/mobile/user/android/app/build/outputs/apk/debug/app-debug.apk";; worker) echo "$ROOT_DIR/mobile/worker/android/app/build/outputs/apk/debug/app-debug.apk";; esac; }

ADB_CMD="$ADB"
[ -n "$SERIAL" ] && ADB_CMD="$ADB -s $SERIAL"

$ADB_CMD get-state >/dev/null || { echo "[smoke] FAIL 无可用设备" >&2; exit 1; }

OUT_DIR="/tmp/smoke-android"
mkdir -p "$OUT_DIR"

[ "$APPS" = "both" ] && APPS="user worker"
RC=0
for app in $APPS; do
  echo "[smoke] == $app =="
  APK_PATH="$(apk_of "$app")"
  if [ "$SKIP_INSTALL" -eq 0 ]; then
    if [ ! -f "$APK_PATH" ]; then
      echo "[smoke] FAIL APK 不存在: $APK_PATH(先 assembleDebug)" >&2
      RC=1
      continue
    fi
    if ! $ADB_CMD install -r "$APK_PATH" | grep -q Success; then
      echo "[smoke] FAIL 安装失败" >&2
      RC=1
      continue
    fi
  fi
  $ADB_CMD logcat -c
  $ADB_CMD shell am force-stop "$(pkg_of "$app")" || true
  $ADB_CMD shell am start -n "$(act_of "$app")" >/dev/null
  sleep 6
  $ADB_CMD exec-out screencap -p > "$OUT_DIR/$app-launch.png"
  $ADB_CMD shell uiautomator dump "/sdcard/smoke-$app.xml" >/dev/null 2>&1 || true
  $ADB_CMD shell cat "/sdcard/smoke-$app.xml" > "$OUT_DIR/$app-dump.xml" 2>/dev/null || true
  local_texts=$(grep -o 'text="[^"]\+"' "$OUT_DIR/$app-dump.xml" 2>/dev/null | head -5 || true)
  fatal_count=$($ADB_CMD logcat -d 2>/dev/null | grep -c 'FATAL EXCEPTION' || true)
  echo "[smoke] $app 前台文本采样: ${local_texts:-<无>}"
  echo "[smoke] $app FATAL EXCEPTION = $fatal_count"
  if [ "${fatal_count:-1}" -ne 0 ]; then
    echo "[smoke] FAIL $app 检测到崩溃,详见 $OUT_DIR/$app-launch.png" >&2
    RC=1
  fi
done
[ "$RC" -eq 0 ] && echo "[smoke] PASS 冒烟通过(截图与 dump 在 $OUT_DIR)"
exit "$RC"
