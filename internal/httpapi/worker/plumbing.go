package workerapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// respond/respondErr 薄包装,端内 handler 保持原调用形态。
func respond(c *gin.Context, code apitypes.Code, data any) { httpx.Respond(c, code, data) }
func respondErr(c *gin.Context, err error)                 { httpx.RespondErr(c, err) }
