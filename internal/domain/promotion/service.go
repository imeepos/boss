package promotion

import "context"

// 券类型。
const (
	TypeFullCut = "FULL_CUT" // 满减:满 threshold 减 face_value
	TypeDiscout = "DISCOUNT" // 折扣:face_value 为折扣率‱(8500=85折),封顶 max_discount
	TypeCash    = "CASH"     // 代金券:无门槛直减 face_value
)

// 券来源。
const (
	SourceAdmin    = "ADMIN_ISSUE"
	SourceCampaign = "CAMPAIGN"
	SourceRedeem   = "REDEEM"
	SourceGift     = "GIFT"
	SourceInvite   = "INVITE"
	SourceLoyalty  = "LOYALTY"
)

// ErrNotFound 目标记录未命中。
var ErrNotFound = &notFoundError{}

type notFoundError struct{}

func (*notFoundError) Error() string { return "promotion: not found" }

// ErrConflict 状态冲突(超发/超领/券不可用/码已兑换)。
var ErrConflict = &conflictError{reason: ""}

type conflictError struct{ reason string }

func (e *conflictError) Error() string {
	if e.reason == "" {
		return "promotion: conflict"
	}
	return "promotion: conflict: " + e.reason
}

// Template 券模板。
type Template struct {
	TemplateID       int64  `json:"templateId"`
	LegalEntityID    int64  `json:"legalEntityId" binding:"required"`
	Name             string `json:"name" binding:"required"`
	Type             string `json:"type" binding:"required,oneof=FULL_CUT DISCOUNT CASH"`
	FaceValue        int64  `json:"faceValue" binding:"required,gt=0"` // 分;折扣率为‱
	Threshold        int64  `json:"threshold"`                         // 分,0=无门槛
	MaxDiscount      int64  `json:"maxDiscount"`                       // 分,折扣券封顶,0=不封顶
	ScopeType        string `json:"scopeType"`                         // ALL/PRODUCT/FIRST_ORDER
	ScopeRef         int64  `json:"scopeRef"`
	TotalQty         int64  `json:"totalQty"` // 0=不限
	PerCustomerLimit int    `json:"perCustomerLimit"`
	ValidDays        int    `json:"validDays"` // 领取后 N 天有效;0=用固定窗口
	ValidFrom        string `json:"validFrom"` // RFC3339,可空
	ValidTo          string `json:"validTo"`   // RFC3339,可空
	Status           string `json:"status"`    // DRAFT/ENABLED/DISABLED
	IssuedQty        int64  `json:"issuedQty"`
	PointsPrice      int64  `json:"pointsPrice"` // 积分兑换价,0=不可积分兑换(000104)
}

// Coupon 券实例(客户视角)。
type Coupon struct {
	CouponID    string `json:"couponId"`
	CustomerID  int64  `json:"customerId"`
	TemplateID  int64  `json:"templateId"`
	Name        string `json:"name"`
	Type        string `json:"type"`
	FaceValue   int64  `json:"faceValue"` // 分
	Threshold   int64  `json:"threshold"` // 分
	MaxDiscount int64  `json:"maxDiscount"`
	Source      string `json:"source"`
	Code        string `json:"code,omitempty"` // 转赠载体,非空=转赠中
	Status      string `json:"status"`         // ISSUED/USED/EXPIRED/DISABLED
	ExpireAt    string `json:"expireAt"`
}

// CouponCode 兑换码。
type CouponCode struct {
	CodeID     int64  `json:"codeId"`
	Code       string `json:"code"`
	TemplateID int64  `json:"templateId"`
	Status     string `json:"status"` // UNUSED/REDEEMED/DISABLED
	RedeemedBy int64  `json:"redeemedBy"`
}

// Service 营销促销域接口。
type Service interface {
	ListTemplates(ctx context.Context) ([]Template, error)
	// ListExchangeOffers 积分可兑换模板(启用且 points_price>0,按积分价升序)。
	ListExchangeOffers(ctx context.Context) ([]Template, error)
	CreateTemplate(ctx context.Context, t Template) (int64, error)
	DisableTemplate(ctx context.Context, templateID int64) error

	// Issue 按模板向客户批量发券(限总量/限每人),返回实际发放数。
	Issue(ctx context.Context, templateID int64, customerIDs []int64) (int, error)
	// IssueToCustomer 向单客户按模板发一张券(带来源:邀请奖励/积分兑换),返回券号。
	IssueToCustomer(ctx context.Context, templateID, customerID int64, source string) (string, error)
	// PointsPrice 模板积分兑换价(0=不可积分兑换);模板不存在返回 ErrNotFound。
	PointsPrice(ctx context.Context, templateID int64) (int64, error)
	// CreateCodes 按模板生成兑换码批次。
	CreateCodes(ctx context.Context, templateID int64, count int) ([]CouponCode, error)
	ListCodes(ctx context.Context, templateID int64) ([]CouponCode, error)
	// RedeemCode 兑换码领券 / 接收转赠券,返回券号。
	RedeemCode(ctx context.Context, code string, customerID int64) (string, error)
	// CreateGift 生成转赠码(整券转赠,一次有效)。
	CreateGift(ctx context.Context, couponID string, customerID int64) (string, error)

	// ListCustomerCoupons 客户券仓;status 取 available/used/expired/all(派生态);
	// billCents>0 时 available 项附带 estDeduct 预估抵扣,并过滤门槛不满足的券。
	ListCustomerCoupons(ctx context.Context, customerID int64, status string, billCents int64) ([]map[string]any, error)

	// RollbackRedemption 退款回退:核销记录反写,券过期则作废否则回 ISSUED。
	RollbackRedemption(ctx context.Context, paymentID int64) error

	// ---- 赠送时长(阶梯规则:6送1/12送3/24送6) ----

	ListGiftRules(ctx context.Context) ([]GiftRule, error)
	CreateGiftRule(ctx context.Context, r GiftRule) (int64, error)
	DisableGiftRule(ctx context.Context, ruleID int64) error
	// MatchGiftRule 命中实购月数的最优档(buyMonths 最大且 <= 实购);
	// 无命中返回 nil。
	MatchGiftRule(ctx context.Context, productID int64, buyMonths int) (*GiftRule, error)
	// RecordGift 赠送落痕(缴费成功后调用);paymentID 可为 0。
	RecordGift(ctx context.Context, r GiftRecord) error
}

// GiftRule 赠送时长阶梯规则。
type GiftRule struct {
	RuleID        int64  `json:"ruleId"`
	LegalEntityID int64  `json:"legalEntityId" binding:"required"`
	Name          string `json:"name" binding:"required"`
	ScopeType     string `json:"scopeType"` // ALL/PRODUCT
	ScopeRef      int64  `json:"scopeRef"`
	BuyMonths     int    `json:"buyMonths" binding:"required,gt=0"`
	GiftMonths    int    `json:"giftMonths" binding:"required,gt=0"`
	Status        string `json:"status"` // ENABLED/DISABLED
}

// GiftRecord 赠送时长发放记录。
type GiftRecord struct {
	RuleID     int64 `json:"ruleId"`
	CustomerID int64 `json:"customerId"`
	ProductID  int64 `json:"productId"`
	BuyMonths  int   `json:"buyMonths"`
	GiftMonths int   `json:"giftMonths"`
	PaymentID  int64 `json:"paymentId"`
}
