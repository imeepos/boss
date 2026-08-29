package adminapi

// 开单内联建址端点(meeting-minutes/2026-08-29 §九):POST /orders/address。
// 门禁 menu:order——能开单就能建址,开单零阻塞;地址管理页(menu:address)仍是治理入口。

import (
	"fmt"
	"log"
	"strconv"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/notify"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// inlineAddressReq 内联建址请求:五级自上而下全必填,缺失层级由域层就地补建。
type inlineAddressReq struct {
	CustomerID       int64  `json:"customerId"`
	City             string `json:"city"`
	District         string `json:"district"`
	Street           string `json:"street"`
	Compound         string `json:"compound"`
	Building         string `json:"building"`
	BackfillCustomer bool   `json:"backfillCustomer"`
}

// toInlineInput 请求体 → 域层入参。
func (r inlineAddressReq) toInlineInput() user.InlineAddressInput {
	return user.InlineAddressInput{
		CustomerID: r.CustomerID,
		Levels: []user.InlineAddressLevel{
			{Level: 1, Name: r.City}, {Level: 2, Name: r.District}, {Level: 3, Name: r.Street},
			{Level: 4, Name: r.Compound}, {Level: 5, Name: r.Building},
		},
		BackfillCustomer: r.BackfillCustomer,
	}
}

// orderInlineAddressHandler 开单流程内联建址:单事务全链补建到楼栋级;
// 回执含归属推导供前端预览;祖先链无区域覆盖时留 ALERT + 归属修正 P1 待办。
func orderInlineAddressHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req inlineAddressReq
		if !httpx.BindAndValidate(c, &req, func() error {
			return httpx.CollectErrors(
				httpx.RequirePositiveID(req.CustomerID, "customerId"),
				httpx.RequireString(req.City, "city", 60),
				httpx.RequireString(req.District, "district", 60),
				httpx.RequireString(req.Street, "street", 60),
				httpx.RequireString(req.Compound, "compound", 60),
				httpx.RequireString(req.Building, "building", 60),
			)
		}) {
			return
		}
		res, err := a.User.CreateInlineAddressChain(c.Request.Context(), req.toInlineInput())
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "address.inline-create", "addresses",
			strconv.FormatInt(res.AddressID, 10), map[string]any{
				"customerId": req.CustomerID, "path": res.FullPath, "fallback": res.Fallback,
			})
		reportFallback(a, c, req.CustomerID, res)
		respond(c, apitypes.CodeOK, inlineAddressResp(res))
	}
}

// reportFallback 兜底留痕:可 grep ALERT 日志 + 归属修正待办(幂等,按地址 path)。
// 订单已按零阻塞放行(000077 兜底归属),修正走待办闭环,不阻断开单。
func reportFallback(a *app.Application, c *gin.Context, customerID int64, res user.InlineAddressResult) {
	if !res.Fallback {
		return
	}
	log.Printf("[order-ownership] REGION UNCOVERED ALERT path=%s customerId=%d fallbackEntity=%d operator=%d",
		res.FullPath, customerID, res.LegalEntityID, httpx.ClaimsAccountID(c))
	if a.Notify == nil {
		return
	}
	err := a.Notify.Emit(c.Request.Context(), notify.Input{
		Category: notify.CategoryTodo,
		Level:    notify.LevelUrgent,
		DueHours: 4, // P1 处理时限,对齐 000115 待办 SLA 口径
		Title:    "内联建址归属待修正",
		Content: fmt.Sprintf("地址 %s 祖先链无经营区域覆盖,订单兜底平台总公司;请挂接区域并核对归属(P1)",
			res.FullPathNames),
		Link:    "/base/address",
		RefType: "order-inline-addr",
		RefID:   res.FullPath,
	})
	if err != nil {
		log.Printf("[order-ownership] TODO EMIT FAILED path=%s err=%v", res.FullPath, err)
	}
}

// inlineAddressResp 回执 → 响应体(契约见 fields.md §1.5.0b)。
func inlineAddressResp(res user.InlineAddressResult) gin.H {
	return gin.H{
		"addressId":     res.AddressID,
		"fullPath":      res.FullPath,
		"fullPathNames": res.FullPathNames,
		"legalEntityId": res.LegalEntityID,
		"regionPath":    res.RegionPath,
		"fallback":      res.Fallback,
		"needsReview":   res.NeedsReview,
		"backfilled":    res.Backfilled,
	}
}
