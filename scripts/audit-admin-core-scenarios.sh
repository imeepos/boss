#!/usr/bin/env bash
# audit-admin-core-scenarios.sh -- admin 核心业务场景齐全性机械审计
#
# why grep 静态断言:上线门禁需要"机械可判"的齐全性检查——不起服务、不依赖运行时数据,
# 只对 admin.yaml 聚合路由清单做存在性断言;期望清单每条自带契约依据(terms.md 环节号/
# domain-map.md 分组),红即缺失,零灰区,纯 grep 秒级,可直接进 CI 门禁。
# 期望清单裁定规则:某能力在 admin.yaml 无路由时,先查 domain-map.md 判域归属;
# 确属非 admin 域 -> 从清单移除并留注释引用文档;确属 admin 域但缺失 -> 保留断言让脚本红。
# 本清单 59 条已按 domain-map.md 第 1/2 节逐条核对,全部确属 admin 域,无移除项。

set -u

repo="$(cd "$(dirname "${BASH_SOURCE[0]}")/.." && pwd)"
spec="$repo/api/openapi/admin.yaml"

# admin.yaml 是聚合契约:路由行格式为两空格缩进+/path+冒号+$ref 指向 admin/ 域文件,
# 前缀 /api/admin/v1 由 servers 段统一声明,故行首两空格加斜杠即路由条目
if [ ! -f "$spec" ]; then
  echo "FATAL: 找不到 $spec"
  exit 1
fi

total=$(grep -c '^  /' "$spec" 2>/dev/null) || total=0
# 防作弊自保护:契约文件被清空/截断时路由总数骤降,此时断言红绿均不可信,直接判文件异常
if [ "$total" -lt 100 ]; then
  echo "FATAL: admin.yaml 路由行仅 $total 条(<100),契约文件异常,审计不可信"
  exit 1
fi

