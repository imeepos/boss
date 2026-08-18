package main

import "fmt"

// routeEntry 路由目录项。
type routeEntry struct {
	Method string
	Path   string
	Desc   string
}

// routeCatalog 全部 API 路由目录(来源: internal/app/http_*.go)。
// 前缀统一为 /api/v1。路径参数用 :name 表示,调用时替换为实际值。
var routeCatalog = []routeEntry{
	// auth
	{"POST", "/auth/login", "账号登录,获取 JWT"},
	{"POST", "/auth/register", "自助注册"},
	{"GET", "/auth/me", "当前登录身份"},
	{"POST", "/auth/logout", "退出登录"},
	// org 组织与权限
	{"GET", "/legal-entities", "子公司列表"},
	{"POST", "/legal-entities", "新建子公司"},
	{"PUT", "/legal-entities/:legalEntityId", "更新子公司"},
	{"GET", "/accounts", "账号列表"},
	{"GET", "/menu-perms", "菜单权限矩阵"},
	{"GET", "/departments", "部门列表"},
	{"GET", "/posts", "岗位列表"},
	{"GET", "/regions", "经营区域"},
	{"GET", "/addresses", "地址列表"},
	{"PUT", "/addresses/:id/geo", "地址挂接锚点"},
	{"GET", "/addresses/search", "地址搜索"},
	{"POST", "/addresses", "新建地址"},
	{"PUT", "/addresses/:id", "更新地址名"},
	{"DELETE", "/addresses/:id", "删除地址节点"},
	{"POST", "/addresses/import", "地址批量导入"},
	// apikey
	{"GET", "/api-keys", "API key 列表"},
	{"POST", "/api-keys", "创建 API key"},
	{"DELETE", "/api-keys/:id", "吊销 API key"},
	// ai OpenAI 能力网关
	{"GET", "/ai/openai/config", "OpenAI 集中配置视图(脱敏)"},
	{"PUT", "/ai/openai/config", "变更 OpenAI 集中配置(热更)"},
	{"POST", "/ai/chat/completions", "AI 对话补全"},
	{"POST", "/ai/embeddings", "AI 文本向量化"},
	// order 订单
	{"GET", "/orders", "订单列表"},
	{"POST", "/orders", "新建订单"},
	{"GET", "/orders/:orderNo", "订单详情"},
	// dispatch 派单
	{"GET", "/dispatch/pool", "任务池"},
	{"POST", "/dispatch/pool/:ticketNo/assign", "领取任务"},
	{"GET", "/dispatch/my-tickets", "我的工单"},
	{"GET", "/dispatch/transfers", "转单列表(派单)"},
	{"POST", "/dispatch/tickets/:ticketNo/transfer", "转单"},
	// dashboard 工作台
	{"GET", "/dashboard", "运营总览"},
	// billing 计费
	{"GET", "/bills", "出账查询"},
	{"GET", "/payments", "缴费记录"},
	{"GET", "/arrears", "欠费列表"},
	{"GET", "/stop-resume-tasks", "停复机任务"},
	{"POST", "/stop-resume-tasks/:taskId/retry", "停复机重试"},
	{"GET", "/reconciliations", "对账批次"},
	{"POST", "/reconciliations/:batchNo/settle", "对账结算"},
	// customer 客户
	{"GET", "/customers", "客户列表"},
	{"GET", "/customers/:id/verify-logs", "实名核验记录"},
	{"GET", "/products", "产品列表"},
	{"POST", "/products", "新建产品"},
	{"GET", "/products/:id/price-history", "产品价格历史"},
	// resource 资源
	{"GET", "/resources", "资源列表"},
	{"GET", "/ports", "端口列表"},
	{"GET", "/ports/:portId/change-history", "端口变更历史"},
	{"POST", "/reserves/:reserveId/release", "释放预占"},
	{"GET", "/reserves", "预占列表"},
	{"POST", "/transfers", "资源转移申请"},
	{"POST", "/transfers/:transferNo/approve", "转移审批通过"},
	{"POST", "/transfers/:transferNo/reject", "转移审批驳回"},
	{"GET", "/transfers", "资源转移列表"},
	{"GET", "/expansions", "扩容列表"},
	{"POST", "/expansions", "发起扩容"},
	{"GET", "/expansions/qos-templates", "QoS 模板"},
	// scan 扫码
	{"POST", "/tickets/:ticketNo/scan-bind", "扫码绑定"},
	{"POST", "/tickets/:ticketNo/dismantle/scan", "拆机扫码"},
	{"GET", "/scan-logs", "扫码日志"},
	{"GET", "/quad-conflicts", "四码冲突"},
	{"POST", "/quad-conflicts/:id/resolve", "处理冲突"},
	{"POST", "/quad-links/reconcile", "四码对账"},
	// asset 资产
	{"GET", "/assets", "资产台账"},
	{"GET", "/assets/:assetId/lifecycle", "资产生命周期"},
	{"GET", "/assets/batches", "资产批次"},
	{"GET", "/assets/assignments", "资产分配"},
	{"GET", "/tags", "标签列表"},
	{"POST", "/stocktakes", "发起盘点"},
	{"POST", "/stocktakes/:taskId/diff-handle", "盘差处理"},
	{"POST", "/replacements", "发起换件"},
	{"GET", "/stocktakes", "盘点任务"},
	{"GET", "/replacements", "换件记录"},
	// aaa 认证计费
	{"GET", "/lo-accounts", "LO 认证账号"},
	{"GET", "/cdrs", "话单查询"},
	{"GET", "/auth-logs", "认证日志"},
	// device 设备
	{"GET", "/alarms", "告警列表"},
	{"POST", "/alarms/:alarmId/ack", "告警确认"},
	{"POST", "/alarms/batch-retest", "批量复测"},
	{"GET", "/device/metrics", "设备指标"},
	{"GET", "/device/maintenances", "设备维护"},
	// geo 地理
	{"GET", "/geo/countries", "国家列表"},
	{"GET", "/geo/countries/:code", "国家详情"},
	{"POST", "/geo/countries", "新建国家"},
	{"PUT", "/geo/countries/:code", "更新国家"},
	{"PUT", "/geo/countries/:code/active", "启停国家"},
	{"POST", "/geo/countries/:code/names", "国家多语言名称"},
	{"DELETE", "/geo/countries/:code/names/:locale/:nameType", "删除国家名称"},
	{"PUT", "/geo/countries/:code/attrs", "更新国家属性"},
	{"GET", "/geo/subdivisions", "行政区划列表"},
	{"GET", "/geo/subdivisions/:code/names", "区划多语言名称"},
	{"POST", "/geo/subdivisions", "新建区划"},
	{"PUT", "/geo/subdivisions/:code", "更新区划"},
	{"PUT", "/geo/subdivisions/:code/active", "启停区划"},
	{"POST", "/geo/subdivisions/:code/names", "区划多语言名称"},
	{"DELETE", "/geo/subdivisions/:code/names/:locale/:nameType", "删除区划名称"},
	{"POST", "/geo/import", "地理数据导入"},
	// gis
	{"GET", "/gis/drill", "GIS 下钻"},
	{"GET", "/gis/levels", "GIS 层级"},
	{"GET", "/gis/resources/:resourceId/detail", "资源详情"},
	// analytics 经营分析
	{"GET", "/analytics/indicators", "经营指标"},
	{"GET", "/analytics/heatmap", "热力图"},
	{"GET", "/analytics/maintenance", "维护分析"},
	// report 报告
	{"GET", "/reports", "报告列表"},
	{"GET", "/reports/latest", "最新报告"},
	// provision 配置下发
	{"GET", "/provision-templates", "下发模板"},
	{"GET", "/provision-tasks", "下发任务"},
	{"GET", "/provision-logs", "下发日志"},
	{"POST", "/provision-tasks/:taskNo/retry", "下发重试"},
	// quadlink 四码
	{"GET", "/quad-links", "四码关联"},
	{"GET", "/quad-links/by-asset", "按资产查"},
	{"GET", "/quad-links/by-customer", "按客户查"},
	{"GET", "/quad-links/by-port", "按端口查"},
	{"GET", "/quad-links/by-address", "按地址查"},
	// worker 施工
	{"GET", "/worker-groups", "班组"},
	{"GET", "/workers", "师傅列表"},
	{"GET", "/worker-performances", "绩效"},
	{"GET", "/worker-commissions", "佣金"},
	{"GET", "/worker-schedules", "排期"},
	{"GET", "/worker-materials", "材料"},
	{"GET", "/worker-tools", "工具"},
	{"GET", "/worker-feedbacks", "反馈"},
	{"GET", "/asset-returns", "资产回收"},
	{"GET", "/worker-messages", "消息"},
	{"POST", "/worker-messages", "下发消息"},
	{"PUT", "/workers/:workerId/settings", "师傅设置"},
	{"GET", "/notices", "公告列表"},
	{"POST", "/notices", "发布公告"},
	{"PUT", "/notices/:noticeId/toggle", "公告启停"},
}

