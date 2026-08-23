package adminapi

import (
	"github.com/gin-gonic/gin"
	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/cs"
	"github.com/ymm-001/boss/internal/pkg/clock"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerCSClosureRoutes registers CS ticket lifecycle routes.
func registerCSClosureRoutes(g *gin.RouterGroup, a *app.Application) {
	perm := requirePerm(a.User, "menu:complaint")
	g.POST("/complaints/:ticketNo/status", perm, transitionTicketHandler(a))
	g.POST("/complaints/:ticketNo/escalate", perm, escalateTicketHandler(a))
	g.GET("/complaints/:ticketNo/events", perm, listTicketEventsHandler(a))
	g.POST("/callbacks/:callbackId/complete", perm, completeCallbackHandler(a))
	g.POST("/ar/credit-profiles/:customerId/evaluate", requirePerm(a.User, "menu:arrears"), evaluateCreditProfileHandler(a))
	g.POST("/ar/collection-tasks/generate", requirePerm(a.User, "menu:arrears"), generateCollectionTasksHandler(a))
	g.POST("/ar/writeoffs/:id/approve", requirePerm(a.User, "menu:arrears"), approveWriteoffHandler(a))
	g.POST("/ar/writeoffs/:id/reject", requirePerm(a.User, "menu:arrears"), rejectWriteoffHandler(a))
	g.POST("/ar/replay/:sourceType", requirePerm(a.User, "menu:arrears"), replayEventsHandler(a))
	g.GET("/ar/replay-events", requirePerm(a.User, "menu:arrears"), listReplayEventsHandler(a))
}

func transitionTicketHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		ticketNo := c.Param("ticketNo")
		var body struct {
			Status string `json:"status"`
			Actor  int64  `json:"actor"`
			Note   string `json:"note"`
		}
		if !httpx.BindBody(c, &body) {
			return
		}
		if err := a.Tickets.TransitionTicket(c.Request.Context(), ticketNo, body.Status, body.Actor, body.Note); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

func escalateTicketHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		ticketNo := c.Param("ticketNo")
		var body struct {
			Actor int64  `json:"actor"`
			Note  string `json:"note"`
		}
		if !httpx.BindBody(c, &body) {
			return
		}
		if err := a.Tickets.EscalateTicket(c.Request.Context(), ticketNo, body.Actor, body.Note); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

func listTicketEventsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		events, err := a.Tickets.ListTicketEventsByNo(c.Request.Context(), c.Param("ticketNo"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": events})
	}
}

func completeCallbackHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "callbackId")
		if !ok {
			return
		}
		var body struct {
			Result     string `json:"result"`
			Rating     int16  `json:"rating"`
			Comment    string `json:"comment"`
			OperatorID int64  `json:"operatorId"`
		}
		if !httpx.BindBody(c, &body) {
			return
		}
		now := clock.Now()
		actual := cs.Callback{
			CompletedAt: &now, Result: body.Result, Rating: body.Rating,
			Comment: body.Comment, OperatorID: body.OperatorID,
		}
		if err := a.Callbacks.CompleteCallback(c.Request.Context(), id, actual); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

func evaluateCreditProfileHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		customerID, ok := httpx.ParsePathParamInt64(c, "customerId")
		if !ok {
			return
		}
		cp, err := a.ARClosure.EvaluateCreditProfile(c.Request.Context(), customerID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, cp)
	}
}

func generateCollectionTasksHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		day := clock.Now()
		count, err := a.ARClosure.GenerateCollectionTasks(c.Request.Context(), day)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"date": day.Format("2006-01-02"), "generated": count})
	}
}

func approveWriteoffHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		var body struct {
			Approver int64 `json:"approver"`
		}
		if !httpx.BindBody(c, &body) {
			return
		}
		if err := a.ARClosure.ApproveWriteoff(c.Request.Context(), id, body.Approver); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

func rejectWriteoffHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		var body struct {
			Note string `json:"note"`
		}
		if !httpx.BindBody(c, &body) {
			return
		}
		if err := a.ARClosure.RejectWriteoff(c.Request.Context(), id, body.Note); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

func replayEventsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		count, err := a.ARClosure.ReplayEvents(c.Request.Context(), c.Param("sourceType"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"sourceType": c.Param("sourceType"), "replayed": count})
	}
}

func listReplayEventsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.ARClosure.ListReplayEvents(c.Request.Context(), c.Query("sourceType"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}
