// Package userdata 用户端数据域:承接 api/openapi/admin/userdata.yaml(用户主档聚合
// + 用户端专属实体表 + 全局配置),实体外键统一挂 customers.id。
package userdata

import "context"

// ErrNotFound 目标记录未命中(路由层映射 CodeNotFound)。
var ErrNotFound = &notFoundError{}

type notFoundError struct{}

func (*notFoundError) Error() string { return "userdata: not found" }

// ErrContractDrift 列表契约漂移:某行必填主键/列返回 nil 或空串。
// 兜住"前端假设字段名(id)与后端 SQL AS 别名(addonId/couponId/...)不一致"
// 这种静默失败——一旦发生立刻 500,前端看到提示而不是 undefined。
// 路由层映射 CodeInternal(2026-08-22 postmortem 0002 的纵深防御)。
var ErrContractDrift = &contractDriftError{}

type contractDriftError struct {
	op    string
	field string
}

func (e *contractDriftError) Error() string {
	if e.field == "" {
		return "userdata: contract drift"
	}
	return "userdata: contract drift: " + e.op + " 缺关键列 " + e.field
}

// Is 允许 errors.Is 链路识别。
func (e *contractDriftError) Is(target error) bool {
	_, ok := target.(*contractDriftError)
	return ok
}

// AssertListContract 列表契约守护:items 非空时每行必须包含 pkField 且值非零值
// (nil/空串/0)。命中漂移返回 ErrContractDrift;空列表直接放行(无数据≠契约漂移)。
// 调用点必须按后端 SQL AS 别名填写 pkField,如 "addonId"/"couponId"/"denomId"。
func AssertListContract(op, pkField string, items []map[string]any) error {
	if len(items) == 0 {
		return nil
	}
	for i, row := range items {
		v, ok := row[pkField]
		if !ok || v == nil {
			return &contractDriftError{op: op, field: pkField + "(missing in row#" + strconvI(i) + ")"}
		}
		switch x := v.(type) {
		case string:
			if x == "" {
				return &contractDriftError{op: op, field: pkField + "(empty in row#" + strconvI(i) + ")"}
			}
		case int64:
			if x == 0 {
				return &contractDriftError{op: op, field: pkField + "(zero in row#" + strconvI(i) + ")"}
			}
		case int:
			if x == 0 {
				return &contractDriftError{op: op, field: pkField + "(zero in row#" + strconvI(i) + ")"}
			}
		}
	}
	return nil
}

// strconvI 局部轻量 itoa,避免本包新增 strconv 导入(各处已有 fmt)。
func strconvI(n int) string {
	if n == 0 {
		return "0"
	}
	neg := n < 0
	if neg {
		n = -n
	}
	var buf [20]byte
	i := len(buf)
	for n > 0 {
		i--
		buf[i] = byte('0' + n%10)
		n /= 10
	}
	if neg {
		i--
		buf[i] = '-'
	}
	return string(buf[i:])
}

// UserAccountUpdate 用户账户设置修改(自动缴费开关)。
type UserAccountUpdate struct {
	AutoPay bool `json:"autoPay"`
}

// UserAddress 地址簿条目。
type UserAddress struct {
	CustomerID int64  `json:"customerId" binding:"required"`
	AddrCode   string `json:"addrCode" binding:"required"`
	Contact    string `json:"contact" binding:"required"`
	Phone      string `json:"phone" binding:"required"`
	Detail     string `json:"detail" binding:"required"`
	IsDefault  bool   `json:"isDefault"`
	// AddressPath 地址层级树 ltree 字符串(addresses.path 同构,弱引用,迁移 000152);
	// 空串=历史自由文本地址。
	AddressPath string `json:"addressPath"`
}

// UserPlan 套餐订购关系。
type UserPlan struct {
	CustomerID  int64  `json:"customerId" binding:"required"`
	ProductID   int64  `json:"productId" binding:"required"`
	PlanName    string `json:"planName" binding:"required"`
	Status      string `json:"status"`
	EffectiveAt string `json:"effectiveAt"`
}

// Addon 增值服务目录项。
type Addon struct {
	AddonID string `json:"addonId"`
	Name    string `json:"name" binding:"required"`
	Price   int64  `json:"price"`
	Status  string `json:"status"`
}

