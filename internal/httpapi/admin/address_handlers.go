package adminapi

// 地址层级路由具名 handler(ltree 物化路径,ADR-002;menu:address 门禁)。
// 含国际锚点挂接(geo 域校验)与批量导入(menu:importer)。
// 锚点校验工具 validateGeoAnchor 见 address.go。

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

func addrListAddresses(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

func addrSetGeo(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		var body struct {
			CountryCode string `json:"countryCode" binding:"required,len=2"`
			AdminCode   string `json:"adminCode"`
		}
		if !httpx.BindAndValidate(c, &body) {
			return
		}
		if err := validateGeoAnchor(c.Request.Context(), a, body.CountryCode, body.AdminCode); err != nil {
			respondErr(c, err)
			return
		}
		if err := a.User.SetAddressGeo(c.Request.Context(), id, body.CountryCode, body.AdminCode); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "address.set-geo", "addresses", strconv.FormatInt(id, 10),
			map[string]any{"countryCode": body.CountryCode, "adminCode": body.AdminCode})
		respond(c, apitypes.CodeOK, nil)
	}
}

func addrSearchAddresses(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
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
	}
}

func addrCreateAddress(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			ParentID    int64  `json:"parentId"`
			Label       string `json:"label" binding:"required"`
			Name        string `json:"name" binding:"required"`
			CountryCode string `json:"countryCode"`
			AdminCode   string `json:"adminCode"`
		}
		if !httpx.BindAndValidate(c, &body, func() error {
			return httpx.CollectErrors(
				httpx.RequireString(body.Label, "label", 32),
				httpx.RequireString(body.Name, "name", 128),
			)
		}) {
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
	}
}

func addrUpdateAddressName(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		var body struct {
			Name string `json:"name" binding:"required"`
		}
		if !httpx.BindAndValidate(c, &body, func() error {
			return httpx.CollectErrors(
				httpx.RequireString(body.Name, "name", 128),
			)
		}) {
			return
		}
		if err := a.User.UpdateAddressName(c.Request.Context(), id, body.Name); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "address.rename", "addresses", strconv.FormatInt(id, 10),
			map[string]any{"name": body.Name})
		respond(c, apitypes.CodeOK, nil)
	}
}

func addrDeleteAddress(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		if err := a.User.DeleteAddress(c.Request.Context(), id); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "address.delete", "addresses", strconv.FormatInt(id, 10), nil)
		respond(c, apitypes.CodeOK, nil)
	}
}

func addrImportAddresses(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Rows []struct {
				Path        string `json:"path" binding:"required"`
				Name        string `json:"name" binding:"required"`
				CountryCode string `json:"countryCode"`
				AdminCode   string `json:"adminCode"`
			} `json:"rows" binding:"required"`
		}
		if !httpx.BindAndValidate(c, &req) {
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
		emitTask(c.Request.Context(), a, refImporter, "addr-"+fmt.Sprint(time.Now().Unix()),
			"地址导入完成:"+strconv.Itoa(int(imported))+" 行", linkImporter, false)
		respond(c, apitypes.CodeOK, gin.H{"imported": imported})
	}
}
