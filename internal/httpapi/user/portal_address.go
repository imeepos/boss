package userapi

// 用户端家庭地址端点:list/create/update/delete + 共用 addressInfoOf 转换器。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	udcustomer "github.com/ymm-001/boss/internal/domain/customer/userdata"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerPortalAddressRoutes /addresses 系列:分页 list + create + update + delete。
// 地址层级树(/address-tree)见 portal_address_tree.go,在此统一挂载。
func registerPortalAddressRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/addresses", portalListAddresses(a))
	g.POST("/addresses", portalCreateAddress(a))
	g.PUT("/addresses/:addressId", portalUpdateAddress(a))
	g.DELETE("/addresses/:addressId", portalDeleteAddress(a))
	registerPortalAddressTreeRoutes(g, a)
}

// portalListAddresses GET /addresses?page=&pageSize= 我的家庭地址分页(契约 AddressInfo)。
// 仅返回归属当前客户的地址;pageSize 缺省 20,最大 100;响应附 page/pageSize/hasMore/total。
func portalListAddresses(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
		if page < 1 {
			page = 1
		}
		if pageSize < 1 || pageSize > 100 {
			pageSize = 20
		}
		rows, err := a.UserData.ListUserAddresses(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		mine := make([]map[string]any, 0, len(rows))
		for _, r := range rows {
			if toInt64(r["customerId"]) == cid {
				mine = append(mine, r)
			}
		}
		total := len(mine)
		start := (page - 1) * pageSize
		if start > total {
			start = total
		}
		end := start + pageSize
		if end > total {
			end = total
		}
		items := make([]gin.H, 0, end-start)
		for _, r := range mine[start:end] {
			items = append(items, addressInfoOf(r))
		}
		respond(c, apitypes.CodeOK, gin.H{
			"items":    items,
			"page":     page,
			"pageSize": pageSize,
			"total":    total,
			"hasMore":  end < total,
		})
	}
}

// portalCreateAddress POST /addresses:新增家庭地址(user_addresses 落库)。
func portalCreateAddress(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req struct {
			Community string `json:"community" binding:"required"`
			Building  string `json:"building"`
			Door      string `json:"door"`
			Contact   string `json:"contact" binding:"required"`
			Phone     string `json:"phone"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		phone := req.Phone
		if phone == "" {
			if cust, err := a.Customer.Get(c.Request.Context(), cid); err == nil {
				phone = cust.Phone
			}
		}
		if _, err := a.UserData.CreateUserAddress(c.Request.Context(), udcustomer.UserAddress{
			CustomerID: cid, AddrCode: req.Community,
			Contact: req.Contact, Phone: phone,
			Detail:    req.Community + " " + req.Building + " " + req.Door,
			IsDefault: false,
		}); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// portalUpdateAddress PUT /addresses/:addressId 修改家庭地址。
// 归属校验由 domain 层(WHERE customer_id=$2);未命中返回 404。
func portalUpdateAddress(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		id, ok := httpx.ParsePathParamInt64(c, "addressId")
		if !ok {
			return
		}
		var req struct {
			Community string `json:"community" binding:"required"`
			Building  string `json:"building"`
			Door      string `json:"door"`
			Contact   string `json:"contact" binding:"required"`
			Phone     string `json:"phone"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		phone := req.Phone
		if phone == "" {
			if cust, err := a.Customer.Get(c.Request.Context(), cid); err == nil {
				phone = cust.Phone
			}
		}
		err := a.UserData.UpdateUserAddress(c.Request.Context(), cid, id, udcustomer.UserAddress{
			CustomerID: cid, AddrCode: req.Community,
			Contact: req.Contact, Phone: phone,
			Detail:    req.Community + " " + req.Building + " " + req.Door,
			IsDefault: false,
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// portalDeleteAddress DELETE /addresses/:addressId 删除家庭地址。
func portalDeleteAddress(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		id, ok := httpx.ParsePathParamInt64(c, "addressId")
		if !ok {
			return
		}
		if err := a.UserData.DeleteUserAddress(c.Request.Context(), cid, id); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// addressInfoOf user_addresses 行转契约 AddressInfo;portal_list 与 portal_update 共用。
func addressInfoOf(r map[string]any) gin.H {
	return gin.H{
		"addressId":   toStr(r["id"]),
		"label":       toStr(r["detail"]),
		"isDefault":   r["isDefault"],
		"contact":     toStr(r["contact"]),
		"phoneMasked": httpx.MaskPhone(toStr(r["phone"])),
		"community":   toStr(r["addrCode"]),
		"building":    "",
		"door":        "",
	}
}
