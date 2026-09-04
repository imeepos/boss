#!/usr/bin/env bash
# T19 月度填报后端事实域 机械自测(秒级;devloop_accept 唯一验收判定)。
# 覆盖:迁移文件存在 / go build / go vet 月度包 / go test 域单测(BOM 解析、幂等 upsert、
# 派生列、非法拒绝、模板字节级导出回读)/ admin 路由注册断言 / 契约门禁路由对账。
set -u
ROOT="$(cd "$(dirname "$0")/../.." && pwd)"
cd "$ROOT"
fail=0
fail_step() { echo "FAIL $1"; fail=1; }

# 1 迁移文件存在性(000181 up/down 成对)。
if [ -f migrations/000181_monthly_report.up.sql ] && [ -f migrations/000181_monthly_report.down.sql ]; then
  echo "PASS migration 000181 up/down files exist"
else
  fail_step "migration 000181 up/down files exist"
fi

# 1a 迁移内容三事实表+白名单+权限(机械 grep)。
if grep -q 'CREATE TABLE monthly_user_revenue' migrations/000181_monthly_report.up.sql \
   && grep -q 'CREATE TABLE monthly_network_delivery' migrations/000181_monthly_report.up.sql \
   && grep -q 'CREATE TABLE monthly_finance_cost' migrations/000181_monthly_report.up.sql \
   && grep -q 'CREATE TABLE monthly_regions' migrations/000181_monthly_report.up.sql \
   && grep -q "'menu:monthly'" migrations/000181_monthly_report.up.sql \
   && grep -q 'GENERATED ALWAYS AS' migrations/000181_monthly_report.up.sql; then
  echo "PASS migration content: 3 fact tables + region whitelist + menu perm + generated derived cols"
else
  fail_step "migration content"
fi

# 1b 白名单种子 51 行(权威输入 _RegionList;INSERT 块内逐 token 计数)。
seed_rows=$(awk '/INSERT INTO monthly_regions/,/Tikay\);/' migrations/000181_monthly_report.up.sql | grep -oE "\('[^']*'\)" | wc -l | tr -d ' ')
if [ "$seed_rows" -eq 51 ]; then
  echo "PASS region whitelist seeds = 51"
else
  fail_step "region whitelist seeds = 51 (got $seed_rows)"
fi

# 2 全仓构建。
if go build ./... > /tmp/monthly-build.log 2>&1; then
  echo "PASS go build ./..."
else
  cat /tmp/monthly-build.log
  fail_step "go build ./..."
fi

# 3 vet 月度包 + admin 月度文件。
if go vet ./internal/domain/monthly/ ./internal/httpapi/admin/ ./internal/app/ > /tmp/monthly-vet.log 2>&1; then
  echo "PASS go vet monthly packages"
else
  cat /tmp/monthly-vet.log
  fail_step "go vet monthly packages"
fi

# 4 域单测:必须包含五类关键用例(存在性断言 + 实际执行)。
test_file=internal/domain/monthly/pg_test.go
pure_file=internal/domain/monthly/monthly_test.go
ok_names=1
grep -q 'func TestParseCSV_BOMAndRows' "$pure_file" || ok_names=0
grep -q 'func TestImportCSV_Idempotent' "$test_file" || ok_names=0
grep -q 'func TestDerivedColumns' "$pure_file" || ok_names=0
grep -q 'func TestImportCSV_RowErrors' "$test_file" || ok_names=0
grep -q 'func TestUpsertWhitelistReject' "$test_file" || ok_names=0
grep -q 'func TestExportCSV' "$test_file" || ok_names=0
grep -q 'func TestExportMatchesTemplateBytes' "$pure_file" || ok_names=0
if [ "$ok_names" -eq 1 ]; then
  echo "PASS required test cases present (BOM parse / idempotent upsert / derived / invalid reject / export roundtrip)"
else
  fail_step "required test cases present"
fi
if go test ./internal/domain/monthly/... > /tmp/monthly-test.log 2>&1; then
  echo "PASS go test ./internal/domain/monthly/..."
else
  cat /tmp/monthly-test.log
  fail_step "go test ./internal/domain/monthly/..."
fi

# 5 admin 路由注册断言(14 条:12 表级 + regions + summary)。
if go test ./internal/httpapi/admin/ -run TestMonthlyRoutes -count=1 > /tmp/monthly-routes.log 2>&1; then
  echo "PASS admin monthly routes registered (14)"
else
  cat /tmp/monthly-routes.log
  fail_step "admin monthly routes registered (14)"
fi

# 6 权限码登记(menu:monthly 进迁移,menuperm 门禁口径)。
if grep -q "menu:monthly" internal/httpapi/admin/monthly.go && grep -q "'menu:monthly'" migrations/000181_monthly_report.up.sql; then
  echo "PASS perm code menu:monthly wired (routes + migration)"
else
  fail_step "perm code menu:monthly wired"
fi

if [ "$fail" -ne 0 ]; then echo "FAIL monthly-report-selftest"; exit 1; fi
echo "PASS monthly-report-selftest"
