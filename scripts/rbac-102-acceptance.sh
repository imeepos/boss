#!/usr/bin/env bash
# RBAC 五项修复 102 真实环境验收(2026-08-28)
# req 契约: req METHOD PATH [TOKEN] [BODY] —— token 恒为 $3,body 恒为 $4
set -u
BASE=http://192.168.0.102:28080/api/admin/v1
PW='Rbac2Test!2026x'
PASS=0; FAIL=0
ok(){ echo "PASS: $1"; PASS=$((PASS+1)); }
bad(){ echo "FAIL: $1"; FAIL=$((FAIL+1)); }

req(){ curl -s -X "$1" "$BASE$2" -H 'Content-Type: application/json' ${3:+-H "Authorization: Bearer $3"} ${4:+-d "$4"}; }
tok(){ req POST /auth/login '' "{\"username\":\"$1\",\"password\":\"$2\"}" | python3 -c 'import sys,json;d=json.load(sys.stdin);print(d.get("data",{}).get("token","") if d.get("code")==0 else "")' 2>/dev/null; }
jcode(){ python3 -c 'import sys,json;d=json.load(sys.stdin);print(d.get("code"))' 2>/dev/null; }

echo "===== 0. admin 登录 ====="
AT=$(tok admin admin123)
[ -n "$AT" ] && ok "admin 登录" || { bad "admin 登录"; exit 1; }

echo "===== 0.5 前置清扫(上轮残留) ====="
ssh imeepos@192.168.0.102 "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -c \"DELETE FROM customers WHERE name LIKE 'e2e-rbac2-%'\" -c \"DELETE FROM accounts WHERE username LIKE 'e2e-rbac2-%'\"" | tail -2

echo "===== 1. 造号(API) ====="
mk(){ req POST /accounts "$AT" "{\"username\":\"$1\",\"password\":\"$2\",\"realName\":\"$3\",\"roleCode\":\"$4\"${5:+,\"legalEntityId\":$5}${6:+,\"regionScope\":\"$6\"}}" | python3 -c 'import sys,json;print(json.load(sys.stdin).get("data",{}).get("id",""))'; }
OPS_ID=$(mk e2e-rbac2-ops "$PW" 'e2e ops' ops 1 root.luzon)
TECH_ID=$(mk e2e-rbac2-tech "$PW" 'e2e tech' technician "" "")
GONE_ID=$(mk e2e-rbac2-gone "$PW" 'e2e gone' ops "" "")
PT_ID=$(mk e2e-rbac2-pt "$PW" 'e2e pt' partner_admin 7 "")
echo "ids: ops=$OPS_ID tech=$TECH_ID gone=$GONE_ID pt=$PT_ID"
[ -n "$OPS_ID" ] && [ -n "$TECH_ID" ] && [ -n "$GONE_ID" ] && [ -n "$PT_ID" ] && ok "4 个测试号创建" || bad "测试号创建"

echo "===== 2. 探针客户(root.visayas.cebu.city, 区域外) ====="
PROBE=$(req POST /customers "$AT" '{"name":"e2e-rbac2-probe-cebu","phone":"13900000001","regionId":8,"legalEntityId":6,"addressId":288,"idNo":"E2E88888888","idType":"护照"}' | python3 -c 'import sys,json;print(json.load(sys.stdin).get("data",{}).get("id",""))')
echo "probe customer id=$PROBE"
[ -n "$PROBE" ] && ok "探针客户创建" || bad "探针客户创建"

echo "===== 3. 各号登录 ====="
OPS=$(tok e2e-rbac2-ops "$PW"); TECH=$(tok e2e-rbac2-tech "$PW"); GONE=$(tok e2e-rbac2-gone "$PW"); PT=$(tok e2e-rbac2-pt "$PW")
[ -n "$OPS" ] && [ -n "$TECH" ] && [ -n "$GONE" ] && [ -n "$PT" ] && ok "4 号登录" || bad "4 号登录"

