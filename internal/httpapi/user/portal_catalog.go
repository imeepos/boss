package userapi

// 用户端门户产品目录域:产品套餐列表 + 增值服务(available/subscribed)。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// portalListProducts GET /products?category=:产品套餐列表(items + addons)。
// category 为空返回全部;broadband/fusion 只返回对应分类,addon 分类下 items 为空(增值服务走 addons 数组)。
func portalListProducts(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		category := c.Query("category")
		list, err := a.Product.ListProducts(c.Request.Context(), 0)
		if err != nil {
			respondErr(c, err)
			return
		}
		items := make([]gin.H, 0, len(list))
		for _, p := range list {
			if p.Status != "PUBLISHED" || (category != "" && category != "addon" && p.Category != category) {
				continue
			}
			items = append(items, gin.H{
				"productId": strconv.FormatInt(p.ID, 10), "category": p.Category,
				"name": p.Name, "bandwidth": p.Bandwidth, "monthlyFee": p.MonthlyFee,
				"description": "", "contractMonths": 12, "featured": false,
			})
		}
		addons, err := portalAddonList(a, c)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items, "addons": addons})
	}
}

// portalListAddons GET /addons:增值服务(available + subscribed)。
func portalListAddons(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		views, err := portalAddonViews(a, c)
		if err != nil {
			respondErr(c, err)
			return
		}
		available, owned := make([]gin.H, 0), make([]gin.H, 0)
		for _, v := range views {
			if v["subscribed"].(bool) {
				owned = append(owned, v)
			} else {
				available = append(available, v)
			}
		}
		respond(c, apitypes.CodeOK, gin.H{"available": available, "subscribed": owned})
	}
}

// portalAddonList /products 响应里的 addons 数组:在售目录(含已订购标记)。
func portalAddonList(a *app.Application, c *gin.Context) ([]gin.H, error) {
	views, err := portalAddonViews(a, c)
	if err != nil {
		return nil, err
	}
	return views, nil
}

// portalAddonViews 增值服务目录 → 契约 Addon 视图(分单位转元,off 下架剔除)。
func portalAddonViews(a *app.Application, c *gin.Context) ([]gin.H, error) {
	cid, _ := requireCustomer(c)
	items, err := a.UserData.ListAddons(c.Request.Context())
	if err != nil {
		return nil, err
	}
	subs, err := a.UserData.ListAddonSubscriptions(c.Request.Context())
	if err != nil {
		return nil, err
	}
	subscribed := make(map[string]bool)
	for _, s := range subs {
		if toInt64(s["customerId"]) == cid && toStr(s["action"]) == "subscribe" {
			subscribed[toStr(s["addonId"])] = true
		}
	}
	out := make([]gin.H, 0, len(items))
	for _, row := range items {
		if toStr(row["status"]) == "off" {
			continue
		}
		out = append(out, gin.H{
			"addonId": toStr(row["addonId"]), "name": toStr(row["name"]),
			"monthlyFee":  float64(toInt64(row["price"])) / 100,
			"description": "", "subscribed": subscribed[toStr(row["addonId"])],
		})
	}
	return out, nil
}
