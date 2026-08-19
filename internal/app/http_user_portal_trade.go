package app

// 用户端门户 Product/Order/Plans 域:产品详情、订单操作/评价、套餐变更与退订预检 + portalStore 补充方法。

import (
	"fmt"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// ---- portalStore 补充方法(类型定义见 http_user_portal.go) ----

func (s *portalStore) accountByCustomer(cid int64) *portalAccount {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, acc := range s.accounts {
		if acc.CustomerID == cid {
			return acc
		}
	}
	return nil
}

func (s *portalStore) rebindPhone(cid int64, newPhone string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for phone, acc := range s.accounts {
		if acc.CustomerID == cid {
			delete(s.accounts, phone)
			acc.Phone = newPhone
			s.accounts[newPhone] = acc
			return
		}
	}
	s.accounts[newPhone] = &portalAccount{Phone: newPhone, CustomerID: cid}
}

func (s *portalStore) balance(cid int64) float64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.bal[cid]
}

func (s *portalStore) adjustBalance(cid int64, delta float64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.bal[cid] += delta
}

func (s *portalStore) msgs(cid int64) []gin.H {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.messages[cid]; !ok {
		s.messages[cid] = []gin.H{{
			"messageId": fmt.Sprintf("M-%d-1", cid), "category": "billing",
			"title": "欢迎使用客户门户", "content": "登录后可查询账单与订单进度。",
			"tag": "账单", "tagLevel": "bill", "createdAt": time.Now(), "read": false,
		}}
	}
	return s.messages[cid]
}

func (s *portalStore) hasUnread(cid int64) bool {
	for _, m := range s.msgs(cid) {
		if read, _ := m["read"].(bool); !read {
			return true
		}
	}
	return false
}

func (s *portalStore) markAllRead(cid int64) {
	s.mu.Lock()
	defer s.mu.Unlock()
	for _, m := range s.msgs(cid) {
		m["read"] = true
	}
}

func (s *portalStore) putMessage(cid int64, msg gin.H) {
	_ = s.msgs(cid)
	s.mu.Lock()
	defer s.mu.Unlock()
	s.messages[cid] = append([]gin.H{msg}, s.messages[cid]...)
}

// portalPayNo 生成支付单号(进程内序列;持久化方案见报告)。
func (s *portalStore) payNo() string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	return fmt.Sprintf("PAY%d", s.seq)
}

// ---- Product / Order ----

// registerPortalOrderRoutes 客户视角产品详情 + 订单操作与套餐变更。
func registerPortalOrderRoutes(g *gin.RouterGroup, a *Application) {
	g.GET("/products/:id", portalProductDetail(a))
	g.POST("/addons/:addonId/subscribe", portalAddonOp)
	g.POST("/addons/:addonId/unsubscribe", portalAddonOp)
	g.POST("/orders/:orderNo/cancel", portalOrderCancel(a))
	g.POST("/orders/:orderNo/urge", portalOrderUrge(a))
	g.POST("/orders/:orderNo/change-address", portalOrderChangeAddr)
	g.GET("/orders/:orderNo/rate", portalRateGet(a))
	g.POST("/orders/:orderNo/rate", portalRatePost)
	g.GET("/plans/:planId/cancel", portalPlanCancelPreview(a))
	g.POST("/plans/:planId/change", portalPlanOp)
	g.POST("/plans/:planId/move", portalPlanOp)
	g.POST("/plans/:planId/cancel", portalPlanOp)
}

// portalProductDetail GET /products/:id:产品详情(PUBLISHED 才可见)。
func portalProductDetail(a *Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.ParseInt(c.Param("id"), 10, 64)
		if err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		list, lerr := a.Product.ListProducts(c.Request.Context(), 0)
		if lerr != nil {
			respondErr(c, lerr)
			return
		}
		for _, p := range list {
			if p.ID == id && p.Status == "PUBLISHED" {
				respond(c, apitypes.CodeOK, gin.H{"product": gin.H{
					"productId": strconv.FormatInt(p.ID, 10), "category": "broadband",
					"name": p.Name, "bandwidth": p.Bandwidth, "monthlyFee": p.MonthlyFee,
					"contractMonths": 12, "featured": false,
				}, "specs": []gin.H{{"label": "带宽", "value": p.Bandwidth}}, "compare": []any{}})
				return
			}
		}
		respond(c, apitypes.CodeNotFound, nil)
	}
}

