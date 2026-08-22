// 招商引资/合作入驻域路由注册(迁移 000098)。
// 公开提交端点挂 api 根组(与 /auth/login 同级,免登录);审核与企业工作台走鉴权组。
package adminapi

import (
	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
)

// registerPartnerPublicRoutes 公开端点:企业自助提交入驻申请(免登录)。
func registerPartnerPublicRoutes(api *gin.RouterGroup, a *app.Application) {
	api.POST("/partner/applications", partnerSubmitHandler(a))
}

// registerPartnerRoutes 鉴权端点:后台审核(menu:partner)+ 企业工作台(partner_*)。
func registerPartnerRoutes(g *gin.RouterGroup, a *app.Application) {
	g.GET("/partner/applications", requirePerm(a.User, "menu:partner"), partnerListHandler(a))
	g.POST("/partner/applications/:id/approve", requirePerm(a.User, "menu:partner"), partnerApproveHandler(a))
	g.POST("/partner/applications/:id/reject", requirePerm(a.User, "menu:partner"), partnerRejectHandler(a))

	g.GET("/partner/me", requirePerm(a.User, "menu:partner-home"), partnerProfileHandler(a))
	g.GET("/partner/staff", requirePerm(a.User, "menu:partner-staff"), partnerStaffListHandler(a))
	g.POST("/partner/staff", requirePerm(a.User, "menu:partner-staff"), partnerStaffCreateHandler(a))
	g.PUT("/partner/staff/:id/status", requirePerm(a.User, "menu:partner-staff"), partnerStaffStatusHandler(a))
	g.GET("/partner/orders", requirePerm(a.User, "menu:partner-orders"), partnerOrdersHandler(a))
}

// partnerSubmitReq 入驻申请公开提交请求体。
type partnerSubmitReq struct {
	CompanyName  string `json:"companyName" binding:"required"`
	CreditCode   string `json:"creditCode" binding:"required"`
	ContactName  string `json:"contactName" binding:"required"`
	ContactPhone string `json:"contactPhone" binding:"required"`
	Email        string `json:"email"`
	BusinessDesc string `json:"businessDesc" binding:"required"`
}

// partnerStaffCreateReq 企业管理员新建员工请求体。
type partnerStaffCreateReq struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required,min=6"`
	RealName string `json:"realName" binding:"required"`
	Phone    string `json:"phone"`
}

// partnerStaffStatusReq 员工启用/停用请求体。
type partnerStaffStatusReq struct {
	Status int16 `json:"status" binding:"oneof=0 1"` // required 会拒 0(停用),只用 oneof
}
