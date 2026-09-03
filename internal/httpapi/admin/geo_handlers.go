package adminapi

// 国际地理基础数据维护路由具名 handler(menu:geo 门禁)。
// 请求体类型与请求体→域对象转换见 geo.go;批量导入见 geo_import.go。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/geo"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

func geoListCountries(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.Geo.ListCountries(c.Request.Context(), c.Query("locale"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}

func geoGetCountry(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		d, err := a.Geo.GetCountry(c.Request.Context(), c.Param("code"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, d)
	}
}

func geoCreateCountryHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		alpha2, err := geoCreateCountry(c, a)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "geo.create-country", "geo_country", alpha2, nil)
		respond(c, apitypes.CodeOK, gin.H{"alpha2": alpha2})
	}
}

func geoUpdateCountry(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req geoCountryReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		err := a.Geo.UpdateCountry(c.Request.Context(), c.Param("code"), geoCountryOf(req))
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "geo.update-country", "geo_country", c.Param("code"), nil)
		respond(c, apitypes.CodeOK, nil)
	}
}

func geoSetCountryActive(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Active bool `json:"active"`
		}
		if !httpx.BindAndValidate(c, &body) {
			return
		}
		if err := a.Geo.SetCountryActive(c.Request.Context(), c.Param("code"), body.Active); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "geo.set-country-active", "geo_country", c.Param("code"), map[string]any{"active": body.Active})
		respond(c, apitypes.CodeOK, nil)
	}
}

func geoAddCountryName(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req geoNameReq
		if !httpx.BindAndValidate(c, &req) {
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
	}
}

func geoRemoveCountryName(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := a.Geo.RemoveCountryName(c.Request.Context(),
			c.Param("code"), c.Param("locale"), c.Param("nameType"))
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "geo.remove-country-name", "geo_country", c.Param("code"), nil)
		respond(c, apitypes.CodeOK, nil)
	}
}

func geoReplaceCountryAttrs(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var attrs geo.CountryAttrs
		if !httpx.BindAndValidate(c, &attrs) {
			return
		}
		if err := a.Geo.ReplaceCountryAttrs(c.Request.Context(), c.Param("code"), attrs); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "geo.replace-country-attrs", "geo_country", c.Param("code"), nil)
		respond(c, apitypes.CodeOK, nil)
	}
}

// geoListSubdivisions GET /geo/subdivisions:country/locale 兼容过滤 + parentCode 下钻
// + keyword 搜索(须配合 country,否则参数错)+ limit 截断(钳制口径在域层,默认 50 最大 200)。
func geoListSubdivisions(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		f := geo.SubdivisionFilter{
			CountryCode: c.Query("country"),
			Locale:      c.Query("locale"),
			Keyword:     c.Query("keyword"),
		}
		// parentCode 三态:缺省不过滤;传空值取顶层节点;传码取直接子节点。
		if _, ok := c.Request.URL.Query()["parentCode"]; ok {
			v := c.Query("parentCode")
			f.ParentCode = &v
		}
		if f.Keyword != "" && f.CountryCode == "" {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "keyword requires country"})
			return
		}
		if n, err := strconv.Atoi(c.Query("limit")); err == nil {
			f.Limit = n
		}
		list, err := a.Geo.ListSubdivisions(c.Request.Context(), f)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	}
}

func geoListSubdivisionNames(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		names, err := a.Geo.ListSubdivisionNames(c.Request.Context(), c.Param("code"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, names)
	}
}

func geoCreateSubdivisionHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		code, err := geoCreateSubdiv(c, a)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "geo.create-subdivision", "geo_subdivision", code, nil)
		respond(c, apitypes.CodeOK, gin.H{"code": code})
	}
}

func geoUpdateSubdivision(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req geoSubdivReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		err := a.Geo.UpdateSubdivision(c.Request.Context(), c.Param("code"), geoSubdivOf(req))
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "geo.update-subdivision", "geo_subdivision", c.Param("code"), nil)
		respond(c, apitypes.CodeOK, nil)
	}
}

func geoSetSubdivisionActive(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Active bool `json:"active"`
		}
		if !httpx.BindAndValidate(c, &body) {
			return
		}
		if err := a.Geo.SetSubdivisionActive(c.Request.Context(), c.Param("code"), body.Active); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "geo.set-subdivision-active", "geo_subdivision", c.Param("code"), map[string]any{"active": body.Active})
		respond(c, apitypes.CodeOK, nil)
	}
}

func geoAddSubdivisionName(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req geoNameReq
		if !httpx.BindAndValidate(c, &req) {
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
	}
}

func geoRemoveSubdivisionName(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		err := a.Geo.RemoveSubdivisionName(c.Request.Context(),
			c.Param("code"), c.Param("locale"), c.Param("nameType"))
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "geo.remove-subdivision-name", "geo_subdivision", c.Param("code"), nil)
		respond(c, apitypes.CodeOK, nil)
	}
}
