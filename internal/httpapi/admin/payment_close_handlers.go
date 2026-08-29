package adminapi

// 柜台日结与柜面收款风控 handler(纪要 2026-08-28-柜面现金收款)。

import (
	"context"
	"errors"
	"log"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/billing"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// cashSingleLimitParam cash 单笔限额参数键(biz_params,元;0/缺失=不拦)。
const cashSingleLimitParam = "payment.cash.singleLimit"

// paramGetter biz_params 窄口(对齐 reserve_timeout_loop paramLister 先例)。
type paramGetter interface {
	GetParam(ctx context.Context, key string) (string, error)
}

// enforceCashLimit cash 单笔限额硬拦(纪要 Battle A 收敛,郑凯最小方案):
// 阈值走 biz_params 热调,仅对 cash 生效、仅单笔维度;超限 42200 拒绝并写审计留痕。
// 配置缺失/非法/显式 0 降级不拦——限额缺失由审计与日结差异兜底,不阻塞正常收款。
func enforceCashLimit(c *gin.Context, a *app.Application, p billing.Payment) bool {
	if p.Method != "cash" || p.Amount <= 0 {
		return true
	}
	limit, err := cashLimitValue(c.Request.Context(), a)
	if err != nil {
		log.Printf("[payment-cash] LIMIT READ FAILED need-review param=%s err=%v", cashSingleLimitParam, err)
		return true
	}
	if limit <= 0 || p.Amount <= limit {
		return true
	}
	respond(c, apitypes.CodeInvalidParam, gin.H{
		"reason": "现金收款超过单笔限额 " + strconv.FormatFloat(limit, 'f', 2, 64) + " 元,请走复核或引导其他支付方式",
	})
	httpx.RecordAudit(a, c, "payment.cash.limit_reject", "payment", p.PayNo, gin.H{
		"amount": p.Amount, "limit": limit, "method": p.Method,
		"customerId": p.CustomerID, "siteName": p.SiteName, "operator": p.OperatorName,
	})
	return false
}

// cashLimitValue 读 cash 单笔限额(元)。
func cashLimitValue(ctx context.Context, a *app.Application) (float64, error) {
	pg, ok := a.User.(paramGetter)
	if !ok {
		return 0, errors.New("adminapi: user store has no param reader")
	}
	v, err := pg.GetParam(ctx, cashSingleLimitParam)
	if err != nil || v == "" {
		return 0, err
	}
	return strconv.ParseFloat(v, 64)
}

// dailyCashSummary GET /daily-closings/summary?date=:
// cash 流水按网点+操作员汇总,收入/退款分列(退款按流水发生日归属),附已回填实点。
func dailyCashSummary(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		date := c.Query("date")
		if date == "" {
			date = time.Now().Format("2006-01-02")
		}
		rows, err := a.Billing.DailyCashSummary(c.Request.Context(), date)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"date": date, "items": rows})
	}
}

// dailyCashItems GET /daily-closings/items?date=:当日 cash 流水逐笔(报表下钻,
// 含 REFUNDED 凭证及 refund_reason/refunded_at,周敏口径零新增字段)。
func dailyCashItems(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		date := c.Query("date")
		if date == "" {
			date = time.Now().Format("2006-01-02")
		}
		items, err := a.Billing.CashPaymentsByDate(c.Request.Context(), date)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"date": date, "items": items})
	}
}

// saveDailyClosing POST /daily-closings:实点金额当日回填(登记义务);
// 不平不阻塞回填,差异输出 [paycheck] DIFF 可 grep 日志附网点/操作员上下文。
// 写侧本人强约束(纪要待定项收口 2026-08-28,Item2 裁决):非 sysadmin 只能回填
// 自己名下(operatorName=登录名)的日结,防柜员篡改他柜账目;读侧放开给对账岗,
// 与 admin payments 全量回传口径一致。
func saveDailyClosing(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var cl billing.DailyClosing
		if !httpx.BindBody(c, &cl) {
			return
		}
		if cl.Date == "" {
			cl.Date = time.Now().Format("2006-01-02")
		}
		operator := httpx.ClaimsUsername(c)
		if cl.OperatorName == "" {
			cl.OperatorName = operator
		}
		if cl.SiteName == "" || cl.OperatorName == "" {
			respond(c, apitypes.CodeInvalidParam, gin.H{"reason": "网点与操作员为日结归因必填项"})
			return
		}
		if httpx.ClaimsRoleCode(c) != "sysadmin" && cl.OperatorName != operator {
			respond(c, apitypes.CodeInvalidParam, gin.H{"reason": "只能回填本人名下的柜台日结"})
			return
		}
		cl.CreatedBy = operator
		res, err := a.Billing.SaveDailyClosing(c.Request.Context(), cl)
		if err != nil {
			respondErr(c, err)
			return
		}
		if !res.Balanced {
			log.Printf("[paycheck] DIFF need-review date=%s site=%q operator=%q system=%.2f counted=%.2f diff=%.2f",
				cl.Date, cl.SiteName, cl.OperatorName, res.SystemAmount, cl.CountedAmount, res.DiffAmount)
		}
		respond(c, apitypes.CodeOK, gin.H{"closing": res})
	}
}