# 期望清单:字段用 ~ 分隔 = 类别~能力~grep -E 模式~契约依据;模式匹配聚合路由行,大括号路径参数按 ERE 转义
EXPECT=(
"订单环节~1下单(admin代客下单+订单列表)~^  /orders:~terms.md §1 环节1 执行方=客户/客服,admin 承接客服侧管理面"
"订单环节~2资源核查(触发+预览)~^  /orders/\{orderNo\}/check-(resource|preview):~terms.md §1 环节2 执行方=系统,admin 管理面可手动触发/预览"
"订单环节~3端口预占(触发+查询+释放)~^  /orders/\{orderNo\}/reserve:|^  /reserves~terms.md §1 环节3 执行方=系统,admin 管理面(触发/预占查询/超时释放)"
"订单环节~4合同收费~^  /orders/\{orderNo\}/charge:|^  /payments:~terms.md §1 环节4 执行方=财务/客服,admin 主执行面;未收费不派单(REQ-CL-001)"
"订单环节~5标签预绑定(仅可观测)~^  /tags:~terms.md §1 环节5 执行方=系统,系统自动环节,admin 仅可观测标签台账(domain-map §2 ams/tag)"
"订单环节~6创建账号(仅可观测)~^  /users:|^  /user-accounts:~terms.md §1 环节6 执行方=系统,系统自动环节,admin 仅可观测认证账号视图"
"订单环节~7预下发配置(进度+重试)~^  /provision-tasks~terms.md §1 环节7 执行方=系统,admin 管理面(进度追踪+异常重试 retry)"
"订单环节~8派单(调度指派)~^  /dispatch/pool~terms.md §1 环节8 执行方=调度/系统,admin 管理面(工单池查询+指派 assign)"
"订单环节~9扫码绑定(补录+日志)~^  /scan-logs|/scan-bind:~terms.md §1 环节9 执行方=装维,admin 管理面(补录/重扫 scan-bind)+scan-logs 可观测"
"订单环节~10激活(补激活入口)~^  /tickets/\{ticketNo\}/activate:~terms.md §1 环节10 执行方=装维/系统,admin 管理面补激活入口"
"订单环节~11激活回调(异常重试)~^  /activation-callbacks|^  /callbacks~terms.md §1 环节11 执行方=系统;domain-map §2.1 callback 页=订单第11环节;admin 管理面(查询+失败重试)"
"订单环节~12更新GIS(仅可观测)~^  /gis/~terms.md §1 环节12 执行方=系统,系统自动环节,admin 仅可观测 GIS 图层(domain-map §2 intel/gis)"
"核心域~运营总览~^  /dashboard:~domain-map §2 overview 运营总览 dashboard"
"核心域~权限-后台账号~^  /accounts:~domain-map §2 base(SYS) account"
"核心域~权限-角色~^  /roles:~domain-map §2 org(SYS+BRAND) menuperm 组"
"核心域~权限-权限码~^  /permissions:|^  /menu-perms:~domain-map §2 org menuperm/datascope"
"核心域~客户档案~^  /customers:~domain-map §2 bss(CRM) customer"
"核心域~客户-实名核验~^  /customers/\{id\}/real-name|^  /verifications:~terms.md §4 real_name_status/verifications;domain-map §2 bss"
"核心域~客户-停复机~^  /arrears/\{customerId\}/stop:|^  /stop-resume-tasks:~domain-map §2 billing/stopsrv;terms.md AR 行 停机经 stop_resume_tasks 编排"
"核心域~产品资费~^  /products:~domain-map §2 bss(PROD) product"
"核心域~订单详情查询~^  /orders/\{orderNo\}:~domain-map §2 boss(ORD) order"
"核心域~缴费流水~^  /payments:~domain-map §2 billing(PAY) payment"
"核心域~渠道对账~^  /reconciliations~domain-map §2 billing paycheck 渠道对账"
"核心域~账单~^  /bills:~domain-map §2 billing(BIL) billing"
"核心域~欠费催收~^  /arrears:~domain-map §2.1 arrears 页属 AR 应收信用域,挂 billing 分组"
"核心域~发票~^  /invoices:~domain-map §1 TAX 行(无专用页入 billing);terms.md §4 invoice.status/tax_status"
"核心域~报障工单~^  /complaints:~domain-map §2.1 complaint 页属 CS 客服工单域,挂 boss 分组"
"核心域~拆机~^  /dismantles:~domain-map §2 boss(ORD) dismantle"
"核心域~资产台账~^  /assets:~domain-map §2 ams(AMS) asset"
"核心域~端口~^  /ports~domain-map §2 oss(OSS) resource"
"核心域~设备与维护~^  /device/~domain-map §2 oss(DEV) device"
"核心域~换新单~^  /replacements:~domain-map §2 ams replace;terms.md §4 replacement.status"
"核心域~盘点~^  /stocktakes:~domain-map §2 ams stock;terms.md §4 stocktake.status"
"核心域~告警~^  /alarms:~domain-map §2 alarm(MON) alarm;terms.md §4 alarm.level/status"
"核心域~话单~^  /cdrs:~domain-map §2 aaa(AAA) aaalog;terms.md §4 cdr.billing_status"
"核心域~认证日志~^  /auth-logs:~domain-map §2 aaa(AAA) aaalog;terms.md §4 auth_log.result"
"核心域~认证账号~^  /lo-accounts:~domain-map §2.1 loaccount 页属 AAA 域挂 oss 分组;terms.md §4 lo_account.status"
"核心域~四码合一~^  /quad-links:~domain-map §2 quad(QUAD) quadlink;terms.md §4 quad_link"
"核心域~预配置模板~^  /provision-templates:~domain-map §2 provision(PROV) template"
"核心域~预配置日志~^  /provision-logs:~domain-map §2 provision(PROV) provlog"
"核心域~经营分析报表~^  /analytics/|^  /reports:~domain-map §2 intel(BI) analytics/report"
"核心域~券模板~^  /coupon-templates:~domain-map §1 PROMO 行(模板/发放经 admin API);terms.md §4 template.status"
"核心域~券实例~^  /coupons:~domain-map §1 PROMO 行;terms.md §4 coupon.status/source"
"核心域~券兑换码~^  /coupon-templates/\{templateId\}/codes:~domain-map §1 PROMO 行 兑换码;terms.md §4 code.status"
"核心域~积分忠诚度~^  /points/|^  /loy/~domain-map §1 LOY 行(admin API /points/*)"
"核心域~师傅管理~^  /workers:~domain-map §2 boss worker/worker-ops;terms.md §4 workers.status"
"核心域~师傅入驻审核~^  /worker-registrations:~terms.md §4 worker_registrations.status"
"核心域~后台消息通知~^  /notifications:~domain-map §1 NOT 行(admin 侧 notify 提醒/待办) + §2 boss message 页"
"核心域~经营区域~^  /regions:~domain-map §2 org(BRAND) region"
"核心域~法人主体~^  /legal-entities:~domain-map §2 org company"
"核心域~免登录APIkey~^  /api-keys:~domain-map §1 apikey 行(org 组,绑定 account/worker/customer 三主体)"
"核心域~开放平台~^  /openplat/apps:~domain-map §1 OPEN 行(org 开发者门户 /openplat)"
"核心域~渠道入驻审核~^  /partner/applications:~domain-map §1 CH 行(org 审核页 partner)"
"核心域~ODN局点~^  /odn/sites:~domain-map §1 ODN 行(oss 组 menu:odn)"
"核心域~官网内容~^  /site-posts:~domain-map §1 CMS 行(boss 组 /boss/site)"
"核心域~客户端发版~^  /client-releases:~domain-map §1 APPREL 行(boss 组 /boss/release)"
"核心域~国际地理数据~^  /geo/countries:~domain-map §1 geo 行(base 组)"
"核心域~数据备份~^  /backup/jobs:~domain-map §2 base(backup 迁移 000095)"
"核心域~审计日志~^  /audit-logs:~domain-map §2 base(SYS) audit"
)
pass=0
fail=0
fails=()
for entry in "${EXPECT[@]}"; do
  IFS='~' read -r cls name pat basis <<< "$entry"
  [ -n "$cls" ] || continue
  n=$(grep -Ec "$pat" "$spec" 2>/dev/null) || n=0
  if [ "$n" -ge 1 ]; then
    printf 'PASS  %-8s %-28s 命中%d  %s\n' "$cls" "$name" "$n" "$basis"
    pass=$((pass+1))
  else
    printf 'FAIL  %-8s %-28s 命中0  %s\n' "$cls" "$name" "$basis"
    fail=$((fail+1)); fails+=("$name")
  fi
done
echo ""
echo "================ 审计汇总 ================"
echo "契约文件: $spec"
echo "路由总数: $total"
echo "期望条目: $((pass+fail))  PASS: $pass  FAIL: $fail"
if [ "$fail" -gt 0 ]; then
  echo "----------- FAIL 明细(确属 admin 域但路由缺失) -----------"
  for f in "${fails[@]}"; do echo "  MISSING: $f"; done
  echo "裁定提示: 先查 domain-map.md 判域归属;非 admin 域移除断言留注释,admin 域缺失即上线阻塞"
  exit 1
fi
echo "全部核心业务场景在 admin 契约中齐备"
exit 0
