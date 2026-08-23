package adminapi

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

func partnerCommissionListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		items, err := a.PartnerCommission.ListCommissionLedger(c.Request.Context(), claims.AccountID, c.Query("status"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

func partnerCommissionSettleHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		if err := a.PartnerCommission.SettleCommissionLedger(c.Request.Context(), id, claims.AccountID); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "partner_commission.settle", "partner_commission_ledger", strconv.FormatInt(id, 10), gin.H{})
		respond(c, apitypes.CodeOK, nil)
	}
}