echo "===== P1 客户正控: ops(region_scope=root.luzon) 读范围内客户应 200 ====="
LUZON_ID=$(ssh imeepos@192.168.0.102 "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -tAc \"SELECT c.id FROM customers c JOIN regions r ON r.id=c.region_id WHERE r.path::text LIKE 'root.luzon%' ORDER BY c.id LIMIT 1\"")
C_IN=$(req GET "/customers/$LUZON_ID" "$OPS")
echo "  luzon id=$LUZON_ID code=$(echo "$C_IN" | jcode)"
[ "$(echo "$C_IN" | jcode)" = "0" ] && ok "P1 范围内客户可达" || bad "P1 范围内客户可达"

echo "===== P2 客户越权 vs 不存在: 响应必须逐字节一致 ====="
C_OUT=$(req GET "/customers/$PROBE" "$OPS")
C_MISS=$(req GET "/customers/999999" "$OPS")
echo "  out: $(echo "$C_OUT" | head -c 120)"
echo "  miss: $(echo "$C_MISS" | head -c 120)"
[ "$C_OUT" = "$C_MISS" ] && ok "P2 客户越权与不存在不可区分" || bad "P2 客户越权与不存在可区分!"

echo "===== P3 订单越权 vs 不存在: ORD-20260828-000575(le=6,root) ====="
O_OUT=$(req GET "/orders/ORD-20260828-000575" "$OPS")
O_MISS=$(req GET "/orders/NOPE-404" "$OPS")
echo "  out: $(echo "$O_OUT" | head -c 120)"
echo "  miss: $(echo "$O_MISS" | head -c 120)"
[ "$O_OUT" = "$O_MISS" ] && ok "P3 订单越权与不存在不可区分" || bad "P3 订单越权与不存在可区分!"

echo "===== P4 越权 cancel 应被拦且数据不变 ====="
OC=$(req POST "/orders/ORD-20260828-000575/cancel" "$OPS")
echo "  cancel resp: $(echo "$OC" | head -c 120)"
[ "$OC" = "$O_MISS" ] && ok "P4 越权 cancel 拦截同族" || bad "P4 越权 cancel 异常"
ST=$(ssh imeepos@192.168.0.102 "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -tAc \"SELECT status FROM orders WHERE order_no='ORD-20260828-000575'\"")
[ "$ST" = "CANCELLED" ] && ok "P4 订单状态未被触碰" || bad "P4 订单状态被改! $ST"

echo "===== P5 停用账号旧 token 应 401 ====="
ME_BEFORE=$(req GET /auth/me "$GONE" | jcode)
req PUT "/accounts/$GONE_ID" "$AT" "{\"username\":\"e2e-rbac2-gone\",\"realName\":\"e2e gone\",\"roleCode\":\"ops\",\"status\":0}" > /dev/null
ME_AFTER=$(req GET /auth/me "$GONE" | jcode)
RELOGIN=$(tok e2e-rbac2-gone "$PW")
echo "  停用前 me code=$ME_BEFORE / 停用后=$ME_AFTER / 重登录 token len=${#RELOGIN}"
[ "$ME_BEFORE" = "0" ] && [ "$ME_AFTER" = "401" ] && [ -z "$RELOGIN" ] && ok "P5 停用即拒旧 token+拒新登录" || bad "P5 停用后 me=$ME_AFTER relogin=${#RELOGIN}"

echo "===== P6 partner region-scope NULL公共区/清空/外资区 ====="
R1=$(req PUT /partner/region-scope "$PT" '{"regionPath":"root.visayas"}'); echo "  NULL公共区 visayas: $(echo "$R1" | head -c 100)"
R2=$(req PUT /partner/region-scope "$PT" '{"regionPath":""}'); echo "  清空:               $(echo "$R2" | head -c 100)"
R3=$(req PUT /partner/region-scope "$PT" '{"regionPath":"root"}'); echo "  外资区 root(企业6): $(echo "$R3" | head -c 100)"
[ "$(echo "$R1" | jcode)" = "0" ] && ok "P6a NULL公共区域设置成功(原50000)" || bad "P6a=$(echo "$R1"|jcode)"
[ "$(echo "$R2" | jcode)" = "0" ] && ok "P6b 清空 scope 成功(原坏路径)" || bad "P6b clear=$(echo "$R2"|jcode)"
[ "$(echo "$R3" | jcode)" = "40300" ] && ok "P6c 外资区域 40300 业务码(原500)" || bad "P6c=$(echo "$R3"|jcode)"

