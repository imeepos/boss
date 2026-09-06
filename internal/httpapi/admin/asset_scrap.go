package adminapi

// 报废资产 handler(P3-F 三要素确认版,自 asset_handlers.go 迁出控制行数):
// confirmAssetCode/confirmSn/confirmTagNo 服务端强校验防绕过前端,不符 422(领域
// ErrScrapConfirmMismatch 经 RespondErr 统一映射);审计沿用 RecordAudit,confirmSn
// 只记末 4 位,不落全量 SN。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// assetScrapHandler POST /assets/{assetId}/scrap:报废资产(P1-T2+P3-F 三要素确认)。
// 终态幂等(重放恒成功无二次副作用);三要素与现值不符 422 且信息不回显现值;
// 标签仍绑时强制解绑写 RECYCLE 事件,轨迹落 SCRAP 行(同事务)。
func assetScrapHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "assetId")
		if !ok {
			return
		}
		var req struct {
			Reason           string `json:"reason"`
			ConfirmAssetCode string `json:"confirmAssetCode"`
			ConfirmSn        string `json:"confirmSn"`
			ConfirmTagNo     string `json:"confirmTagNo"`
		}
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequireString(req.Reason, "reason", 64),
			)
		}) {
			return
		}
		confirm := asset.ScrapConfirm{AssetCode: req.ConfirmAssetCode, SN: req.ConfirmSn, TagNo: req.ConfirmTagNo}
		if err := a.Asset.ScrapAsset(c.Request.Context(), id, httpx.ClaimsAccountID(c), req.Reason, confirm); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "状态变更", "asset", c.Param("assetId"), map[string]any{
			"op": "scrap", "reason": req.Reason, "confirmSnLast4": snLast4(req.ConfirmSn)})
		respond(c, apitypes.CodeOK, nil)
	}
}

// snLast4 审计脱敏:只留末 4 位;不足 4 位时末 4 位即全量,原样返回(长度不构成额外泄露)。
func snLast4(sn string) string {
	if len(sn) <= 4 {
		return sn
	}
	return sn[len(sn)-4:]
}
