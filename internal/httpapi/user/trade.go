package userapi

// 用户端门户 Order 域:订单操作/评价 + 产品详情(路由注册驻留此处)。

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerPortalOrderRoutes 客户视角产品/套餐/订单路由入口。
// 各 handler 拆分于 portal_orders.go / portal_plans.go / portal_addons.go。
func registerPortalOrderRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/products", portalListProducts(a))
	g.GET("/products/:id", portalProductDetail(a))
	g.GET("/addons", portalListAddons(a))
	g.POST("/addons/:addonId/subscribe", portalAddonSubscribe(a))
	g.POST("/addons/:addonId/unsubscribe", portalAddonUnsubscribe(a))
	g.GET("/orders", portalListOrders(a))
	g.POST("/orders", portalSubmitOrder(a))
	g.GET("/orders/:orderNo", portalGetOrder(a))
	g.GET("/orders/:orderNo/technician-contact", portalOrderTechnicianContact(a))
	g.POST("/orders/:orderNo/cancel", portalOrderCancel(a))
	g.POST("/orders/:orderNo/urge", portalOrderUrge(a))
	g.POST("/orders/:orderNo/change-address", portalOrderChangeAddr(a))
	g.GET("/orders/:orderNo/rate", portalRateGet(a))
	g.POST("/orders/:orderNo/rate", portalRatePost(a))
	g.GET("/plans/:planId/cancel", portalPlanCancelPreview(a))
	g.POST("/plans/:planId/change", portalPlanChange(a))
	g.POST("/plans/:planId/move", portalPlanMove(a))
	g.POST("/plans/:planId/cancel", portalPlanCancel(a))
}

// portalProductDetail GET /products/:id:产品详情(PUBLISHED 才可见)。
// specs/description/compare 从 ProductOffer 真实字段派生;DB 未建模的字段(合约月数/安装费/设备/适用范围)
// 按 category 规则文案,保证客户端不空。compare 同 category 下取最多 3 个 PUBLISHED 同类。
func portalProductDetail(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		list, lerr := a.Product.ListProducts(c.Request.Context(), 0)
		if lerr != nil {
			respondErr(c, lerr)
			return
		}
		var target *customer.ProductOffer
		for i := range list {
			if list[i].ID == id && list[i].Status == "PUBLISHED" {
				t := list[i]
				target = &t
				break
			}
		}
		if target == nil {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		others := make([]gin.H, 0, 4)
		for i := range list {
			p := list[i]
			if p.Status != "PUBLISHED" || p.ID == id {
				continue
			}
			if p.Category != target.Category {
				continue
			}
			others = append(others, portalProductSummary(p))
			if len(others) >= 3 {
				break
			}
		}
		respond(c, apitypes.CodeOK, gin.H{
			"product": portalProductSummary(*target),
			"specs":   productSpecs(*target),
			"compare": others,
		})
	}
}

// portalProductSummary 产品主信息视图(列表与详情共用),契约 Product 字段集。
func portalProductSummary(p customer.ProductOffer) gin.H {
	return gin.H{
		"productId": strconv.FormatInt(p.ID, 10), "category": p.Category,
		"name": p.Name, "bandwidth": p.Bandwidth, "monthlyFee": p.MonthlyFee,
		"description":    productDescription(p),
		"contractMonths": productContractMonths(p.Category),
		"featured":       productFeatured(p),
	}
}

// productDescription 描述文案:带宽 + 适用人群;DB 无 description 字段时由带宽与分类派生。
func productDescription(p customer.ProductOffer) string {
	bw := p.Bandwidth
	switch p.Category {
	case "broadband":
		return bw + " 高速宽带,适合家庭多设备上网、高清影音"
	case "fusion":
		return bw + " 5G 融合套餐,宽带 + 语音 + 流量一体化"
	default:
		if bw == "" {
			return p.Name
		}
		return bw + " 增值服务"
	}
}

// productContractMonths 合约月数:DB 未建模,按 category 规则约定。
func productContractMonths(category string) int {
	switch category {
	case "broadband":
		return 12
	case "fusion":
		return 24
	default:
		return 1
	}
}

// productFeatured 推荐标记:月费 ≤100 或带宽 ≤500M 视为普通推荐档;高带宽/高价不强制置顶。
func productFeatured(p customer.ProductOffer) bool {
	return p.MonthlyFee > 0 && p.MonthlyFee <= 100
}

