package adminapi

// Q2 每日数据对账读取:GET /reports/recon/latest(五域检查结果,menu:report)。

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/report"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// reportReconLatestHandler GET /reports/recon/latest:最新每日对账快照;无快照 404。
func reportReconLatestHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, err := a.Report.LatestRecon(c.Request.Context())
		if err != nil {
			if errors.Is(err, report.ErrNoSnapshot) {
				respond(c, apitypes.CodeNotFound, nil)
				return
			}
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, p)
	}
}
