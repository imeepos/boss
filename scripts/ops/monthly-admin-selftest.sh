#!/usr/bin/env bash
# T20 月度填报 admin 页自检(唯一验收判定,退出码 0=过):
#   1. web/admin typecheck(+可选 build)
#   2. 菜单 key monthly 注册断言(menu.def / App.tsx / i18n 三语言 / 图标)
#   3. 页面静态断言:三页签、派生列只读、CSV 导入导出、KPI null 防护、禁原生 select
#   4. cdp 采集 102 线上 /intel/monthly 真实渲染截图(light/dark,路径见输出)
# 用法: scripts/ops/monthly-admin-selftest.sh [--skip-build] [--skip-shot]
set -e

ROOT=$(cd "$(dirname $0)/../.." && pwd)
WEB=$ROOT/web/admin
PAGE_DIR=$ROOT/web/admin/src/pages/intel/monthly
CAPTURE=$ROOT/.agents/skills/self-evolving/scripts/cdp-admin-capture.mjs
BASE102="http://192.168.0.102:5180"
SHOT_DIR=/tmp
if printenv SHOT_DIR_OVERRIDE >/dev/null 2>&1; then SHOT_DIR=$SHOT_DIR_OVERRIDE; fi
SKIP_BUILD=$1
SKIP_SHOT=$2
PASS=0
FAIL=0

ok()  { echo "PASS: $1"; PASS=$((PASS+1)); }
bad() { echo "FAIL: $1"; FAIL=$((FAIL+1)); }
expect_grep() { # expect_grep <说明> <文件> <pattern>
  if grep -qE "$3" "$2" 2>/dev/null; then ok "$1"; else bad "$1 (缺 $3 于 $2)"; fi
}

echo "== 1. 中央登记 =="
expect_grep "menu.def 注册 key monthly" "$ROOT/web/admin/src/router/menu.def.ts" "key: .monthly."
expect_grep "App.tsx 挂载 MonthlyPage" "$ROOT/web/admin/src/App.tsx" "pageKey === .monthly."
expect_grep "zh-CN 菜单文案" "$ROOT/web/admin/src/i18n/locales/zh-CN.ts" "monthly: .月度填报."
expect_grep "en-US 菜单文案" "$ROOT/web/admin/src/i18n/locales/en-US.ts" "monthly: .Monthly Reporting."
expect_grep "ms-MY 菜单文案" "$ROOT/web/admin/src/i18n/locales/ms-MY.ts" "monthly: .Laporan Bulanan."
if [ -s "$ROOT/web/admin/public/icons/items/monthly.svg" ]; then ok "菜单图标 monthly.svg 存在"; else bad "菜单图标 monthly.svg 缺失"; fi

echo "== 2. 页面组件静态断言 =="
expect_grep "三页签(tabs 数组)" "$PAGE_DIR/index.tsx" "m\.tabs\["
expect_grep "派生列只读渲染" "$PAGE_DIR/DataTable.tsx" "f\.derived"
expect_grep "派生列元数据声明" "$PAGE_DIR/types.ts" "derived: true"
expect_grep "编辑表单排除派生列" "$PAGE_DIR/EditDrawer.tsx" "!f\.derived"
expect_grep "导入行级失败码 42200" "$PAGE_DIR/api.ts" "42200"
expect_grep "导入结果保留成功计数" "$PAGE_DIR/ImportExport.tsx" "importPartial"
expect_grep "行级错误按行号+原因展示" "$PAGE_DIR/ImportExport.tsx" "errorLine"
expect_grep "导出按 Content-Disposition 落盘" "$PAGE_DIR/api.ts" "Content-Disposition"
expect_grep "KPI 比率先判 null" "$PAGE_DIR/KpiCards.tsx" "v === null"
expect_grep "KPI null 显示占位符" "$PAGE_DIR/KpiCards.tsx" ".—"
expect_grep "区域筛选走 Dropdown 组件" "$PAGE_DIR/index.tsx" "<Dropdown"
if grep -qE "<select" "$PAGE_DIR"/*.tsx; then bad "页面出现原生 select(禁用)"; else ok "无原生 select"; fi
expect_grep "分页缺省 pageSize 20" "$PAGE_DIR/index.tsx" "useQueryInt\(.pageSize., 20\)"

echo "== 3. i18n monthlyPage 键(三语言)=="
for f in zh-CN en-US ms-MY; do
  expect_grep "$f monthlyPage 块" "$ROOT/web/admin/src/i18n/locales/$f.ts" "monthlyPage: ."
done

echo "== 4. typecheck/build =="
# CI=true:pnpm verify-deps 的 TTY 确认在无终端环境必挂(ER_TTY),与门禁无关。
if [ "$SKIP_BUILD" != "--skip-build" ]; then
  if (cd "$WEB" && CI=true pnpm typecheck >/dev/null 2>&1); then ok "pnpm typecheck"; else bad "pnpm typecheck"; fi
  if (cd "$WEB" && CI=true pnpm build >/dev/null 2>&1); then ok "pnpm build"; else bad "pnpm build"; fi
else
  echo "(--skip-build 跳过)"
fi

echo "== 5. 102 线上真实渲染(cdp)=="
if [ "$SKIP_SHOT" != "--skip-shot" ]; then
  shot_light="$SHOT_DIR/t20-monthly-light.png"
  shot_dark="$SHOT_DIR/t20-monthly-dark.png"
  assert_js="(() => { const t=document.body.innerText; if(t.indexOf('月度填报')<0&&t.indexOf('Monthly Reporting')<0) return 'PAGE_MISSING'; return t.indexOf('404')<0?'PAGE_OK':'PAGE_404' })()"
  node "$CAPTURE" "$shot_light" --base "$BASE102" --path /intel/monthly --theme light --lang zh-CN --eval "$assert_js" --settle 3500 > /tmp/t20-cdp-light.log 2>&1 || true
  node "$CAPTURE" "$shot_dark" --base "$BASE102" --path /intel/monthly --theme dark --lang zh-CN --eval "$assert_js" --settle 3500 > /tmp/t20-cdp-dark.log 2>&1 || true
  if grep -q "PAGE_OK" /tmp/t20-cdp-light.log; then ok "102 light 真实渲染(截图 $shot_light)"; else bad "102 light 渲染断言失败(见 /tmp/t20-cdp-light.log)"; fi
  if grep -q "PAGE_OK" /tmp/t20-cdp-dark.log; then ok "102 dark 真实渲染(截图 $shot_dark)"; else bad "102 dark 渲染断言失败(见 /tmp/t20-cdp-dark.log)"; fi
  if [ -s "$shot_light" ] && [ -s "$shot_dark" ]; then ok "截图留档: $shot_light + $shot_dark"; else bad "截图文件缺失"; fi
else
  echo "(--skip-shot 跳过)"
fi

echo ""
echo "== 结果: PASS=$PASS FAIL=$FAIL =="
[ "$FAIL" -eq 0 ]
