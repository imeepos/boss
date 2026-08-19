package userapi

// 用户端门户产品目录域:产品套餐列表 + 增值服务(available/subscribed)。

import (
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// portalListProducts GET /products?category=:产品套餐列表(items + addons)。
func portalListProducts(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		_ = c.Query("category")
		list, err := a.Product.ListProducts(c.Request.Context(), 0)
		if err != nil {
			respondErr(c, err)
			return
		}
		items := make([]gin.H, 0, len(list))
		for _, p := range list {
			if p.Status != "PUBLISHED" {
				continue
			}
			items = append(items, gin.H{
				"productId": strconv.FormatInt(p.ID, 10), "category": "broadband",
				"name": p.Name, "bandwidth": p.Bandwidth, "monthlyFee": p.MonthlyFee,
				"description": "", "contractMonths": 12, "featured": false,
			})
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items, "addons": []any{}})
	}
}

// portalListAddons GET /addons:增值服务(available + subscribed)。
func portalListAddons(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		items, err := a.UserData.ListAddons(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		subs, err := a.UserData.ListAddonSubscriptions(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		subscribed := make(map[string]bool)
		for _, s := range subs {
			if toInt64(s["customerId"]) == cid && toStr(s["action"]) == "subscribe" {
				subscribed[toStr(s["addonId"])] = true
			}
		}
		available, owned := make([]gin.H, 0), make([]gin.H, 0)
		for _, row := range items {
			addon := gin.H{
				"addonId": toStr(row["addonId"]), "name": toStr(row["name"]),
				"monthlyFee":  float64(toInt64(row["price"])) / 100,
				"description": "", "subscribed": subscribed[toStr(row["addonId"])],
			}
			if toStr(row["status"]) == "off" {
				continue
			}
			if subscribed[toStr(row["addonId"])] {
				owned = append(owned, addon)
			} else {
				available = append(available, addon)
			}
		}
		respond(c, apitypes.CodeOK, gin.H{"available": available, "subscribed": owned})
	}
}