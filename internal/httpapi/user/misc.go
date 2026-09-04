package userapi

// 用户端门户 Misc 域:首页聚合/消息中心/优惠券/用量/自助排障/协议。

import (
	"errors"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/promotion"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// ---- Misc 域 ----

// registerPortalMiscRoutes Misc 域:首页聚合/消息/优惠券/用量/自助排障/协议。
// 家庭地址路由见 registerPortalAddressRoutes,本函数保持单文件 ≤300 行。
func registerPortalMiscRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/home", portalHome(a))
	g.GET("/messages", portalListMessages(a))
	g.POST("/messages/read-all", portalReadAllMessages(a))
	g.PUT("/messages/:messageId/read", portalReadOneMessage(a))
	// 用户端 /coupons 仅挂在 /api/user/v1(与 admin /api/admin/v1 前缀隔离,无路由冲突)。
	g.GET("/coupons", portalListCoupons(a))
	g.POST("/coupons/redeem", portalRedeemCoupon(a))
	g.POST("/coupons/:couponId/gift", portalGiftCoupon(a))
	registerPortalAddressRoutes(g, a)
	g.GET("/usage", portalUsage)
	g.GET("/diy/steps", portalDiySteps)
	g.GET("/agreement", portalAgreement)
}

// portalDiySteps 自助排障静态引导(契约 DiySection)。
func portalDiySteps(c *gin.Context) {
	respond(c, apitypes.CodeOK, gin.H{"items": portalDiySections})
}

// portalAgreement 协议占位(用户协议 + 隐私政策三段目录)。
func portalAgreement(c *gin.Context) {
	respond(c, apitypes.CodeOK, gin.H{
		"userAgreement": []string{"服务条款", "费用与账期", "终止与违约"},
		"privacyPolicy": []string{"信息收集范围", "使用与共享", "保存期限与删除"},
	})
}

// portalListCoupons GET /coupons?status=&billCents=:我的优惠券。
// billCents>0 时按账单金额过滤门槛并附预估抵扣 estDeduct(分)。
func portalListCoupons(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		status := c.DefaultQuery("status", "available")
		billCents, _ := strconv.ParseInt(c.Query("billCents"), 10, 64)
		if a.Promotion == nil {
			respond(c, apitypes.CodeInternal, gin.H{"error": "promotion not configured"})
			return
		}
		items, err := a.Promotion.ListCustomerCoupons(c.Request.Context(), cid, status, billCents)
		if err != nil {
			respondErr(c, err)
			return
		}
		invite := portalListCouponsInvite(c, a)
		respond(c, apitypes.CodeOK, gin.H{"items": items, "inviteLink": invite})
	}
}

// portalRedeemCoupon POST /coupons/redeem {code}:兑换码领券 / 接收转赠券。
func portalRedeemCoupon(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var req struct {
			Code string `json:"code" binding:"required"`
		}
		if !httpx.BindBody(c, &req) {
			return
		}
		couponID, err := a.Promotion.RedeemCode(c.Request.Context(), req.Code, cid)
		if err != nil {
			// promotion 域错误在本 handler 边界收口映射(不动共享 httpx/error.go,
			// 避免与并行修复会话 A 的文件域冲突):明确业务码替代一律 50000。
			switch {
			case errors.Is(err, promotion.ErrCodeNotFound):
				respond(c, apitypes.CodeNotFound, gin.H{"reason": "兑换码不存在"})
			case errors.Is(err, promotion.ErrCodeUsedOrDisabled):
				respond(c, apitypes.CodeConflict, gin.H{"reason": "兑换码已被使用或已停用"})
			case errors.Is(err, promotion.ErrTemplateDisabled):
				respond(c, apitypes.CodeConflict, gin.H{"reason": "券模板已停用"})
			case errors.Is(err, promotion.ErrConflict):
				respond(c, apitypes.CodeConflict, gin.H{"reason": promotion.ConflictReason(err)})
			default:
				respondErr(c, err)
			}
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"couponId": couponID})
	}
}

// portalGiftCoupon POST /coupons/:couponId/gift:整券转赠,返回一次性转赠码。
func portalGiftCoupon(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		code, err := a.Promotion.CreateGift(c.Request.Context(), c.Param("couponId"), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"giftCode": code})
	}
}

// portalListCouponsInvite 邀请链接(配置空时返回空串)。
func portalListCouponsInvite(c *gin.Context, a *app.Application) string {
	cfg, err := a.UserData.GetInviteConfig(c.Request.Context())
	if err != nil || len(cfg) == 0 {
		return ""
	}
	return toStr(cfg[0]["inviteLink"])
}

// portalListAddresses GET /addresses?page=&pageSize= 已迁出至 portal_address.go。

func portalListMessages(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		cat := c.DefaultQuery("category", "all")
		msgs, err := a.Portal.Messages(c.Request.Context(), cid)
		if err != nil {
			respondErr(c, err)
			return
		}
		items := make([]gin.H, 0)
		for _, m := range msgs {
			if cat == "all" || m.Payload["category"] == cat {
				item := gin.H{
					"messageId": strconv.FormatInt(m.ID, 10),
					"read":      m.Read,
					"createdAt": m.CreatedAt,
				}
				for key, value := range m.Payload {
					item[key] = value
				}
				items = append(items, item)
			}
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

func portalReadAllMessages(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		if err := a.Portal.MarkAllRead(c.Request.Context(), cid); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// portalReadOneMessage 单条消息标为已读。messageId 不存在或不属于当前客户 → 404,
// 前端乐观更新时据此区分"服务端已确认"与"未知消息"。
func portalReadOneMessage(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		messageID := c.Param("messageId")
		if messageID == "" {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "messageId 必填"})
			return
		}
		found, err := a.Portal.MarkRead(c.Request.Context(), cid, messageID)
		if err != nil {
			respondErr(c, err)
			return
		}
		if !found {
			respond(c, apitypes.CodeNotFound, gin.H{"error": "消息不存在"})
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

func portalUsage(c *gin.Context) {
	period := c.DefaultQuery("period", "current")
	quota, used := 1024.0, 356.8
	respond(c, apitypes.CodeOK, gin.H{
		"periodLabel": map[string]string{"current": "本账期", "last": "上一账期", "six_month": "近 6 月"}[period],
		"used":        used, "unit": "GB", "quota": quota, "percent": int(used / quota * 100),
		"dailyAvg": 11.9, "forecastRemain": 667.2,
		"detail": gin.H{"down": 300.2, "up": 56.6, "iptvNote": "IPTV 用量不计入宽带配额"},
	})
}

// portalDiySections 自助排障静态引导(契约 DiySection)。
var portalDiySections = []gin.H{
	{"id": "power", "title": "设备检查", "desc": "先确认光猫与路由器电源",
		"steps": []gin.H{{"title": "查看指示灯", "desc": "LOS 红灯常亮表示线路故障"}}},
	{"id": "wifi", "title": "Wi-Fi 检查", "desc": "排除无线干扰",
		"steps": []gin.H{{"title": "靠近路由器测试", "desc": "排除距离与穿墙衰减"}}},
	{"id": "cable", "title": "线路检查", "desc": "确认入户光纤与网线",
		"steps": []gin.H{{"title": "重新插拔光纤接头", "desc": "注意接头清洁"}}},
}
