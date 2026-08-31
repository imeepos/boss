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
		// 待治理过滤优先(治理队列入口,fields.md §1.5.0b);与 unlinked 互斥,与前端 toggle 行为对齐。
		switch {
		case c.Query("needsReview") == "1":
			list, err = a.User.ListNeedsReview(c.Request.Context())
		case c.Query("unlinked") == "1":
			list, err = a.User.ListUnlinkedRoots(c.Request.Context())
		default:
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

func addrSetGeom(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, ok := httpx.ParsePathParamInt64(c, "id")
		if !ok {
			return
		}
		var body struct {
			Lat float64 `json:"lat" binding:"required"`
			Lng float64 `json:"lng" binding:"required"`
		}
		if !httpx.BindAndValidate(c, &body) {
			return
		}
		// 范围前置拒(域层同校验兜底);binding:required 顺带拦 0,0(几内亚湾,非法语义)。
		if body.Lat < -90 || body.Lat > 90 || body.Lng < -180 || body.Lng > 180 {
			respondErr(c, user.ErrInvalidInput)
			return
		}
		if err := a.User.SetAddressGeom(c.Request.Context(), id, body.Lat, body.Lng); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "address.set-geom", "addresses", strconv.FormatInt(id, 10),
			map[string]any{"lat": body.Lat, "lng": body.Lng})
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
		// hasMore 暂不透出:admin 响应保持纯数组形状不变(懒加载树按需展开,
		// 截断提示先只做用户端);用户端 /address-tree/search 已透出。
		hits, _, err := a.User.SearchAddresses(c.Request.Context(), kw)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, hits)
	}
}

// addrNearestAddress 逆地理最近邻:GET /addresses/nearest?lat=&lng=&radiusM=
// radiusM 缺省 500,上限 50000(信度上限防远距离误配);未命中回 data:null 非错误。
func addrNearestAddress(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		lat, errLat := strconv.ParseFloat(c.Query("lat"), 64)
		lng, errLng := strconv.ParseFloat(c.Query("lng"), 64)
		if errLat != nil || errLng != nil {
			respondErr(c, user.ErrInvalidInput)
			return
		}
		radius := 500.0
		if v := c.Query("radiusM"); v != "" {
			radius, _ = strconv.ParseFloat(v, 64)
		}
		if radius <= 0 || radius > 50000 {
			respondErr(c, user.ErrInvalidInput)
			return
		}
		hit, err := a.User.NearestAddress(c.Request.Context(), lat, lng, radius)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, hit)
	}
}

func addrCreateAddress(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body addrCreateBody
		if !httpx.BindAndValidate(c, &body, func() error {
			return httpx.CollectErrors(
				httpx.RequireString(body.Label, "label", 32),
				httpx.RequireString(body.Name, "name", 128),
			)
		}) {
			return
		}
		if !addrGeoAnchorOK(c, a, body.CountryCode, body.AdminCode) {
			return
		}
		id, ok2 := addrCreateAndAudit(c, a, &body)
		if !ok2 {
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"id": id})
	}
}

// addrCreateBody 建址请求体。
type addrCreateBody struct {
	ParentID    int64  `json:"parentId"`
	Label       string `json:"label" binding:"required"`
	Name        string `json:"name" binding:"required"`
	CountryCode string `json:"countryCode"`
	AdminCode   string `json:"adminCode"`
}

// addrGeoAnchorOK 地理锚点校验(二者任一非空即校验);失败已回写响应。
func addrGeoAnchorOK(c *gin.Context, a *app.Application, countryCode, adminCode string) bool {
	if countryCode == "" && adminCode == "" {
		return true
	}
	if err := validateGeoAnchor(c.Request.Context(), a, countryCode, adminCode); err != nil {
		respondErr(c, err)
		return false
	}
	return true
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
		if err := a.User.RecordImportTask(c.Request.Context(), "addresses", httpx.ClaimsAccountID(c), int(imported), int(imported), 0, 0, nil, ""); err != nil {
			respondErr(c, err)
			return
		}
		emitTask(c.Request.Context(), a, refImporter, "addr-"+fmt.Sprint(time.Now().Unix()),
			"地址导入完成:"+strconv.Itoa(int(imported))+" 行", linkImporter, false)
		respond(c, apitypes.CodeOK, gin.H{"imported": imported})
	}
}

// addrCreateAndAudit 建址 + 审计留痕;失败已回写响应。
func addrCreateAndAudit(c *gin.Context, a *app.Application, body *addrCreateBody) (int64, bool) {
	id, err := a.User.CreateAddress(c.Request.Context(),
		body.ParentID, body.Label, body.Name, body.CountryCode, body.AdminCode)
	if err != nil {
		respondErr(c, err)
		return 0, false
	}
	httpx.RecordAudit(a, c, "address.create", "addresses", strconv.FormatInt(id, 10), map[string]any{
		"parentId": body.ParentID, "label": body.Label, "name": body.Name,
	})
	return id, true
}