// routes 列出 API 路由目录: bossctl routes
func (c *CLI) routes() {
	fmt.Println("BOSS API 路由目录(前缀 /api/v1,路径参数以 :name 表示)")
	fmt.Println("用法: bossctl call METHOD PATH [--data JSON] [--query k=v]")
	fmt.Println("示例: bossctl call GET /orders --query page=1")
	fmt.Println()
	var curSection string
	for _, r := range routeCatalog {
		section := sectionOf(r.Path)
		if section != curSection {
			curSection = section
			fmt.Printf("\n[%s]\n", section)
		}
		fmt.Printf("  %-6s %s    %s\n", r.Method, r.Path, r.Desc)
	}
}

// sectionOf 依据路径推断所属模块(仅用于路由目录分组显示)。
func sectionOf(path string) string {
	switch {
	case len(path) >= 2 && path[:2] == "/a":
		return "auth / apikey / accounts / assets / alarms / analytics"
	case len(path) >= 2 && path[:2] == "/o":
		return "orders"
	case len(path) >= 2 && path[:2] == "/p":
		return "payments / products / pool / ports / provision"
	case len(path) >= 2 && path[:2] == "/r":
		return "resources / regions / reports / reconciliations"
	case len(path) >= 2 && path[:2] == "/t":
		return "transfers / tickets / tags / stocktakes"
	case len(path) >= 2 && path[:2] == "/c":
		return "customers / cdrs"
	case len(path) >= 2 && path[:2] == "/d":
		return "dashboard / departments / device"
	case len(path) >= 2 && path[:2] == "/g":
		return "geo / gis"
	case len(path) >= 2 && path[:2] == "/q":
		return "quad-links"
	case len(path) >= 2 && path[:2] == "/w":
		return "worker"
	case len(path) >= 2 && path[:2] == "/b":
		return "billing"
	case len(path) >= 2 && path[:2] == "/l":
		return "legal-entities / lo-accounts"
	case len(path) >= 2 && path[:2] == "/m":
		return "menu-perms / my-tickets"
	case len(path) >= 2 && path[:2] == "/s":
		return "scan / stop-resume"
	case len(path) >= 2 && path[:2] == "/n":
		return "notices"
	default:
		return "other"
	}
}
