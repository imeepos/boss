package adminapi

// 国际地理基础数据维护路由(menu:geo 门禁,sysadmin)。
// 具名 handler 见 geo_handlers.go;批量导入见 geo_import.go。

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/geo"
	"github.com/ymm-001/boss/internal/pkg/httpx"
)

// httpx.ErrGeoInvalidParam 参数绑定失败哨兵,respondErr 统一映射 CodeInvalidParam。

// geoCountryReq 国家新建/编辑请求体(字段口径 docs/contract/fields.md 1.5.1)。
type geoCountryReq struct {
	Alpha3        string `json:"alpha3" binding:"required,len=3"`
	NumericCode   string `json:"numericCode" binding:"required,len=3"`
	ShortName     string `json:"shortName" binding:"required"`
	FullName      string `json:"fullName"`
	Status        string `json:"status" binding:"required,oneof=INDEPENDENT DISCONTINUED"`
	ContinentCode string `json:"continentCode" binding:"required,oneof=AS EU NA SA AF OC AN"`
	M49Region     string `json:"m49Region"`
	PostalRegex   string `json:"postalRegex"`
}

// geoSubdivReq 区划新建/编辑请求体。
type geoSubdivReq struct {
	CountryCode   string `json:"countryCode" binding:"required,len=2"`
	ParentCode    string `json:"parentCode"`
	Level         int16  `json:"level" binding:"required,min=1,max=4"`
	Category      string `json:"category" binding:"required"`
	OSMAdminLevel int16  `json:"osmAdminLevel"`
	GeonameID     int64  `json:"geonameId"`
}

// geoNameReq 译名新增请求体。
type geoNameReq struct {
	Locale   string `json:"locale" binding:"required"`
	Name     string `json:"name" binding:"required"`
	NameType string `json:"nameType" binding:"required"`
}

// registerGeoRoutes 注册国际地理基础数据维护路由。
func registerGeoRoutes(g *gin.RouterGroup, a *app.Application) {
	perm := requirePerm(a.User, "menu:geo")
	registerGeoImportRoute(g, a)

	// 国家:列表/详情/新建/编辑/启停。
	g.GET("/geo/countries", perm, geoListCountries(a))
	g.GET("/geo/countries/:code", perm, geoGetCountry(a))
	g.POST("/geo/countries", perm, geoCreateCountryHandler(a))
	g.PUT("/geo/countries/:code", perm, geoUpdateCountry(a))
	g.PUT("/geo/countries/:code/active", perm, geoSetCountryActive(a))

	// 国家译名与关联属性。
	g.POST("/geo/countries/:code/names", perm, geoAddCountryName(a))
	g.DELETE("/geo/countries/:code/names/:locale/:nameType", perm, geoRemoveCountryName(a))
	g.PUT("/geo/countries/:code/attrs", perm, geoReplaceCountryAttrs(a))

	// 区划:列表/新建/编辑/启停/译名。
	g.GET("/geo/subdivisions", perm, geoListSubdivisions(a))
	g.GET("/geo/subdivisions/:code/names", perm, geoListSubdivisionNames(a))
	g.POST("/geo/subdivisions", perm, geoCreateSubdivisionHandler(a))
	g.PUT("/geo/subdivisions/:code", perm, geoUpdateSubdivision(a))
	g.PUT("/geo/subdivisions/:code/active", perm, geoSetSubdivisionActive(a))
	g.POST("/geo/subdivisions/:code/names", perm, geoAddSubdivisionName(a))
	g.DELETE("/geo/subdivisions/:code/names/:locale/:nameType", perm, geoRemoveSubdivisionName(a))
}

// geoCreateCountry 新建国家(主键 alpha2 在 URL 外的 body 顶层)。
func geoCreateCountry(c *gin.Context, a *app.Application) (string, error) {
	var body struct {
		Alpha2 string `json:"alpha2" binding:"required,len=2"`
		geoCountryReq
	}
	if !httpx.BindAndValidate(c, &body) {
		return "", httpx.ErrGeoInvalidParam
	}
	co := geoCountryOf(body.geoCountryReq)
	co.Alpha2 = body.Alpha2
	if err := a.Geo.CreateCountry(c.Request.Context(), co); err != nil {
		return "", err
	}
	return body.Alpha2, nil
}

// geoCreateSubdiv 新建区划。
func geoCreateSubdiv(c *gin.Context, a *app.Application) (string, error) {
	var body struct {
		Code string `json:"code" binding:"required"`
		geoSubdivReq
	}
	if !httpx.BindAndValidate(c, &body) {
		return "", httpx.ErrGeoInvalidParam
	}
	d := geoSubdivOf(body.geoSubdivReq)
	d.Code = body.Code
	if err := a.Geo.CreateSubdivision(c.Request.Context(), d); err != nil {
		return "", err
	}
	return body.Code, nil
}

// geoCountryOf 请求体 → 域对象。
func geoCountryOf(req geoCountryReq) geo.Country {
	return geo.Country{
		Alpha3: req.Alpha3, NumericCode: req.NumericCode, ShortName: req.ShortName,
		FullName: req.FullName, Status: req.Status, ContinentCode: req.ContinentCode,
		M49Region: req.M49Region, PostalRegex: req.PostalRegex, IsActive: true,
	}
}

// geoSubdivOf 请求体 → 域对象。
func geoSubdivOf(req geoSubdivReq) geo.Subdivision {
	return geo.Subdivision{
		CountryCode: req.CountryCode, ParentCode: req.ParentCode, Level: req.Level,
		Category: req.Category, OSMAdminLevel: req.OSMAdminLevel,
		GeonameID: req.GeonameID, IsActive: true,
	}
}
