package workerapi

// W 师傅端门户辅助构造:领料/工具/旧件返库记录(名称来自主档,归属快照由域内默认)。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// lookupMaterialItem 按路径 itemId 查物料主档;未命中/非数字直接回 404 语义。
func lookupMaterialItem(c *gin.Context, a *app.Application) *worker.MaterialItem {
	id, err := strconv.ParseInt(c.Param("itemId"), 10, 64)
	if err != nil {
		respond(c, apitypes.CodeNotFound, nil)
		return nil
	}
	item, err := a.WorkerEvent.GetMaterialItem(c.Request.Context(), id)
	if err == worker.ErrNotFound {
		respond(c, apitypes.CodeNotFound, nil)
		return nil
	}
	if err != nil {
		respondErr(c, err)
	}
	return item
}

// lookupToolItem 按路径 toolId 查工具主档;未命中/非数字直接回 404 语义。
func lookupToolItem(c *gin.Context, a *app.Application) *worker.ToolItem {
	id, err := strconv.ParseInt(c.Param("toolId"), 10, 64)
	if err != nil {
		respond(c, apitypes.CodeNotFound, nil)
		return nil
	}
	tool, err := a.WorkerEvent.GetTool(c.Request.Context(), id)
	if err == worker.ErrNotFound {
		respond(c, apitypes.CodeNotFound, nil)
		return nil
	}
	if err != nil {
		respondErr(c, err)
	}
	return tool
}

// workerAssetReturnOf 旧件返库登记:asset 待 EPC 反查(PENDING 确认流)。
func workerAssetReturnOf(c *gin.Context, workerID int64) worker.AssetReturn {
	return worker.AssetReturn{
		WorkerID: workerID, AssetID: 0, Reason: "旧件返库",
		Status: "PENDING",
	}
}
