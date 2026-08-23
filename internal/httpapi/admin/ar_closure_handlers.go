package adminapi

import (
	"github.com/gin-gonic/gin"
	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/pkg/clock"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
	"time"
)

func parseSnapshotDate(c *gin.Context) (time.Time, bool) {
	day := clock.Now()
	if value := c.Query("date"); value != "" {
		parsed, err := time.ParseInLocation("2006-01-02", value, clock.Location())
		if err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return time.Time{}, false
		}
		day = parsed
	}
	return day, true
}
func generateAgingSnapshots(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		day, ok := parseSnapshotDate(c)
		if !ok {
			return
		}
		count, err := a.ARClosure.GenerateAgingSnapshots(c.Request.Context(), day)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"date": day.Format("2006-01-02"), "generated": count})
	}
}
func listAgingSnapshots(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		day, ok := parseSnapshotDate(c)
		if !ok {
			return
		}
		items, err := a.ARClosure.ListAgingSnapshots(c.Request.Context(), day, queryInt64(c, "customerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}
func listPaymentPromises(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := a.ARClosure.ListPaymentPromises(c.Request.Context(), queryInt64(c, "customerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}
func createPaymentPromise(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var v billing.PaymentPromise
		if !httpx.BindAndValidate(c, &v) {
			return
		}
		id, err := a.ARClosure.CreatePaymentPromise(c.Request.Context(), v)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}
func updatePaymentPromiseStatus(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := pathIDValid(c, "id")
		if !ok {
			return
		}
		var body struct {
			Status string `json:"status"`
		}
		if !httpx.BindAndValidate(c, &body) {
			return
		}
		if err := a.ARClosure.UpdatePaymentPromiseStatus(c.Request.Context(), id, body.Status); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}
func listWriteoffs(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		items, err := a.ARClosure.ListWriteoffs(c.Request.Context(), queryInt64(c, "customerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}
func createWriteoff(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var v billing.Writeoff
		if !httpx.BindAndValidate(c, &v) {
			return
		}
		id, err := a.ARClosure.CreateWriteoff(c.Request.Context(), v)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}
