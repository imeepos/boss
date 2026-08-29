package adminapi

// 代客受理目录:开户/下单表单的选项数据源。
// 门禁跟随受理域自身权限(menu:customer / menu:order),不要求操作员持有
// 组织(menu:region/menu:company)或资源(menu:provision)域菜单——
// ops 受理角色凭本域权限即可走完代客开户+下单全程(后台代客闭环目标)。
// 渠道目录复用 ChannelService.ListChannels,产品目录复用 ProductService.ListProducts,
// 地址搜索复用 user.Service.SearchAddresses,不重复造域口。

import (
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerOnbehalfCatalogRoutes 代客受理目录路由。
func registerOnbehalfCatalogRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/orders/catalog", requirePerm(a.User, "menu:order"), orderCatalogHandler(a))
	g.GET("/customers/onboarding-catalog", requirePerm(a.User, "menu:customer"), customerCatalogHandler(a))
}

// catalogAddress 地址选择项(命中节点 + 祖先链拍平为全路径名)。
type catalogAddress struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	FullPath string `json:"fullPath"`
}

// catalogAddressHits SearchAddresses 命中 → 选择器选项。
func catalogAddressHits(hits []user.AddressHit) []catalogAddress {
	out := make([]catalogAddress, 0, len(hits))
	for _, h := range hits {
		parts := make([]string, 0, len(h.Ancestors)+1)
		for _, anc := range h.Ancestors {
			parts = append(parts, anc.Name)
		}
		parts = append(parts, h.Node.Name)
		out = append(out, catalogAddress{ID: h.Node.ID, Name: h.Node.Name, FullPath: strings.Join(parts, " / ")})
	}
	return out
}

// appendAddressHits q 非空时把地址搜索结果并入 resp;出错原样返回。
func appendAddressHits(a *app.Application, c *gin.Context, resp gin.H) error {
	q := c.Query("q")
	if q == "" {
		return nil
	}
	hits, _, err := a.User.SearchAddresses(c.Request.Context(), q)
	if err != nil {
		return err
	}
	resp["addresses"] = catalogAddressHits(hits)
	return nil
}

// orderCatalogHandler GET /orders/catalog:代客下单选项目录(在售产品+渠道;带 q 附地址搜索)。
// 产品只回 PUBLISHED(下单校验拒绝非在售,提前过滤免得操作员选了被拒)。
func orderCatalogHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		products, err := a.Product.ListProducts(c.Request.Context(), 0)
		if err != nil {
			respondErr(c, err)
			return
		}
		onSale := make([]customer.ProductOffer, 0, len(products))
		for _, p := range products {
			if p.Status == "PUBLISHED" {
				onSale = append(onSale, p)
			}
		}
		channels, err := a.Channel.ListChannels(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		resp := gin.H{"products": onSale, "channels": channels}
		if err := appendAddressHits(a, c, resp); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, resp)
	}
}

// customerCatalogHandler GET /customers/onboarding-catalog:代客开户选项目录
// (经营区域+运营主体;带 q 附地址搜索)。
func customerCatalogHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		regions, err := a.User.ListRegions(c.Request.Context(), "")
		if err != nil {
			respondErr(c, err)
			return
		}
		entities, err := a.User.ListLegalEntities(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		resp := gin.H{"regions": regions, "legalEntities": entities}
		if err := appendAddressHits(a, c, resp); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, resp)
	}
}
