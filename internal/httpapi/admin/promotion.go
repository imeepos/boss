package adminapi

// 营销促销域路由:券模板 CRUD/批量发放/兑换码批次(docs/design/promotion-coupon.md)。

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/promotion"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerPromotionRoutes 注册促销域路由(权限挂 menu:userdata 组,与老 /coupons 同域)。
func registerPromotionRoutes(g *gin.RouterGroup, a *app.Application) {
	perm := requirePerm(a.User, "menu:userdata")
	g.GET("/coupon-templates", perm, promoListTemplates(a))
	g.POST("/coupon-templates", perm, promoCreateTemplate(a))
	g.PUT("/coupon-templates/:templateId/disable", perm, promoDisableTemplate(a))
	g.POST("/coupon-templates/:templateId/issue", perm, promoIssue(a))
	g.POST("/coupon-templates/:templateId/codes", perm, promoCreateCodes(a))
	g.GET("/coupon-templates/:templateId/codes", perm, promoListCodes(a))
	g.GET("/gift-rules", perm, promoListGiftRules(a))
	g.POST("/gift-rules", perm, promoCreateGiftRule(a))
	g.PUT("/gift-rules/:ruleId/disable", perm, promoDisableGiftRule(a))
	g.GET("/coupon-recon", perm, promoCouponRecon(a))
}

// couponReconer Promotion 的可选对账能力(PGStore 实现,窄口断言不污染接口)。
type couponReconer interface {
	CouponRecon(ctx context.Context) ([]promotion.CouponReconRow, promotion.CouponReconSummary, error)
}

// promoCouponRecon GET /coupon-recon:券对账报表(diff=drift 只看差异行,缺省全量)。
func promoCouponRecon(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		rc, ok := a.Promotion.(couponReconer)
		if !ok {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		rows, sum, err := rc.CouponRecon(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		if c.Query("diff") == "drift" {
			rows = filterCouponDrift(rows)
		}
		respond(c, apitypes.CodeOK, gin.H{"rows": rows, "summary": sum})
	}
}

// filterCouponDrift 只保留非 MATCH 行。
func filterCouponDrift(rows []promotion.CouponReconRow) []promotion.CouponReconRow {
	out := make([]promotion.CouponReconRow, 0, len(rows))
	for _, r := range rows {
		if r.DiffKind != promotion.ReconDiffMatch {
			out = append(out, r)
		}
	}
	return out
}

// promoListTemplates GET /coupon-templates:模板列表。
func promoListTemplates(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Promotion.ListTemplates(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// promoCreateTemplate POST /coupon-templates:新建模板。
func promoCreateTemplate(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var t promotion.Template
		if !httpx.BindBody(c, &t) {
			return
		}
		id, err := a.Promotion.CreateTemplate(c.Request.Context(), t)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"templateId": id})
	}
}

// promoDisableTemplate PUT /coupon-templates/:templateId/disable:停用模板。
func promoDisableTemplate(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := pathIDValid(c, "templateId")
		if !ok {
			return
		}
		if err := a.Promotion.DisableTemplate(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// promoIssue POST /coupon-templates/:templateId/issue {customerIds}:批量发放。
func promoIssue(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			CustomerIDs []int64 `json:"customerIds" binding:"required,min=1"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		id, ok := pathIDValid(c, "templateId")
		if !ok {
			return
		}
		n, err := a.Promotion.Issue(c.Request.Context(), id, req.CustomerIDs)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"issued": n})
	}
}

// promoCreateCodes POST /coupon-templates/:templateId/codes {count}:生成兑换码批次。
func promoCreateCodes(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Count int `json:"count" binding:"required,gt=0,max=1000"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		id, ok := pathIDValid(c, "templateId")
		if !ok {
			return
		}
		codes, err := a.Promotion.CreateCodes(c.Request.Context(), id, req.Count)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": codes})
	}
}

// promoListCodes GET /coupon-templates/:templateId/codes:兑换码列表。
func promoListCodes(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := pathIDValid(c, "templateId")
		if !ok {
			return
		}
		codes, err := a.Promotion.ListCodes(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": codes})
	}
}

// promoListGiftRules GET /gift-rules:赠送时长规则列表。
func promoListGiftRules(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Promotion.ListGiftRules(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// promoCreateGiftRule POST /gift-rules:新建赠送规则。
func promoCreateGiftRule(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var r promotion.GiftRule
		if !httpx.BindBody(c, &r) {
			return
		}
		id, err := a.Promotion.CreateGiftRule(c.Request.Context(), r)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ruleId": id})
	}
}

// promoDisableGiftRule PUT /gift-rules/:ruleId/disable:停用赠送规则。
func promoDisableGiftRule(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := pathIDValid(c, "ruleId")
		if !ok {
			return
		}
		if err := a.Promotion.DisableGiftRule(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}
