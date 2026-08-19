package workerapi

// W 师傅端门户辅助构造:领料/工具/旧件返库记录(归属快照由域内默认,当前仅师傅维度)。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/worker"
)

// workerMaterialOf 领料出库记录:name 暂用 itemId 占位(物料主档缺口见报告)。
func workerMaterialOf(c *gin.Context, workerID int64) worker.Material {
	return worker.Material{WorkerID: workerID, Name: c.Param("itemId"), Qty: 1}
}

// workerToolOf 工具借用/归还记录:name 暂用 toolId 占位。
func workerToolOf(c *gin.Context, workerID int64, borrowed bool) worker.Tool {
	return worker.Tool{WorkerID: workerID, Name: c.Param("toolId"), Borrowed: borrowed}
}

// workerAssetReturnOf 旧件返库登记:asset 待 EPC 反查(PENDING 确认流)。
func workerAssetReturnOf(c *gin.Context, workerID int64) worker.AssetReturn {
	return worker.AssetReturn{
		WorkerID: workerID, AssetID: 0, Reason: "旧件返库",
		Status: "PENDING",
	}
}
