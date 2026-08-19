package adminapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/geo"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
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

// registerGeoRoutes 注册国际地理基础数据维护路由(menu:geo 门禁,sysadmin)。
func registerGeoRoutes(g *gin.RouterGroup, a *app.Application) {
	perm := requirePerm(a.User, "menu:geo")
	registerGeoImportRoute(g, a)

	// 国家:列表/详情/新建/编辑/启停。
	g.GET("/geo/countries", perm, func(c *gin.Context) {
		list, err := a.Geo.ListCountries(c.Request.Context(), c.Query("locale"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	})
	g.GET("/geo/countries/:code", perm, func(c *gin.Context) {
		d, err := a.Geo.GetCountry(c.Request.Context(), c.Param("code"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, d)
	})
	g.POST("/geo/countries", perm, func(c *gin.Context) {
		alpha2, err := geoCreateCountry(c, a)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "geo.create-country", "geo_country", alpha2, nil)
		respond(c, apitypes.CodeOK, gin.H{"alpha2": alpha2})
	})
	g.PUT("/geo/countries/:code", perm, func(c *gin.Context) {
		var req geoCountryReq
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		err := a.Geo.UpdateCountry(c.Request.Context(), c.Param("code"), geoCountryOf(req))
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "geo.update-country", "geo_country", c.Param("code"), nil)
		respond(c, apitypes.CodeOK, nil)
	})
	g.PUT("/geo/countries/:code/active", perm, func(c *gin.Context) {
		var body struct {
			Active bool `json:"active"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if err := a.Geo.SetCountryActive(c.Request.Context(), c.Param("code"), body.Active); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "geo.set-country-active", "geo_country", c.Param("code"), map[string]any{"active": body.Active})
		respond(c, apitypes.CodeOK, nil)
	})

	// 国家译名与关联属性。
	g.POST("/geo/countries/:code/names", perm, func(c *gin.Context) {
		var req geoNameReq
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		err := a.Geo.AddCountryName(c.Request.Context(), c.Param("code"),
			geo.CountryName{Locale: req.Locale, Name: req.Name, NameType: req.NameType})
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "geo.add-country-name", "geo_country", c.Param("code"), nil)
		respond(c, apitypes.CodeOK, nil)
	})
	g.DELETE("/geo/countries/:code/names/:locale/:nameType", perm, func(c *gin.Context) {
		err := a.Geo.RemoveCountryName(c.Request.Context(),
			c.Param("code"), c.Param("locale"), c.Param("nameType"))
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "geo.remove-country-name", "geo_country", c.Param("code"), nil)
		respond(c, apitypes.CodeOK, nil)
	})
	g.PUT("/geo/countries/:code/attrs", perm, func(c *gin.Context) {
		var attrs geo.CountryAttrs
		if err := c.ShouldBindJSON(&attrs); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if err := a.Geo.ReplaceCountryAttrs(c.Request.Context(), c.Param("code"), attrs); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "geo.replace-country-attrs", "geo_country", c.Param("code"), nil)
		respond(c, apitypes.CodeOK, nil)
	})

	// 区划:列表/新建/编辑/启停/译名。
	g.GET("/geo/subdivisions", perm, func(c *gin.Context) {
		list, err := a.Geo.ListSubdivisions(c.Request.Context(),
			c.Query("country"), c.Query("locale"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	})
	g.GET("/geo/subdivisions/:code/names", perm, func(c *gin.Context) {
		names, err := a.Geo.ListSubdivisionNames(c.Request.Context(), c.Param("code"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, names)
	})
	g.POST("/geo/subdivisions", perm, func(c *gin.Context) {
		code, err := geoCreateSubdiv(c, a)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "geo.create-subdivision", "geo_subdivision", code, nil)
		respond(c, apitypes.CodeOK, gin.H{"code": code})
	})
	g.PUT("/geo/subdivisions/:code", perm, func(c *gin.Context) {
		var req geoSubdivReq
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		err := a.Geo.UpdateSubdivision(c.Request.Context(), c.Param("code"), geoSubdivOf(req))
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "geo.update-subdivision", "geo_subdivision", c.Param("code"), nil)
		respond(c, apitypes.CodeOK, nil)
	})
	g.PUT("/geo/subdivisions/:code/active", perm, func(c *gin.Context) {
		var body struct {
			Active bool `json:"active"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if err := a.Geo.SetSubdivisionActive(c.Request.Context(), c.Param("code"), body.Active); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "geo.set-subdivision-active", "geo_subdivision", c.Param("code"), map[string]any{"active": body.Active})
		respond(c, apitypes.CodeOK, nil)
	})
	g.POST("/geo/subdivisions/:code/names", perm, func(c *gin.Context) {
		var req geoNameReq
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		err := a.Geo.AddSubdivisionName(c.Request.Context(), c.Param("code"),
			geo.SubdivisionName{Locale: req.Locale, Name: req.Name, NameType: req.NameType})
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "geo.add-subdivision-name", "geo_subdivision", c.Param("code"), nil)
		respond(c, apitypes.CodeOK, nil)
	})
	g.DELETE("/geo/subdivisions/:code/names/:locale/:nameType", perm, func(c *gin.Context) {
		err := a.Geo.RemoveSubdivisionName(c.Request.Context(),
			c.Param("code"), c.Param("locale"), c.Param("nameType"))
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "geo.remove-subdivision-name", "geo_subdivision", c.Param("code"), nil)
		respond(c, apitypes.CodeOK, nil)
	})
}

// geoCreateCountry 新建国家(主键 alpha2 在 URL 外的 body 顶层)。
func geoCreateCountry(c *gin.Context, a *app.Application) (string, error) {
	var body struct {
		Alpha2 string `json:"alpha2" binding:"required,len=2"`
		geoCountryReq
	}
	if err := c.ShouldBindJSON(&body); err != nil {
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
	if err := c.ShouldBindJSON(&body); err != nil {
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
