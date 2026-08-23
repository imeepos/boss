package userapi

// 用户端门户 Order 域:handler 实现(trade.go 仅留路由表 + 产品视图工具)。

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
		target := portalProductFindTarget(list, id)
		if target == nil {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"product": portalProductSummary(*target),
			"specs":   productSpecs(*target),
			"compare": portalProductCompare(list, *target),
		})
	}
}

// portalProductFindTarget 详情目标产品(PUBLISHED 才命中)。
func portalProductFindTarget(list []customer.ProductOffer, id int64) *customer.ProductOffer {
	for i := range list {
		if list[i].ID == id && list[i].Status == "PUBLISHED" {
			t := list[i]
			return &t
		}
	}
	return nil
}

// portalProductCompare 同 category 下最多 3 个 PUBLISHED 同类产品(排除自身)。
func portalProductCompare(list []customer.ProductOffer, target customer.ProductOffer) []gin.H {
	others := make([]gin.H, 0, 4)
	for i := range list {
		p := list[i]
		if p.Status != "PUBLISHED" || p.ID == target.ID {
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
	return others
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
		if err := portalOrderUrgeMessage(c, a, cid, o.OrderNo); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// portalOrderUrgeMessage 落催单站内消息(MSG 单号 + 内容)。
func portalOrderUrgeMessage(c *gin.Context, a *app.Application, cid int64, orderNo string) error {
	msgID, err := a.Portal.NextNo(c.Request.Context(), "MSG")
	if err != nil {
		return err
	}
	return a.Portal.PutMessage(c.Request.Context(), cid, gin.H{
		"messageId": msgID, "category": "fault", "title": "催单已受理",
		"content": "订单 " + orderNo + " 已加急处理", "tag": "订单", "tagLevel": "info",
		"createdAt": time.Now(), "read": false,
	})
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
		if o.Status != "DONE" || portalOrderIsRated(a, c, o.OrderNo) {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"orderNo": o.OrderNo, "productName": productName(a, c, o.OfferID),
			"address": addressName(a, c, o.AddressID, ""), "finishedAt": o.CreatedAt,
		})
	}
}

// portalOrderIsRated 该订单是否已评价。
func portalOrderIsRated(a *app.Application, c *gin.Context, orderNo string) bool {
	rated, err := a.Order.RatingExists(c.Request.Context(), orderNo)
	return err == nil && rated
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
