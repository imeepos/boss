package adminapi

// 订单单资源数据范围守卫:镜像订单 List 的实体+区域子树语义;
// 越界以 ErrOrderNotFound 同码同响应写回,保证"不存在/越界"不可区分(2026-08-28 battle oracle)。

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/httpx"
)

// requireOrderInScope 校验订单落在账号数据范围内;越界时写回 NotFound 并返回 false。
func requireOrderInScope(c *gin.Context, a *app.Application, o *order.Order) bool {
	scope, err := a.User.GetDataScope(c.Request.Context(), httpx.ClaimsAccountID(c))
	if err != nil {
		respondErr(c, err)
		return false
	}
	if inDataScope(scope, o.LegalEntityID, o.RegionPath) {
		return true
	}
	respondErr(c, order.ErrOrderNotFound)
	return false
}

// inDataScope 镜像 List SQL 范围谓词:实体 0 不限;区域空不限,否则 path 相等或为其子树。
func inDataScope(scope user.DataScope, legalEntityID int64, regionPath string) bool {
	if scope.LegalEntityID != 0 && legalEntityID != scope.LegalEntityID {
		return false
	}
	return scope.RegionScope == "" || regionPath == scope.RegionScope ||
		strings.HasPrefix(regionPath, scope.RegionScope+".")
}
