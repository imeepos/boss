package promotion

import (
	"context"
	"fmt"
)

// reconRows 全模板对账:券实例状态分布 + 核销记录金额,与模板计数器三方核对。
func (s *PGStore) reconRows(ctx context.Context) ([]CouponReconRow, error) {
	// 一行一模板:实例分布与核销聚合同查询取出,再在内存判差异。
	rows, err := s.db.Query(ctx, `
		SELECT tt.template_id, tt.name, tt.type, tt.status, tt.issued_qty,
		       COUNT(c.coupon_id),
		       COUNT(*) FILTER (WHERE c.status='ISSUED'),
		       COUNT(*) FILTER (WHERE c.status='USED'),
		       COUNT(*) FILTER (WHERE c.status='EXPIRED'),
		       COUNT(*) FILTER (WHERE c.status='DISABLED'),
		       COALESCE(SUM(c.face_value) FILTER (WHERE c.status='ISSUED'),0),
		       (SELECT COUNT(*) FROM coupon_redemptions r
		         WHERE r.coupon_id IN (SELECT coupon_id FROM coupons WHERE template_id=tt.template_id)),
		       (SELECT COALESCE(SUM(r.deducted_amount),0) FROM coupon_redemptions r
		         WHERE r.coupon_id IN (SELECT coupon_id FROM coupons WHERE template_id=tt.template_id))
		FROM coupon_templates tt
		LEFT JOIN coupons c ON c.template_id = tt.template_id
		GROUP BY tt.template_id, tt.name, tt.type, tt.status, tt.issued_qty
		ORDER BY tt.template_id`)
	if err != nil {
		return nil, fmt.Errorf("promotion: recon query: %w", err)
	}
	defer rows.Close()
	out := make([]CouponReconRow, 0)
	for rows.Next() {
		var r CouponReconRow
		var issued, used, expired, disabled int64
		if err := rows.Scan(&r.TemplateID, &r.Name, &r.Type, &r.Status, &r.IssuedQty,
			&r.ActualIssued, &issued, &used, &expired, &disabled,
			&r.FaceValueTotal, &r.RedemptionCnt, &r.RedeemedAmount); err != nil {
			return nil, fmt.Errorf("promotion: recon scan: %w", err)
		}
		r.ByStatus = map[string]int64{
			"ISSUED": issued, "USED": used, "EXPIRED": expired, "DISABLED": disabled,
		}
		r.UsedCount = used
		r.DiffKind = classifyCouponRecon(r)
		out = append(out, r)
	}
	return out, rows.Err()
}

// classifyCouponRecon 差异判定:计数漂移优先,其次核销闭环缺失,否则 MATCH。
func classifyCouponRecon(r CouponReconRow) string {
	if r.IssuedQty != r.ActualIssued {
		return ReconDiffCounterDrift
	}
	if r.UsedCount != r.RedemptionCnt {
		return ReconDiffRedemptionLost
	}
	return ReconDiffMatch
}