// productSpecs 套餐规格行(详情卡片):带宽/合约期/安装费/设备/适用范围。
// DB 未建模的字段(安装费/设备/适用范围)按 category 给出文案占位,前端不改契约即可渲染。
func productSpecs(p customer.ProductOffer) []gin.H {
	bwValue := p.Bandwidth
	if bwValue != "" {
		bwValue += "bps"
	}
	installFee := "¥0"
	device := "光猫免费租用"
	scope := "家庭宽带"
	switch p.Category {
	case "fusion":
		device = "5G CPE 路由器"
		scope = "5G 融合套餐"
	case "addon":
		installFee = "—"
		device = "—"
		scope = "增值服务"
	}
	return []gin.H{
		{"label": "带宽", "value": orDash(bwValue)},
		{"label": "合约期", "value": strconv.Itoa(productContractMonths(p.Category)) + " 个月"},
		{"label": "安装费", "value": installFee},
		{"label": "设备", "value": device},
		{"label": "适用范围", "value": scope},
	}
}

func orDash(s string) string {
	if s == "" {
		return "—"
	}
	return s
}

// portalOwnedOrder 取订单并校验归属(客户视角越权一律 404,不泄露存在性)。
func portalOwnedOrder(a *app.Application, c *gin.Context, cid int64) (*order.Order, bool) {
	o, err := a.Order.GetByNo(c.Request.Context(), c.Param("orderNo"))
	if err != nil || o.CustomerID != cid {
		respond(c, apitypes.CodeNotFound, nil)
		return nil, false
	}
	return o, true
}

// portalOrderCancel POST /orders/:orderNo/cancel:取消订单(释放预占端口)。
func portalOrderCancel(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		o, ok := portalOwnedOrder(a, c, cid)
		if !ok {
			return
		}
		if err := a.Order.Cancel(c.Request.Context(), o.ID); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// portalOrderUrge POST /orders/:orderNo/urge:催单落站内消息(消息表持久化)。
func portalOrderUrge(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		o, ok := portalOwnedOrder(a, c, cid)
		if !ok {
			return
		}
		msgID, err := a.Portal.NextNo(c.Request.Context(), "MSG")
		if err != nil {
			respondErr(c, err)
			return
		}
		if err := a.Portal.PutMessage(c.Request.Context(), cid, gin.H{
			"messageId": msgID, "category": "fault", "title": "催单已受理",
			"content": "订单 " + o.OrderNo + " 已加急处理", "tag": "订单", "tagLevel": "info",
			"createdAt": time.Now(), "read": false,
		}); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// portalOrderChangeAddr POST /orders/:orderNo/change-address:变更安装地址(orders.address_id 落库)。
func portalOrderChangeAddr(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		o, ok := portalOwnedOrder(a, c, cid)
		if !ok {
			return
		}
		var req struct {
			AddressID string `json:"addressId" binding:"required"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		addrID, err := strconv.ParseInt(req.AddressID, 10, 64)
		if err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if err := a.Order.ChangeAddress(c.Request.Context(), o.ID, addrID); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// portalRateGet GET /orders/:orderNo/rate:待评价信息(DONE 且未评价)。
func portalRateGet(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		o, ok := portalOwnedOrder(a, c, cid)
		if !ok {
			return
		}
		if o.Status != "DONE" {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		if rated, err := a.Order.RatingExists(c.Request.Context(), o.OrderNo); err == nil && rated {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"orderNo": o.OrderNo, "productName": productName(a, c, o.OfferID),
			"address": addressName(a, c, o.AddressID, ""), "finishedAt": o.CreatedAt,
		})
	}
}

// portalRatePost POST /orders/:orderNo/rate:提交评价(order_ratings 落库,唯一单号)。
func portalRatePost(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		o, ok := portalOwnedOrder(a, c, cid)
		if !ok {
			return
		}
		var req struct {
			Stars    int    `json:"stars" binding:"required,min=1,max=5"`
			Attitude int    `json:"attitude" binding:"required,min=1,max=5"`
			Quality  int    `json:"quality" binding:"required,min=1,max=5"`
			Comment  string `json:"comment"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		if err := a.Order.SaveRating(c.Request.Context(), order.Rating{
			OrderNo: o.OrderNo, CustomerID: cid,
			Stars: int8(req.Stars), Attitude: int8(req.Attitude),
			Quality: int8(req.Quality), Comment: req.Comment,
		}); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}
