#!/usr/bin/env bash
# T16 机械验收:订单列表「下单时间」列 + 时间轴完成时间缺失降级。
# 用途:静态断言三语列名/页面渲染口径/fields.md 登记 + 聚焦单测,秒级输出可 grep 的 PASS/FAIL。
# 环境: ORDERS_TIME_SKIP_TYPECHECK=1 可跳过 typecheck(已被外层门禁覆盖时)。
set -u
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
ADMIN="$ROOT/web/admin"
fail=0
fail_step() { echo "FAIL $1"; fail=1; }

# 1) 订单列表三语列:状态与操作之间新增下单时间列(顺序固定)
if grep -Fq "columns: ['订单号', '客户', '产品', '地址', '当前环节', '状态', '下单时间', '操作']" "$ADMIN/src/i18n/locales/zh-CN.ts" \
  && grep -Fq "columns: ['Order No', 'Customer', 'Product', 'Address', 'Stage', 'Status', 'Order Time', 'Actions']" "$ADMIN/src/i18n/locales/en-US.ts" \
  && grep -Fq "columns: ['No. Pesanan', 'Pelanggan', 'Produk', 'Alamat', 'Peringkat', 'Status', 'Masa Pesanan', 'Tindakan']" "$ADMIN/src/i18n/locales/ms-MY.ts"; then
  echo "PASS order list createdAt column in 3 locales"
else
  fail_step "order list createdAt column in 3 locales"
fi

# 2) 时间轴缺失降级文案键:types + 三语 locale 同步
if grep -q "timelineUnfinished: string" "$ADMIN/src/i18n/types.ts" \
  && grep -q "timelineUnfinished: '未完成'" "$ADMIN/src/i18n/locales/zh-CN.ts" \
  && grep -q "timelineUnfinished: 'Not finished'" "$ADMIN/src/i18n/locales/en-US.ts" \
  && grep -q "timelineUnfinished: 'Belum selesai'" "$ADMIN/src/i18n/locales/ms-MY.ts"; then
  echo "PASS timelineUnfinished i18n key synced in 3 locales"
else
  fail_step "timelineUnfinished i18n key synced in 3 locales"
fi

# 3) 页面口径:列表 createdAt 走 timeCells(fmtTime),时间轴缺失走 i18n 降级,无裸 UTC/裸 — 渲染
PAGE="$ADMIN/src/pages/boss/order/index.tsx"
if grep -q "createdAtText(r.createdAt)" "$PAGE" && ! grep -Eq '\{r\.createdAt\}' "$PAGE"; then
  echo "PASS order list createdAt rendered via fmtTime helper"
else
  fail_step "order list createdAt rendered via fmtTime helper"
fi
if grep -q "timelineFinishedText(x.finishedAt, o.timelineUnfinished)" "$PAGE" && ! grep -Eq "x\.finishedAt \? fmtTime" "$PAGE"; then
  echo "PASS timeline finishedAt fallback uses i18n text, no bare dash"
else
  fail_step "timeline finishedAt fallback uses i18n text, no bare dash"
fi

# 4) fields.md 契约已登记下单时间列与时间轴降级口径
if grep -q '^| 下单时间 | `CreatedAt` | created_at |' "$ROOT/docs/contract/fields.md" \
  && grep -q '不显示裸 —' "$ROOT/docs/contract/fields.md"; then
  echo "PASS fields.md createdAt column + fallback note registered"
else
  fail_step "fields.md createdAt column + fallback note registered"
fi

# 5) 聚焦单测:时间口径 helper + 三语键集一致性
if (cd "$ADMIN" && CI=true pnpm vitest run src/pages/boss/order/timeCells.test.ts src/i18n/locales/keys.test.ts > /tmp/orders-time-test.log 2>&1) \
  && grep -qE "Test Files[[:space:]]+[0-9]+ passed" /tmp/orders-time-test.log; then
  echo "PASS focused vitest (timeCells + i18n keys parity)"
else
  cat /tmp/orders-time-test.log
  fail_step "focused vitest (timeCells + i18n keys parity)"
fi

# 6) admin typecheck
if [ "${ORDERS_TIME_SKIP_TYPECHECK:-0}" = "1" ]; then
  echo "PASS admin typecheck (covered by prior gate)"
elif (cd "$ADMIN" && CI=true pnpm typecheck > /tmp/orders-time-typecheck.log 2>&1); then
  echo "PASS admin typecheck"
else
  cat /tmp/orders-time-typecheck.log
  fail_step "admin typecheck"
fi

if [ "$fail" -ne 0 ]; then echo "FAIL orders-time-columns-selftest"; exit 1; fi
echo "PASS orders-time-columns-selftest"