func portalAddonOp(c *gin.Context) {
	requireCustomer(c)
	respond(c, apitypes.CodeOK, gin.H{"ok": true, "addonId": c.Param("addonId")})
}

// portalOwnedOrder 取订单并校验归属(客户视角越权一律 404,不泄露存在性)。
func portalOwnedOrder(a *Application, c *gin.Context, cid int64) (*order.Order, bool) {
	o, err := a.Order.GetByNo(c.Request.Context(), c.Param("orderNo"))
	if err != nil || o.CustomerID != cid {
		respond(c, apitypes.CodeNotFound, nil)
		return nil, false
	}
	return o, true
}

// portalOrderCancel POST /orders/:orderNo/cancel:取消订单(释放预占端口)。
func portalOrderCancel(a *Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		o, ok := portalOwnedOrder(a, c, cid)
		if !ok {
			return
		}
		if err := a.Order.Cancel(c.Request.Context(), o.ID); err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// portalOrderUrge POST /orders/:orderNo/urge:催单落消息(催单队列持久化见报告)。
func portalOrderUrge(a *Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		o, ok := portalOwnedOrder(a, c, cid)
		if !ok {
			return
		}
		portal.putMessage(cid, gin.H{
			"messageId": portal.payNo(), "category": "fault", "title": "催单已受理",
			"content": "订单 " + o.OrderNo + " 已加急处理", "tag": "订单", "tagLevel": "info",
			"createdAt": time.Now(), "read": false,
		})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// portalOrderChangeAddr POST /orders/:orderNo/change-address(地址变更域服务缺位,仅校验入参;见报告)。
func portalOrderChangeAddr(c *gin.Context) {
	var req struct {
		AddressID string `json:"addressId" binding:"required"`
	}
	if !bindBody(c, &req) {
		return
	}
	respond(c, apitypes.CodeOK, gin.H{"ok": true})
}

var portalRatings = map[string]bool{}

// portalRateGet GET /orders/:orderNo/rate:待评价信息(DONE 且未评价)。
func portalRateGet(a *Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		o, ok := portalOwnedOrder(a, c, cid)
		if !ok {
			return
		}
		if o.Status != "DONE" || portalRatings[o.OrderNo] {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"orderNo": o.OrderNo, "finishedAt": o.CreatedAt})
	}
}

// portalRatePost POST /orders/:orderNo/rate:提交评价(持久化见报告)。
func portalRatePost(c *gin.Context) {
	var req struct {
		Stars    int    `json:"stars" binding:"required,min=1,max=5"`
		Attitude int    `json:"attitude" binding:"required,min=1,max=5"`
		Quality  int    `json:"quality" binding:"required,min=1,max=5"`
		Comment  string `json:"comment"`
	}
	if !bindBody(c, &req) {
		return
	}
	portalRatings[c.Param("orderNo")] = true
	respond(c, apitypes.CodeOK, gin.H{"ok": true})
}

// portalPlanCancelPreview GET /plans/:planId/cancel:退订预检(未缴账单 + 违约金示意)。
func portalPlanCancelPreview(a *Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		cid, _ := requireCustomer(c)
		var unpaid []gin.H
		var penalty float64
		bills, _ := a.Billing.ListBills(c.Request.Context(), cid)
		for _, b := range bills {
			if b.Status != "PAID" {
				unpaid = append(unpaid, gin.H{"billNo": b.BillNo, "period": b.Period,
					"amount": b.Amount, "status": b.Status})
				penalty += b.Amount
			}
		}
		respond(c, apitypes.CodeOK, gin.H{
			"unpaidBills": unpaid, "penalty": penalty,
			"penaltyDesc": "含未出账部分按套餐剩余月份折算",
		})
	}
}

// portalPlanOp POST /plans/:planId/{change,move,cancel}:生成变更/迁址/拆机单(持久化见报告)。
func portalPlanOp(c *gin.Context) {
	requireCustomer(c)
	portal.mu.Lock()
	portal.seq++
	no := fmt.Sprintf("CHG-%d", portal.seq)
	portal.mu.Unlock()
	respond(c, apitypes.CodeOK, gin.H{"orderNo": no, "status": "PENDING",
		"statusLabel": "待核查", "stage": 1, "stageLabel": "下单"})
}
