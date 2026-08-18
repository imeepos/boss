#!/usr/bin/env bash
# BOSS API 接口系统化测试脚本
# 用法: ./api_test.sh [--server URL] [--api-key KEY]
# 输出: 测试结果摘要,失败项明细
set -euo pipefail

SERVER="${BOSS_SERVER:-http://192.168.0.102:28080}"
BOSS="${BOSS:-$(dirname "$0")/../assets/bossctl-darwin-arm64}"
API_KEY="${BOSS_API_KEY:-}"
TEST_DATA_DIR="/tmp/boss-test-$$"
PASS=0
FAIL=0
FAILURES=""

# 认证参数
AUTH=""
if [ -n "$API_KEY" ]; then
  AUTH="--api-key $API_KEY"
fi

ok()   { echo "  PASS: $1"; ((PASS++)); }
fail() { echo "  FAIL: $1"; ((FAIL++)); FAILURES="$FAILURES  - $1\n"; }

# 测试 GET 端点
test_get() {
  local desc="$1" path="$2"
  local code
  code=$(BOSS_SERVER="$SERVER" $BOSS $AUTH call GET "$path" 2>&1 | head -1)
  # 检查是否包含 code=0 或请求成功
  if echo "$code" | grep -qE '"code":0|\{'; then
    ok "$desc ($path)"
  else
    fail "$desc ($path): $code"
  fi
}

# 测试 POST 端点(带 body)
test_post() {
  local desc="$1" path="$2" body="$3"
  local result
  result=$(BOSS_SERVER="$SERVER" $BOSS $AUTH call POST "$path" --data "$body" 2>&1)
  if echo "$result" | grep -qE '"code":0|"id":'; then
    ok "$desc ($path)"
  else
    fail "$desc ($path): $result"
  fi
}

# 测试 404 预期的端点
test_not_found() {
  local desc="$1" path="$2"
  local result
  result=$(BOSS_SERVER="$SERVER" $BOSS $AUTH call GET "$path" 2>&1)
  if echo "$result" | grep -qE "請求失敗|404|not found"; then
    ok "$desc ($path) 返回预期错误"
  else
    fail "$desc ($path): 预期404但得到 $result"
  fi
}

echo "=========================================="
echo "BOSS API 接口测试"
echo "服务器: $SERVER"
echo "时间: $(date)"
echo "=========================================="
echo ""

# ====== 认证模块 ======
echo "[auth] 认证模块"
test_get "获取当前身份" "/auth/me"

# ====== 组织管理 ======
echo "[org] 组织管理"
test_get "子公司列表" "/legal-entities"
test_get "账号列表" "/accounts"
test_get "菜单权限矩阵" "/menu-perms"
test_get "部门列表" "/departments"
test_get "岗位列表" "/posts"
test_get "经营区域" "/regions"
test_get "地址列表" "/addresses"
test_get "地址搜索" "/addresses/search"

# ====== 订单 ======
echo "[order] 订单"
test_get "订单列表" "/orders"

# ====== 派单 ======
echo "[dispatch] 派单"
test_get "任务池" "/pool"
test_get "我的工单" "/my-tickets"
test_get "转单列表" "/transfers"

# ====== 工作台 ======
echo "[dashboard] 工作台"
test_get "运营总览" "/dashboard"

# ====== 计费 ======
echo "[billing] 计费"
test_get "出账查询" "/billing"
test_get "缴费记录" "/payments"
test_get "欠费列表" "/arrears"
test_get "停复机任务" "/stop-resume-tasks"
test_get "对账批次" "/reconciliations"

# ====== 客户 ======
echo "[customer] 客户"
test_get "客户列表" "/customers"
test_get "产品列表" "/products"

# ====== 资源 ======
echo "[resource] 资源"
test_get "资源列表" "/resources"
test_get "端口列表" "/ports"
test_get "预占列表" "/reserves"
test_get "资源转移列表" "/transfers"
test_get "扩容列表" "/expansions"
test_get "QoS 模板" "/expansions/qos-templates"

# ====== 扫码 ======
echo "[scan] 扫码"
test_get "扫码日志" "/scan-logs"
test_get "四码冲突" "/quad-conflicts"

# ====== 资产 ======
echo "[asset] 资产"
test_get "资产台账" "/assets"
test_get "资产批次" "/assets/batches"
test_get "资产分配" "/assets/assignments"
test_get "标签列表" "/tags"
test_get "盘点任务" "/stocktakes"
test_get "换件记录" "/replacements"

# ====== AAA ======
echo "[aaa] 认证计费"
test_get "LO 认证账号" "/lo-accounts"
test_get "话单查询" "/cdrs"
test_get "认证日志" "/auth-logs"

# ====== 设备 ======
echo "[device] 设备"
test_get "告警列表" "/alarms"
test_get "设备指标" "/device/metrics"
test_get "设备维护" "/device/maintenances"

# ====== 地理 ======
echo "[geo] 地理"
test_get "国家列表" "/geo/countries"
test_get "行政区划" "/geo/subdivisions"

# ====== GIS ======
echo "[gis] GIS"
test_get "GIS 下钻" "/gis/drill"
test_get "GIS 层级" "/gis/levels"

# ====== 经营分析 ======
echo "[analytics] 经营分析"
test_get "经营指标" "/analytics/indicators"
test_get "热力图" "/analytics/heatmap"
test_get "维护分析" "/analytics/maintenance"

# ====== 报告 ======
echo "[report] 报告"
test_get "报告列表" "/reports"
test_get "最新报告" "/reports/latest"

# ====== 配置下发 ======
echo "[provision] 配置下发"
test_get "下发模板" "/provision-templates"
test_get "下发任务" "/provision-tasks"
test_get "下发日志" "/provision-logs"

# ====== 四码 ======
echo "[quadlink] 四码"
test_get "四码关联列表" "/quad-links"
test_get "按资产查询" "/quad-links/by-asset"
test_get "按客户查询" "/quad-links/by-customer"
test_get "按端口查询" "/quad-links/by-port"
test_get "按地址查询" "/quad-links/by-address"

# ====== 施工 ======
echo "[worker] 施工"
test_get "班组" "/worker-groups"
test_get "师傅列表" "/workers"
test_get "绩效" "/worker-performances"
test_get "佣金" "/worker-commissions"
test_get "排期" "/worker-schedules"
test_get "材料" "/worker-materials"
test_get "工具" "/worker-tools"
test_get "反馈" "/worker-feedbacks"
test_get "资产回收" "/asset-returns"
test_get "消息" "/worker-messages"
test_get "公告列表" "/notices"

# ====== 新功能模块 ======
echo "[apikey] API key 管理(新功能)"
echo -n "  GET /api-keys: "
curl -s -o /dev/null -w "%{http_code}" -H "Authorization: Bearer $(cat ~/.bossctl/token 2>/dev/null)" "$SERVER/api/v1/api-keys"
echo " (预期 200, 若 404 说明服务器镜像未更新)"

echo ""
echo "=========================================="
echo "测试完成"
echo "通过: $PASS  失败: $FAIL"
if [ $FAIL -gt 0 ]; then
  echo -e "失败明细:\n$FAILURES"
fi
echo "=========================================="