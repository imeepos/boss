package adminapi

// 订单域路由 handler 实现(承接 registerOrderRoutes)。

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// orderListHandler GET /orders:订单列表。
func orderListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope, err := a.User.GetDataScope(c.Request.Context(), httpx.ClaimsAccountID(c))
		if err != nil {
			respondErr(c, err)
			return
		}
		list, err := a.Order.List(c.Request.Context(), order.OrderQuery{
			Keyword:       c.Query("keyword"),
			Status:        c.Query("status"),
			LegalEntityID: scope.LegalEntityID,
			RegionScope:   scope.RegionScope,
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		items := make([]orderListResp, 0, len(list))
		for _, it := range list {
			items = append(items, orderListResp{
				ID: it.ID, OrderNo: it.OrderNo, Customer: it.Customer, Product: it.Product, Address: it.Address,
				Stage: it.Stage, StageLabel: stageNames[it.Stage], Status: it.Status,
				Ops: orderOps(it.Status), CreatedAt: it.CreatedAt,
			})
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

// orderSubmitHandler POST /orders:管理员代客下单。
func orderSubmitHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req order.SubmitReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequirePositiveID(req.CustomerID, "customerId"),
				httpx.RequirePositiveID(req.OfferID, "offerId"),
				httpx.RequirePositiveID(req.AddressID, "addressId"),
				httpx.RequirePositiveID(req.ChannelID, "channelId"),
			)
		}) {
			return
		}
		o, err := a.Order.Submit(c.Request.Context(), req)
		if err != nil {
			if errors.Is(err, order.ErrDirectPhoneCap) || errors.Is(err, order.ErrDirectAddressCap) {
				httpx.RecordAudit(a, c, "order.risk.blocked", "customer",
					strconv.FormatInt(req.CustomerID, 10), map[string]any{"reason": err.Error()})
			}
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "order", o.OrderNo, map[string]any{
			"customerId": req.CustomerID, "offerId": req.OfferID, "channelId": req.ChannelID,
		})
		respond(c, apitypes.CodeOK, gin.H{"id": o.ID, "orderNo": o.OrderNo, "stage": o.Stage, "status": o.Status})
	}
}

// orderGetHandler GET /orders/{orderNo}:订单详情(含时间轴)。
func orderGetHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		orderNo := c.Param("orderNo")
		if orderNo == "" {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "orderNo is required"})
			return
		}
		o, err := a.Order.GetByNo(c.Request.Context(), orderNo)
		if err != nil {
			respondErr(c, err)
			return
		}
		_, logs, err := a.Order.Track(c.Request.Context(), o.ID)
		if err != nil {
			respondErr(c, err)
			return
		}
		var location any
		if a.WorkerLocation != nil {
			location, err = a.WorkerLocation.LatestLocationForOrder(c.Request.Context(), o.ID)
			if err != nil {
				respondErr(c, err)
				return
			}
		}
		respond(c, apitypes.CodeOK, gin.H{
			"order":          o,
			"timeline":       buildTimeline(o, logs),
			"latestLocation": location,
		})
	}
}
