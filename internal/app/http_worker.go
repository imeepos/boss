package app

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerWorkerRoutes 注册师傅域路由(承接 api/openapi/admin/worker.yaml)。
func registerWorkerRoutes(g *gin.RouterGroup, a *Application) {
	g.GET("/worker-groups", requirePerm(a.User, "menu:order"), func(c *gin.Context) {
		list, err := a.Worker.ListGroups(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.GET("/workers", requirePerm(a.User, "menu:dispatch"), func(c *gin.Context) {
		list, err := a.Worker.ListWorkers(c.Request.Context(), queryInt64(c, "groupId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.GET("/worker-performances", requirePerm(a.User, "menu:order"), func(c *gin.Context) {
		list, err := a.WorkerFact.ListPerformances(c.Request.Context(), queryInt64(c, "workerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.GET("/worker-commissions", requirePerm(a.User, "menu:order"), func(c *gin.Context) {
		list, err := a.WorkerFact.ListCommissions(c.Request.Context(), queryInt64(c, "workerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.GET("/worker-schedules", requirePerm(a.User, "menu:order"), func(c *gin.Context) {
		list, err := a.WorkerFact.ListSchedules(c.Request.Context(), queryInt64(c, "workerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.GET("/worker-materials", requirePerm(a.User, "menu:order"), func(c *gin.Context) {
		list, err := a.WorkerEvent.ListMaterials(c.Request.Context(), queryInt64(c, "workerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.GET("/worker-tools", requirePerm(a.User, "menu:order"), func(c *gin.Context) {
		list, err := a.WorkerEvent.ListTools(c.Request.Context(), queryInt64(c, "workerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.GET("/worker-feedbacks", requirePerm(a.User, "menu:order"), func(c *gin.Context) {
		list, err := a.WorkerEvent.ListFeedbacks(c.Request.Context(), queryInt64(c, "workerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.GET("/asset-returns", requirePerm(a.User, "menu:order"), func(c *gin.Context) {
		list, err := a.WorkerEvent.ListAssetReturns(c.Request.Context(), queryInt64(c, "workerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})

	g.GET("/worker-messages", requirePerm(a.User, "menu:dispatch"), func(c *gin.Context) {
		list, err := a.WorkerLedger.ListMessages(c.Request.Context(), queryInt64(c, "workerId"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	})
}
