package httpx

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/quadlink"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// RespondScanErr 扫码类端点错误映射:四码不一致/未预绑/缺参/地址冲突单独编码,其余走领域映射。
func RespondScanErr(c *gin.Context, err error) {
	switch {
	case errors.Is(err, quadlink.ErrScanMismatch):
		Respond(c, apitypes.CodeScanMismatch, nil)
	case errors.Is(err, quadlink.ErrAddressConflict):
		// 同地址活跃链路归属他客(重装复用守卫,2026-09-04 任务A):40920 族 + 原因透传。
		Respond(c, apitypes.CodeScanMismatch, gin.H{"reason": err.Error()})
	case errors.Is(err, quadlink.ErrNotPrebound):
		Respond(c, apitypes.CodeStateInvalid, nil)
	case errors.Is(err, quadlink.ErrScanRequired):
		Respond(c, apitypes.CodeInvalidParam, nil)
	default:
		RespondErr(c, err)
	}
}
