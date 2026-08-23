package adminapi

// 四码合一域路由 handler 实现(承接 registerQuadlinkRoutes)。

import (
	"fmt"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/quadlink"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// quadlinkCreateLinkHandler POST /quad-links:新建四码绑定(装维前预绑定:资产+客户+端口+地址,status=UNLINKED)。
func quadlinkCreateLinkHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var q quadlink.QuadLink
		if !httpx.BindAndValidate(c, &q, func() error {
			return httpx.CollectErrors(
				httpx.RequirePositiveID(q.AssetID, "assetId"),
				httpx.RequirePositiveID(q.CustomerID, "customerId"),
				httpx.RequirePositiveID(q.PortID, "portId"),
				httpx.RequirePositiveID(q.AddressID, "addressId"),
				httpx.RequirePositiveID(q.LegalEntityID, "legalEntityId"),
			)
		}) {
			return
		}
		if q.Status == "" {
			q.Status = "UNLINKED"
		}
		id, err := a.QuadLink.CreateLink(c.Request.Context(), q)
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "quad_link", fmt.Sprint(id), nil)
		respond(c, apitypes.CodeOK, gin.H{"id": id, "status": q.Status})
	}
}

// quadlinkListLinksHandler GET /quad-links:四码绑定列表。
func quadlinkListLinksHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.QuadLink.ListLinks(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": list})
	}
}

// quadlinkGetByAssetHandler GET /quad-links/by-asset:按资产反查。
func quadlinkGetByAssetHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		assetID, ok := requireID(c, "assetId")
		if !ok {
			return
		}
		q, err := a.QuadLink.GetByAsset(c.Request.Context(), assetID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, q)
	}
}

// quadlinkGetByCustomerHandler GET /quad-links/by-customer:按客户反查。
func quadlinkGetByCustomerHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		customerID, ok := requireID(c, "customerId")
		if !ok {
			return
		}
		q, err := a.QuadLink.GetByCustomer(c.Request.Context(), customerID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, q)
	}
}

// quadlinkGetByPortHandler GET /quad-links/by-port:按端口反查。
func quadlinkGetByPortHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		portID, ok := requireID(c, "portId")
		if !ok {
			return
		}
		q, err := a.QuadLink.GetByPort(c.Request.Context(), portID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, q)
	}
}

// quadlinkGetByAddressHandler GET /quad-links/by-address:按地址反查。
func quadlinkGetByAddressHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		addressID, ok := requireID(c, "addressId")
		if !ok {
			return
		}
		q, err := a.QuadLink.GetByAddress(c.Request.Context(), addressID)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, q)
	}
}
