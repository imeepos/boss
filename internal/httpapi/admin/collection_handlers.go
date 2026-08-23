package adminapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

func listCollectionTasksHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.CollectionQueue == nil {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		items, err := a.CollectionQueue.ListCollectionTasks(c.Request.Context(), c.Query("status"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

func updateCollectionTaskHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "taskId")
		if !ok {
			return
		}
		var body struct {
			Status  string `json:"status" binding:"required"`
			Outcome string `json:"outcome"`
			Note    string `json:"note"`
		}
		if !httpx.BindBody(c, &body) {
			return
		}
		if err := a.CollectionQueue.UpdateCollectionTask(c.Request.Context(), id, body.Status, body.Outcome, body.Note); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}
