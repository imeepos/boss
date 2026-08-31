package adminapi

// 地址层级路由(ltree 物化路径,ADR-002;menu:address 门禁)。
// 具名 handler 见 address_handlers.go;此处仅保留注册表与锚点校验工具。

import (
	"context"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
)

// registerAddressRoutes 注册地址层级路由。
func registerAddressRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/addresses", requirePerm(a.User, "menu:address"), addrListAddresses(a))
	g.PUT("/addresses/:id/geo", requirePerm(a.User, "menu:address"), addrSetGeo(a))
	g.PUT("/addresses/:id/geom", requirePerm(a.User, "menu:address"), addrSetGeom(a))
	g.GET("/addresses/search", requirePerm(a.User, "menu:address"), addrSearchAddresses(a))
	g.GET("/addresses/nearest", requirePerm(a.User, "menu:address"), addrNearestAddress(a))
	g.POST("/addresses", requirePerm(a.User, "menu:address"), addrCreateAddress(a))
	g.PUT("/addresses/:id", requirePerm(a.User, "menu:address"), addrUpdateAddressName(a))
	g.DELETE("/addresses/:id", requirePerm(a.User, "menu:address"), addrDeleteAddress(a))
	g.POST("/addresses/import", requirePerm(a.User, "menu:importer"), addrImportAddresses(a))
}

// validateGeoAnchor 锚点前置校验:国家启用、区划(若给)存在/启用/同国家/一级行政区。
// 非法锚点返回 httpx.ErrGeoInvalidParam(422),国家/区划不存在返回 geo.ErrNotFound(404)。
func validateGeoAnchor(ctx context.Context, a *app.Application, countryCode, adminCode string) error {
	country, err := a.Geo.GetCountry(ctx, countryCode)
	if err != nil {
		return err
	}
	if !country.IsActive {
		return httpx.ErrGeoInvalidParam
	}
	if adminCode == "" {
		return nil
	}
	sub, err := a.Geo.GetSubdivision(ctx, adminCode)
	if err != nil {
		return err
	}
	if !sub.IsActive || sub.CountryCode != countryCode || sub.Level != 1 {
		return httpx.ErrGeoInvalidParam
	}
	return nil
}
