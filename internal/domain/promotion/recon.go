package promotion

import "context"

// 券对账差异种类(按模板定位)。
const (
	ReconDiffMatch          = "MATCH"           // 计数一致且核销闭环
	ReconDiffCounterDrift   = "COUNTER_DRIFT"   // 模板 issued_qty ≠ 券实例数
	ReconDiffRedemptionLost = "REDEMPTION_LOST" // USED 券无核销记录或金额不符
)

// CouponReconRow 券对账行:一模板的发放计数/状态分布/核销金额三角。
type CouponReconRow struct {
	TemplateID   int64            `json:"templateId"`
	Name         string           `json:"name"`
	Type         string           `json:"type"`
	Status       string           `json:"status"`
	IssuedQty    int64            `json:"issuedQty"`    // 模板计数器
	ActualIssued int64            `json:"actualIssued"` // 券实例实数
	ByStatus     map[string]int64 `json:"byStatus"`     // ISSUED/USED/EXPIRED/DISABLED
	// 核销三角:USED 券数 / 核销记录数 / 核销金额合计(分)。
	UsedCount      int64  `json:"usedCount"`
	RedemptionCnt  int64  `json:"redemptionCnt"`
	RedeemedAmount int64  `json:"redeemedAmount"`
	FaceValueTotal int64  `json:"faceValueTotal"` // 在途券面值敞口(分,未核销 ISSUED)
	DiffKind       string `json:"diffKind"`
}

// CouponReconSummary 券对账汇总。
type CouponReconSummary struct {
	Templates      int            `json:"templates"`
	ActualIssued   int64          `json:"actualIssued"`
	UsedCount      int64          `json:"usedCount"`
	RedeemedAmount int64          `json:"redeemedAmount"`
	FaceValueTotal int64          `json:"faceValueTotal"`
	ByDiff         map[string]int `json:"byDiff"`
}

// CouponRecon 券对账报表(全模板)。
func (s *PGStore) CouponRecon(ctx context.Context) ([]CouponReconRow, CouponReconSummary, error) {
	rows, err := s.reconRows(ctx)
	if err != nil {
		return nil, CouponReconSummary{}, err
	}
	sum := CouponReconSummary{Templates: len(rows), ByDiff: map[string]int{}}
	for _, r := range rows {
		sum.ActualIssued += r.ActualIssued
		sum.UsedCount += r.UsedCount
		sum.RedeemedAmount += r.RedeemedAmount
		sum.FaceValueTotal += r.FaceValueTotal
		sum.ByDiff[r.DiffKind]++
	}
	return rows, sum, nil
}
