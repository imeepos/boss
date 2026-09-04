package workerapi

// W 师傅端门户辅助构造:领料/工具/旧件返库记录(名称来自主档,归属快照服务端解析)。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/worker"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// lookupMaterialItem 按路径 itemId 查物料主档;未命中/非数字直接回 404 语义。
func lookupMaterialItem(c *gin.Context, a *app.Application) *worker.MaterialItem {
	id, ok := httpx.ParsePathParamInt64(c, "itemId")
	if !ok {
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
	id, ok := httpx.ParsePathParamInt64(c, "toolId")
	if !ok {
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

// workerAssetReturnOf 旧件返库登记:asset 待 EPC 反查(PENDING 确认流);快照取自师傅主档。
func workerAssetReturnOf(c *gin.Context, snap *worker.FactSnapshot) worker.AssetReturn {
	return worker.AssetReturn{
		WorkerID: snap.WorkerID, GroupID: snap.GroupID, GroupName: snap.GroupName,
		LegalEntityID: snap.LegalEntityID, LegalEntityName: snap.LegalEntityName,
		RegionID: snap.RegionID, RegionName: snap.RegionName,
		AssetID: 0, Reason: "旧件返库",
		Status: "PENDING",
	}
}
