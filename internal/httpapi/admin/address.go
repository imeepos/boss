package adminapi

// 地址层级路由(ltree 物化路径,ADR-002;menu:address 门禁)。
// 含国际锚点挂接(geo 域校验)与批量导入(menu:importer)。

import (
	"context"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

func registerAddressRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/addresses", requirePerm(a.User, "menu:address"), func(c *gin.Context) {
		var (
			list []user.Address
			err  error
		)
		if c.Query("unlinked") == "1" {
			list, err = a.User.ListUnlinkedRoots(c.Request.Context())
		} else {
			list, err = a.User.ListAddresses(c.Request.Context(), queryInt64(c, "parentId"))
		}
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, list)
	})

	// 挂接/改挂国家与一级行政区锚点(仅根节点;字段口径 docs/contract/fields.md 1.5.1)。
	g.PUT("/addresses/:id/geo", requirePerm(a.User, "menu:address"), func(c *gin.Context) {
		var body struct {
			CountryCode string `json:"countryCode" binding:"required,len=2"`
			AdminCode   string `json:"adminCode"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if err := validateGeoAnchor(c.Request.Context(), a, body.CountryCode, body.AdminCode); err != nil {
			respondErr(c, err)
			return
		}
		id := queryInt64(c, "id")
		if err := a.User.SetAddressGeo(c.Request.Context(), id, body.CountryCode, body.AdminCode); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "address.set-geo", "addresses", strconv.FormatInt(id, 10),
			map[string]any{"countryCode": body.CountryCode, "adminCode": body.AdminCode})
		respond(c, apitypes.CodeOK, nil)
	})

	// 全树搜索:命中节点 + 祖先链(前端懒加载树自动展开用)。
	g.GET("/addresses/search", requirePerm(a.User, "menu:address"), func(c *gin.Context) {
		kw := c.Query("q")
		if kw == "" {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		hits, err := a.User.SearchAddresses(c.Request.Context(), kw)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, hits)
	})

	// 新增节点:parentId=0 建根(可带锚点),否则加子级;label 为 path 末段(小写字母数字)。
	g.POST("/addresses", requirePerm(a.User, "menu:address"), func(c *gin.Context) {
		var body struct {
			ParentID    int64  `json:"parentId"`
			Label       string `json:"label" binding:"required"`
			Name        string `json:"name" binding:"required"`
			CountryCode string `json:"countryCode"`
			AdminCode   string `json:"adminCode"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if body.CountryCode != "" || body.AdminCode != "" {
			if err := validateGeoAnchor(c.Request.Context(), a, body.CountryCode, body.AdminCode); err != nil {
				respondErr(c, err)
				return
			}
		}
		id, err := a.User.CreateAddress(c.Request.Context(),
			body.ParentID, body.Label, body.Name, body.CountryCode, body.AdminCode)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "address.create", "addresses", strconv.FormatInt(id, 10), map[string]any{
			"parentId": body.ParentID, "label": body.Label, "name": body.Name,
		})
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	})

	// 改名(path 权威不可变)。
	g.PUT("/addresses/:id", requirePerm(a.User, "menu:address"), func(c *gin.Context) {
		var body struct {
			Name string `json:"name" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		id := queryInt64(c, "id")
		if err := a.User.UpdateAddressName(c.Request.Context(), id, body.Name); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "address.rename", "addresses", strconv.FormatInt(id, 10),
			map[string]any{"name": body.Name})
		respond(c, apitypes.CodeOK, nil)
	})

	// 删除叶节点(有子级/被业务引用返回 409)。
	g.DELETE("/addresses/:id", requirePerm(a.User, "menu:address"), func(c *gin.Context) {
		id := queryInt64(c, "id")
		if err := a.User.DeleteAddress(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "address.delete", "addresses", strconv.FormatInt(id, 10), nil)
		respond(c, apitypes.CodeOK, nil)
	})

	g.POST("/addresses/import", requirePerm(a.User, "menu:importer"), func(c *gin.Context) {
		var req struct {
			Rows []struct {
				Path        string `json:"path" binding:"required"`
				Name        string `json:"name" binding:"required"`
				CountryCode string `json:"countryCode"`
				AdminCode   string `json:"adminCode"`
			} `json:"rows" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		rows := make([]user.AddressRow, 0, len(req.Rows))
		for _, r := range req.Rows {
			rows = append(rows, user.AddressRow{
				Path: r.Path, Name: r.Name,
				CountryCode: r.CountryCode, AdminCode: r.AdminCode,
			})
		}
		imported, err := a.User.ImportAddresses(c.Request.Context(), rows)
		if err != nil {
			respondErr(c, err)
			return
		}
		_ = a.User.RecordImportTask(c.Request.Context(), "addresses", httpx.ClaimsAccountID(c), int(imported), 0, nil)
		respond(c, apitypes.CodeOK, gin.H{"imported": imported})
	})
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