// AddonSubscription 增值服务订购/退订。
type AddonSubscription struct {
	CustomerID int64  `json:"customerId" binding:"required"`
	AddonID    string `json:"addonId" binding:"required"`
	Action     string `json:"action" binding:"required"`
}

// NotifySetting 通知订阅设置。
type NotifySetting struct {
	Business  bool   `json:"business"`
	Marketing bool   `json:"marketing"`
	Channel   string `json:"channel"`
}

// UserFaq 用户 FAQ 知识库条目。
type UserFaq struct {
	FaqID    string `json:"faqId"`
	Category string `json:"category" binding:"required"`
	Question string `json:"question" binding:"required"`
	Answer   string `json:"answer" binding:"required"`
	Active   bool   `json:"active"`
}

// UserMessage 用户消息。
type UserMessage struct {
	CustomerID int64  `json:"customerId" binding:"required"`
	Type       string `json:"type"`
	Title      string `json:"title" binding:"required"`
	Content    string `json:"content" binding:"required"`
}

// Coupon 优惠券。
type Coupon struct {
	CouponID   string `json:"couponId"`
	CustomerID int64  `json:"customerId" binding:"required"`
	Name       string `json:"name" binding:"required"`
	Amount     int64  `json:"amount"`
	ExpireAt   string `json:"expireAt"`
}

// Agreement 协议条款。
type Agreement struct {
	Type        string `json:"type" binding:"required"`
	Version     string `json:"version" binding:"required"`
	Content     string `json:"content" binding:"required"`
	EffectiveAt string `json:"effectiveAt"`
}

// BalanceAdjust 余额调整(delta 正充负扣,单位分)。
type BalanceAdjust struct {
	Delta int64 `json:"delta" binding:"required"`
}

// TopupDenomination 充值面额。
type TopupDenomination struct {
	Amount int64 `json:"amount" binding:"required"`
	Bonus  int64 `json:"bonus"`
	Active bool  `json:"active"`
}

// UserInvoice 电子发票开具请求。
type UserInvoice struct {
	CustomerID int64  `json:"customerId" binding:"required"`
	BillNo     string `json:"billNo" binding:"required"`
	InvoiceNo  string `json:"invoiceNo" binding:"required"`
	Amount     int64  `json:"amount"`
	Title      string `json:"title" binding:"required"`
}

// ProductSpec 产品卖点配置。
type ProductSpec struct {
	Highlights string `json:"highlights" binding:"required"`
	Specs      string `json:"specs" binding:"required"`
}

