package adminapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/aaa"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// aaaSummaryHandler 基于 AAA 真实读模型聚合总览，不复制一套统计表。
func aaaSummaryHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx := c.Request.Context()
		accounts, err := a.Aaa.ListLoAccounts(ctx)
		if err != nil {
			respondErr(c, err)
			return
		}
		cdrs, err := a.Aaa.ListCdrs(ctx, "")
		if err != nil {
			respondErr(c, err)
			return
		}
		auths, err := a.Aaa.ListAuthLogs(ctx, "")
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"summary": summarizeAAA(accounts, cdrs, auths)})
	}
}

func summarizeAAA(accounts []aaa.LoAccount, cdrs []aaa.CdrRecord, auths []aaa.AuthLog) gin.H {
	return gin.H{
		"accounts": len(accounts), "active": countLoStatus(accounts, "ACTIVE"),
		"suspended": countLoStatus(accounts, "SUSPENDED"), "closed": countLoStatus(accounts, "CLOSED"),
		"cdrs": len(cdrs), "unbilled": countCdrStatus(cdrs, "UNBILLED"),
		"authSuccess": countAuthResult(auths, "SUCCESS"), "authFailed": countAuthResult(auths, "FAILED"),
	}
}

func countLoStatus(rows []aaa.LoAccount, status string) int {
	count := 0
	for _, row := range rows {
		if row.Status == status {
			count++
		}
	}
	return count
}

func countCdrStatus(rows []aaa.CdrRecord, status string) int {
	count := 0
	for _, row := range rows {
		if row.BillingStatus == status {
			count++
		}
	}
	return count
}

func countAuthResult(rows []aaa.AuthLog, result string) int {
	count := 0
	for _, row := range rows {
		if row.Result == result {
			count++
		}
	}
	return count
}