echo "===== P7 install-logs 收权 ====="
IL_T=$(req GET "/install-logs?ticketId=221" "$TECH" | jcode)
IL_O=$(req GET "/install-logs?ticketId=221" "$OPS" | jcode)
IL_A=$(req GET "/install-logs?ticketId=221" "$AT" | jcode)
echo "  tech=$IL_T ops=$IL_O admin=$IL_A"
[ "$IL_T" = "403" ] && [ "$IL_O" = "0" ] && [ "$IL_A" = "0" ] && ok "P7 tech 403 拒/ops 通/admin 通" || bad "P7 tech=$IL_T ops=$IL_O admin=$IL_A"

echo "===== 清理 ====="
for id in $OPS_ID $TECH_ID $GONE_ID $PT_ID; do
  req PUT "/accounts/$id" "$AT" '{"status":0,"username":"x","realName":"x","roleCode":"ops"}' > /dev/null
  DEL=$(req DELETE "/accounts/$id" "$AT" | jcode)
  echo "  account $id: disable+delete code=$DEL"
done
ssh imeepos@192.168.0.102 "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -c \"DELETE FROM customers WHERE name LIKE 'e2e-rbac2-%'\" -c \"DELETE FROM accounts WHERE username LIKE 'e2e-rbac2-%'\"" | tail -2
LEFT=$(ssh imeepos@192.168.0.102 "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -tAc \"SELECT count(*) FROM accounts WHERE username LIKE 'e2e-rbac2-%'\"")
LEFT2=$(ssh imeepos@192.168.0.102 "docker exec -i boss-infra-postgres-1 psql -U boss -d boss -tAc \"SELECT count(*) FROM customers WHERE name LIKE 'e2e-rbac2-%'\"")
[ "$LEFT" = "0" ] && [ "$LEFT2" = "0" ] && ok "清理复核 0 残留" || bad "残留 acc=$LEFT cust=$LEFT2"

echo "===== V2 组: 工单寻址端点数据范围守卫(fix7, 2026-08-28 二轮) ====="
# 前置: 造 rs=root.luzon 的 ops 号 + 无区域工单 TK-TEST-L1 + luzon 子树工单 TK-20260822-374
RSOPS_ID=$(req POST /accounts "$AT" "{\"username\":\"e2e-rbac2-rsops\",\"password\":\"$PW\",\"realName\":\"e2e rsops\",\"roleCode\":\"ops\",\"regionScope\":\"root.luzon\"}" | python3 -c 'import sys,json;print(json.load(sys.stdin).get("data",{}).get("id",""))')
RTOK=$(tok e2e-rbac2-rsops "$PW")
OT1=$(req POST "/tickets/TK-TEST-L1/activate" "$RTOK" | jcode)
OT2=$(req POST "/tickets/TK-TEST-L1/scan-bind" "$RTOK" '{"epc":"X"}' | jcode)
OT3=$(req POST "/dispatch/tickets/TK-TEST-L1/transfer" "$RTOK" '{"toMasterId":5,"reason":"e2e"}' | jcode)
[ "$OT1" = "40400" ] && [ "$OT2" = "40400" ] && [ "$OT3" = "40400" ] && ok "V2 区域受限 ops 对无区域工单三端点全 40400" || bad "V2 activate=$OT1 scan=$OT2 transfer=$OT3"
req PUT "/accounts/$RSOPS_ID" "$AT" '{"status":0,"username":"x","realName":"x","roleCode":"ops"}' > /dev/null
req DELETE "/accounts/$RSOPS_ID" "$AT" > /dev/null

echo "===== 结果: PASS=$PASS FAIL=$FAIL ====="
[ "$FAIL" = "0" ]
