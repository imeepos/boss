package adminapi

import (
	"regexp"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

var ledgerPeriodRe = regexp.MustCompile(`^\d{4}-(0[1-9]|1[0-2])$`)

// ledgerReconHandler GET /billing/ledger-recon:账实核对(应收/实收/开票三角,按账单定位差异)。
// 数据范围沿用当前账号 DataScope(legal_entity + region 子树),与 AAA 管理收敛同一模式。
func ledgerReconHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		svc, ok := a.Billing.(billing.LedgerReconService)
		if !ok {
			respond(c, apitypes.CodeInternal, nil)
			return
		}
		period := c.Query("period")
		if !ledgerPeriodRe.MatchString(period) {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "period must be YYYY-MM"})
			return
		}
		scope, err := a.User.GetDataScope(c.Request.Context(), httpx.ClaimsAccountID(c))
		if err != nil {
			respondErr(c, err)
			return
		}
		legalEntity := queryInt64(c, "legalEntityId")
		if scope.LegalEntityID > 0 {
			legalEntity = scope.LegalEntityID
		}
		q := billing.LedgerReconQuery{
			Period: period, LegalEntityID: legalEntity, RegionScope: scope.RegionScope,
			Page: queryInt(c, "page", 1), PageSize: queryInt(c, "pageSize", 50),
		}
		items, total, summary, err := svc.LedgerRecon(c.Request.Context(), q)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items, "total": total,
			"page": q.Page, "pageSize": q.PageSize, "summary": summary})
	}
}
