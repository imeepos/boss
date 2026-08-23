package userapi

// 用户端门户 Profile 域:实名认证(/auth/verify) + 我的/账号安全/通知订阅/语言。

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/customer"
)

// portalCustomerPhone 取客户手机号:优先 customers 主档,合成客户(隔离空间,无 customers 主档)回退 portal_accounts。
func portalCustomerPhone(ctx context.Context, a *app.Application, cid int64) string {
	if v, err := a.Customer.Get(ctx, cid); err == nil {
		return v.Phone
	}
	if acc, err := a.Portal.AccountByCustomer(ctx, cid); err == nil {
		return acc.Phone
	}
	return ""
}

// portalMaskPhone 138****1234;portalMaskName 王**;portalMaskIDNo 首3尾4。
func portalMaskPhone(s string) string {
	if len(s) < 7 {
		return s
	}
	return s[:3] + "****" + s[len(s)-4:]
}
func portalMaskName(s string) string {
	// 按 rune 取首字:s[:1] 字节切片会把中文首字切成非法 UTF-8,JSON 序列化成 U+FFFD。
	r := []rune(s)
	if len(r) == 0 {
		return ""
	}
	return string(r[0]) + "**"
}
func portalMaskIDNo(s string) string {
	if len(s) < 8 {
		return s
	}
	return s[:3] + strings.Repeat("*", len(s)-7) + s[len(s)-4:]
}

// portalProfilePlan /profile 的 plan 聚合:当前套餐(user_plans + 产品价)
// + 本月账单(最近未缴账单;无未缴返回 0 与空串,端上显示"—")。
func portalProfilePlan(a *app.Application, c *gin.Context, cid int64) gin.H {
	plan := gin.H{"planId": "", "name": "", "monthlyFee": 0, "contractEnd": "",
		"status": "NONE", "installAddress": "", "currentBillAmount": 0, "currentBillDue": ""}
	if row, ok := portalCurrentPlanRow(a, c, cid); ok {
		pid := toInt64(row["productId"])
		plan["planId"] = fmt.Sprintf("%d", pid)
		plan["name"] = toStr(row["planName"])
		plan["status"] = toStr(row["status"])
		if p := portalProductByID(a, c, pid); p != nil {
			plan["monthlyFee"] = p.MonthlyFee
		}
		plan["installAddress"] = portalDefaultAddress(a, c, cid)
	}
	if bills, err := a.Billing.ListBills(c.Request.Context(), cid); err == nil {
		for _, b := range bills {
			if b.Status != "PAID" && b.Period >= toStr(plan["currentBillDue"]) {
				plan["currentBillAmount"] = b.Amount
				plan["currentBillDue"] = b.Period
			}
		}
	}
	return plan
}

// portalCurrentPlanRow 客户当前套餐行(user_plans 最近一条)。
func portalCurrentPlanRow(a *app.Application, c *gin.Context, cid int64) (map[string]any, bool) {
	if a.UserData == nil {
		return nil, false
	}
	rows, err := a.UserData.ListUserPlans(c.Request.Context())
	if err != nil {
		return nil, false
	}
	for _, r := range rows {
		if toInt64(r["customerId"]) == cid {
			return r, true
		}
	}
	return nil, false
}

// portalDefaultAddress 默认地址明细(user_addresses is_default)。
func portalDefaultAddress(a *app.Application, c *gin.Context, cid int64) string {
	if a.UserData == nil {
		return ""
	}
	rows, err := a.UserData.ListUserAddresses(c.Request.Context())
	if err != nil {
		return ""
	}
	for _, r := range rows {
		if toInt64(r["customerId"]) == cid && toBool(r["isDefault"]) {
			return toStr(r["detail"])
		}
	}
	return ""
}

// portalProductByID 产品目录按 ID 寻址(目录未接入返回 nil)。
func portalProductByID(a *app.Application, c *gin.Context, id int64) *customer.ProductOffer {
	if a.Product == nil {
		return nil
	}
	list, err := a.Product.ListProducts(c.Request.Context(), 0)
	if err != nil {
		return nil
	}
	for i := range list {
		if list[i].ID == id {
			return &list[i]
		}
	}
	return nil
}
