package adminapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// partnerOrderSubmitHandler 渠道下单复用直营 OrderService.Submit，保留 12 环节初始状态。
func partnerOrderSubmitHandler(a *app.Application) gin.HandlerFunc {
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
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		profile, err := a.Partner.Profile(c.Request.Context(), claims.AccountID)
		if err != nil {
			respondErr(c, err)
			return
		}
		req.LegalEntityID = profile.LegalEntityID
		o, err := a.Order.Submit(c.Request.Context(), req)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "partner_order.submit", "order", o.OrderNo, gin.H{
			"channelId": req.ChannelID, "legalEntityId": profile.LegalEntityID,
		})
		respond(c, apitypes.CodeOK, gin.H{"id": o.ID, "orderNo": o.OrderNo, "stage": o.Stage, "status": o.Status})
	}
}