// Service 用户端数据域接口;列表行以字段名→值的 map 呈现(列名即 JSON 字段)。
type Service interface {
	ListUsers(ctx context.Context, keyword string) ([]map[string]any, error)
	GetUserDetail(ctx context.Context, customerID int64) (map[string]any, error)

	// Deprecated: 自动缴费权威态在 portal 域(portal_billing_prefs);user_accounts.auto_pay 已停用。
	UpdateUserAccount(ctx context.Context, customerID int64, u UserAccountUpdate) error

	// Deprecated: autoPay 列权威态在 portal_billing_prefs。
	ListUserAccounts(ctx context.Context) ([]map[string]any, error)
	ListUserAddresses(ctx context.Context) ([]map[string]any, error)
	CreateUserAddress(ctx context.Context, a UserAddress) (int64, error)
	// UpdateUserAddress 修改家庭地址明细;按 customerId + id 归属校验,
	// 未命中或归属不符返回 ErrNotFound;详情字段全量替换(detail/contact/phone/isDefault/addrCode)。
	UpdateUserAddress(ctx context.Context, customerID, id int64, a UserAddress) error
	// DeleteUserAddress 删除家庭地址;按 customerId 归属校验,未命中返回 ErrNotFound。
	DeleteUserAddress(ctx context.Context, customerID, id int64) error
	ListUserPlans(ctx context.Context) ([]map[string]any, error)
	CreateUserPlan(ctx context.Context, p UserPlan) (int64, error)

	// GetPlanForRenewal 续费前置:归属校验 + 产品基础月费(区域覆盖不参与,
	// 续费口径见 docs/design/promotion-coupon.md);未命中返回 ErrPlanNotFound。
	GetPlanForRenewal(ctx context.Context, planID, customerID int64) (productID int64, monthlyFee float64, err error)
	// RenewPlan 延长合约到期月:自 max(当前月, contract_end) 起 +months 月,
	// 返回新到期月 'YYYY-MM';未命中返回 ErrPlanNotFound。
	RenewPlan(ctx context.Context, planID, customerID int64, months int) (string, error)

	ListAddons(ctx context.Context) ([]map[string]any, error)
	CreateAddon(ctx context.Context, a Addon) error
	ToggleAddon(ctx context.Context, addonID string) error
	ListAddonSubscriptions(ctx context.Context) ([]map[string]any, error)
	CreateAddonSubscription(ctx context.Context, s AddonSubscription) (int64, error)

	// Deprecated: 通知偏好权威态在 portal 域(portal_prefs.notify)。
	ListNotifySettings(ctx context.Context) ([]map[string]any, error)

	// Deprecated: 通知偏好权威态在 portal 域(portal_prefs.notify)。
	UpdateNotifySettings(ctx context.Context, customerID int64, n NotifySetting) error

	ListUserFaqs(ctx context.Context) ([]map[string]any, error)
	CreateUserFaq(ctx context.Context, f UserFaq) error
	ToggleUserFaq(ctx context.Context, faqID string) error

	// 以下双胞胎方法已按裁定 D1(docs/notes/adopted/2026-08-20-db-dualtrack-convergence.md)降级为
	// 读 portal_*/既有权威表的薄适配层,实现不再读写 user_* 双胞胎表;编译兼容保留签名。

	// Deprecated: 消息权威态在 portal 域(portal_messages)。
	ListUserMessages(ctx context.Context, keyword string) ([]map[string]any, error)

	// Deprecated: 消息权威态在 portal 域(portal_messages)。
	CreateUserMessage(ctx context.Context, m UserMessage) (int64, error)

	// Deprecated: 消息权威态在 portal 域(portal_messages)。
	MarkAllMessagesRead(ctx context.Context, customerID int64) error

	ListCoupons(ctx context.Context) ([]map[string]any, error)
	CreateCoupon(ctx context.Context, cp Coupon) error
	DisableCoupon(ctx context.Context, couponID string) error

	GetInviteConfig(ctx context.Context) ([]map[string]any, error)
	// UpdateInviteConfig 更新邀请配置(邀请链接 + 奖励券模板)。
	UpdateInviteConfig(ctx context.Context, inviteLink string, rewardTemplateID int64) error
	ListUserUsages(ctx context.Context) ([]map[string]any, error)
	ListDiyGuides(ctx context.Context) ([]map[string]any, error)
	ToggleDiyGuide(ctx context.Context, guideID string) error

	ListAgreements(ctx context.Context) ([]map[string]any, error)
	UpdateAgreement(ctx context.Context, agreementID string, a Agreement) error

	// Deprecated: 余额权威态在 portal 域(portal_wallets)。
	ListUserBalances(ctx context.Context) ([]map[string]any, error)

	// Deprecated: 余额权威态在 portal 域(portal_wallets)。
	AdjustUserBalance(ctx context.Context, customerID int64, delta int64) error

	ListTopupDenominations(ctx context.Context) ([]map[string]any, error)
	UpdateTopupDenomination(ctx context.Context, denomID string, d TopupDenomination) error

	// Deprecated: 发票权威态在 billing 域(invoices);写路径已停写(ErrWriteStopped)。
	ListUserInvoices(ctx context.Context) ([]map[string]any, error)

	// Deprecated: 发票权威态在 billing 域(invoices);写路径已停写(ErrWriteStopped)。
	CreateUserInvoice(ctx context.Context, inv UserInvoice) (int64, error)

	// Deprecated: 投诉权威态在 order 域(complaints)。
	ListUserComplaints(ctx context.Context) ([]map[string]any, error)

	// Deprecated: 投诉权威态在 order 域(complaints)。
	CloseUserComplaint(ctx context.Context, complaintID string) error

	ListUserVerifyRecords(ctx context.Context) ([]map[string]any, error)

	ListProductSpecs(ctx context.Context) ([]map[string]any, error)
	UpdateProductSpec(ctx context.Context, productID string, p ProductSpec) error

	ListUserBillItems(ctx context.Context, billNo string) ([]map[string]any, error)
}
