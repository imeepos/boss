#!/usr/bin/env bash
# user Android release 独立签名基建:生成 keystore / 出包 / apksigner 验签。
# 背景(2026-08-27 上线计划 D0):release 此前无 keystore 时回落 debug 签名,
# 内网直装首发必须独立 release 签名,上线 checklist 第一项即 apksigner verify。
set -euo pipefail

ROOT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
ANDROID_DIR="$ROOT_DIR/mobile/user/android"
KEYSTORE_FILE="$ANDROID_DIR/release.keystore"
PROPS_FILE="$ANDROID_DIR/keystore.properties"
ALIAS="boss-user-release"

usage() {
  cat <<'EOF'
用法: scripts/user-android-release-sign.sh [--init]

  --init   生成 release keystore + keystore.properties(交互输密码,不落仓库)
  (无参数) 构建 assembleRelease 并用 apksigner 验证:签名有效 + 指纹对比 debug 包
           (证明非 debug 签名),然后打印证书指纹供上线 checklist 留档
EOF
}

find_java_home() {
  if [[ -n "${JAVA_HOME:-}" && -x "$JAVA_HOME/bin/keytool" ]]; then
    echo "$JAVA_HOME"
    return
  fi
  local candidate
  candidate="$(find /opt/homebrew/Cellar/openjdk@17 -path '*/Contents/Home/bin/keytool' -print -quit 2>/dev/null || true)"
  [[ -n "$candidate" ]] || { echo "错误: 未找到 JDK 17 keytool" >&2; exit 1; }
  dirname "$(dirname "$candidate")"
}

find_apksigner() {
  local sdk="${ANDROID_HOME:-$HOME/Library/Android/sdk}"
  local bt
  bt="$(ls -1d "$sdk"/build-tools/* 2>/dev/null | sort -V | tail -1 || true)"
  [[ -n "$bt" && -x "$bt/apksigner" ]] || { echo "错误: 未找到 apksigner(build-tools)" >&2; exit 1; }
  echo "$bt/apksigner"
}

init_keystore() {
  [[ -f "$KEYSTORE_FILE" ]] && { echo "已存在: $KEYSTORE_FILE(跳过)"; exit 0; }
  local java_home pass
  java_home="$(find_java_home)"
  read -rsp "keystore 密码(将同时用作 key 密码): " pass; echo
  "$java_home/bin/keytool" -genkeypair -v \
    -keystore "$KEYSTORE_FILE" -alias "$ALIAS" \
    -keyalg RSA -keysize 2048 -validity 10000 \
    -storepass "$pass" -keypass "$pass" \
    -dname "CN=BOSS User App, OU=Mobile, O=ymm BOSS, L=Shenzhen, ST=Guangdong, C=CN"
  cat > "$PROPS_FILE" <<EOF
storeFile=release.keystore
storePassword=$pass
keyAlias=$ALIAS
keyPassword=$pass
EOF
  chmod 600 "$PROPS_FILE"
  echo "已生成 $KEYSTORE_FILE + $PROPS_FILE(均 gitignored)"
}

verify_release() {
  [[ -f "$PROPS_FILE" ]] || { echo "错误: 缺 $PROPS_FILE,先跑 --init" >&2; exit 1; }
  local apksigner java_home
  java_home="$(find_java_home)"
  export JAVA_HOME="$java_home"
  apksigner="$(find_apksigner)"

  echo "==> assembleRelease"
  (cd "$ANDROID_DIR" && ./gradlew assembleRelease)

  local apk="$ANDROID_DIR/app/build/outputs/apk/release/app-release.apk"
  local dbg="$ANDROID_DIR/app/build/outputs/apk/debug/app-debug.apk"
  [[ -f "$apk" ]] || { echo "错误: APK 不存在 $apk" >&2; exit 1; }
  [[ -f "$dbg" ]] || (cd "$ANDROID_DIR" && ./gradlew assembleDebug)

  echo "==> apksigner verify(release)"
  "$apksigner" verify --verbose "$apk"
  # 防 release 误回落 debug 签名不可见:用证书指纹强对比阻断。
  local rel dbg_fp
  rel="$("$apksigner" verify --print-certs "$apk" | grep -o 'SHA-256 digest: .*' | head -1)"
  dbg_fp="$("$apksigner" verify --print-certs "$dbg" | grep -o 'SHA-256 digest: .*' | head -1)"
  echo "release 证书: $rel"
  echo "debug  证书: $dbg_fp"
  if [[ "$rel" == "$dbg_fp" ]]; then
    echo "错误: release 包签名指纹与 debug 相同(回落 debug 签名),发布被阻断" >&2
    exit 1
  fi
  echo "OK: release 独立签名通过,指纹已留档: $rel"
}

[[ $# -eq 1 && "$1" == "--init" ]] && { init_keystore; exit 0; }
verify_release