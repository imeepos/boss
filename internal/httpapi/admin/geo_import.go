package adminapi

import (
	"errors"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/geo"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerGeoImportRoute 批量导入路由(与单条维护同门禁;单事务 upsert)。
func registerGeoImportRoute(g *gin.RouterGroup, a *app.Application) {
	perm := requirePerm(a.User, "menu:geo")
	g.POST("/geo/import", perm, func(c *gin.Context) {
		var data geo.ImportData
		if err := c.ShouldBindJSON(&data); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if err := validateGeoImport(data); err != nil {
			respond(c, apitypes.CodeInvalidParam, gin.H{"detail": err.Error()})
			return
		}
		counts, err := a.Geo.Import(c.Request.Context(), data)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "geo.import", "geo_country", "", map[string]any{
			"countries": counts.Countries, "countryNames": counts.CountryNames,
			"subdivisions": counts.Subdivisions, "subdivisionNames": counts.SubdivisionNames,
		})
		_ = a.User.RecordImportTask(c.Request.Context(), "geo", httpx.ClaimsAccountID(c),
			counts.Countries+counts.Subdivisions, 0, map[string]any{
				"countries": counts.Countries, "countryNames": counts.CountryNames,
				"subdivisions": counts.Subdivisions, "subdivisionNames": counts.SubdivisionNames,
			})
		respond(c, apitypes.CodeOK, counts)
	})
}

// validateGeoImport 导入载荷逐行校验(规则与单条接口的 binding 一致)。
func validateGeoImport(d geo.ImportData) error {
	for _, c := range d.Countries {
		if len(c.Alpha2) != 2 || len(c.Alpha3) != 3 || len(c.NumericCode) != 3 ||
			c.ShortName == "" || !inSet(c.Status, "INDEPENDENT", "DISCONTINUED") ||
			!inSet(c.ContinentCode, "AS", "EU", "NA", "SA", "AF", "OC", "AN") {
			return errors.New("invalid country row: " + c.Alpha2)
		}
	}
	for _, r := range d.CountryNames {
		if len(r.CountryCode) != 2 || r.Name.Locale == "" || r.Name.Name == "" || r.Name.NameType == "" {
			return errors.New("invalid country name row: " + r.CountryCode)
		}
	}
	for _, s := range d.Subdivisions {
		if s.Code == "" || len(s.CountryCode) != 2 || s.Level < 1 || s.Level > 4 || s.Category == "" {
			return errors.New("invalid subdivision row: " + s.Code)
		}
	}
	for _, r := range d.SubdivisionNames {
		if r.SubdivisionCode == "" || r.Name.Locale == "" || r.Name.Name == "" || r.Name.NameType == "" {
			return errors.New("invalid subdivision name row: " + r.SubdivisionCode)
		}
	}
	return nil
}

// inSet 集合包含判断。
func inSet(v string, set ...string) bool {
	for _, s := range set {
		if v == s {
			return true
		}
	}
	return false
}
