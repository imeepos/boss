package adminapi

// GET /geo/default-country:默认国家读端点(登录管理员即可读,不设 menu 门槛;
// 选择器全局兜底)。写路径复用既有 PUT /params/geo.default_country(menu:params)。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// geoDefaultCountryHandler 读 biz_params geo.default_country。
// 未配置/值非法返回空值对象 {countryCode:"",configured:false}(契约见 geo.yaml getDefaultGeoCountry)。
func geoDefaultCountryHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		d, err := a.Geo.GetDefaultCountry(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, d)
	}
}
