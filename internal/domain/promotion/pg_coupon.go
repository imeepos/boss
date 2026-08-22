package promotion

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// ListCustomerCoupons 客户券仓;status 展示态(available/used/expired/all)由
// ISSUED/USED/EXPIRED/DISABLED+有效期+code 派生;billCents>0 时附带预估抵扣并过滤门槛不满足券。
func (s *PGStore) ListCustomerCoupons(ctx context.Context, customerID int64, status string, billCents int64) ([]map[string]any, error) {
	rows, err := s.db.Query(ctx, `
		SELECT coupon_id, COALESCE(template_id,0), name, type, COALESCE(face_value,amount),
			threshold, COALESCE(max_discount,0), source, COALESCE(code,''), status,
			COALESCE(to_char(expire_at,'YYYY-MM-DD"T"HH24:MI:SSOF'),'')
		FROM coupons WHERE customer_id=$1 ORDER BY coupon_id`, customerID)
	if err != nil {
		return nil, fmt.Errorf("promotion: list customer coupons: %w", err)
	}
	defer rows.Close()
	out := make([]map[string]any, 0)
	for rows.Next() {
		c := Coupon{}
		if err := rows.Scan(&c.CouponID, &c.TemplateID, &c.Name, &c.Type, &c.FaceValue,
			&c.Threshold, &c.MaxDiscount, &c.Source, &c.Code, &c.Status, &c.ExpireAt); err != nil {
			return nil, fmt.Errorf("promotion: scan coupon: %w", err)
		}
		item, ok := couponView(c, status, billCents)
		if ok {
			out = append(out, item)
		}
	}
	return out, rows.Err()
}

// couponView 单券展示视图;不匹配过滤态返回 ok=false。
func couponView(c Coupon, filter string, billCents int64) (map[string]any, bool) {
	view := c.Status
	switch {
	case c.Status == "ISSUED" && c.Code != "":
		view = "gifting" // 转赠中:自身不可用,仅在 all 视图展示
	case c.Status == "ISSUED" && expired(c.ExpireAt):
		view = "expired"
	case c.Status == "ISSUED":
		view = "available"
	}
	if filter != "all" && filter != "" && view != filter {
		return nil, false
	}
	item := map[string]any{
		"couponId": c.CouponID, "templateId": c.TemplateID, "name": c.Name,
		"type": c.Type, "faceValue": c.FaceValue, "threshold": c.Threshold,
		"source": c.Source, "status": view, "expireAt": c.ExpireAt,
	}
	if view == "available" && billCents > 0 {
		d := deductAmount(c.Type, c.FaceValue, c.Threshold, c.MaxDiscount, billCents)
		if d < 0 {
			return nil, false // 未达门槛,不进可用列表
		}
		item["estDeduct"] = d
	}
	return item, true
}

// expired 有效期字符串判过期(空=永久)。
func expired(expireAt string) bool {
	if expireAt == "" {
		return false
	}
	t, err := time.Parse(time.RFC3339, expireAt)
	return err == nil && t.Before(time.Now())
}

// RedeemCode 兑换码领券(码批次→按模板发券)或接收转赠券(券身 code→过户),返回券号。
func (s *PGStore) RedeemCode(ctx context.Context, code string, customerID int64) (string, error) {
	tx, err := s.db.(beginner).Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("promotion: begin redeem tx: %w", err)
	}
	defer tx.Rollback(ctx)

	// 先试转赠:券身挂 code,过户即领取。
	var couponID string
	err = tx.QueryRow(ctx, `
		UPDATE coupons SET customer_id=$2, code=NULL, source='GIFT'
		WHERE code=$1 AND status='ISSUED' AND customer_id<>$2
		RETURNING coupon_id`, code, customerID).Scan(&couponID)
	if err == nil {
		if err := tx.Commit(ctx); err != nil {
			return "", fmt.Errorf("promotion: commit gift redeem: %w", err)
		}
		return couponID, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return "", fmt.Errorf("promotion: redeem gift: %w", err)
	}

	// 再试兑换码批次。
	var t Template
	row := tx.QueryRow(ctx, `
		SELECT `+templateCols+` FROM coupon_templates tt
		JOIN coupon_codes cc ON cc.template_id=tt.template_id
		WHERE cc.code=$1 AND cc.status='UNUSED' AND tt.status='ENABLED' FOR UPDATE OF cc`, code)
	tp, err := scanTemplate(row)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return "", &conflictError{reason: "兑换码无效或已被使用"}
		}
		return "", err
	}
	t = *tp
	var held int
	if err := tx.QueryRow(ctx,
		`SELECT count(*) FROM coupons WHERE template_id=$1 AND customer_id=$2`,
		t.TemplateID, customerID).Scan(&held); err != nil {
		return "", fmt.Errorf("promotion: count held: %w", err)
	}
	if held >= t.PerCustomerLimit {
		return "", &conflictError{reason: "已达每人限领数量"}
	}
	if err := insertCoupon(ctx, tx, t, customerID, SourceRedeem); err != nil {
		return "", err
	}
	// insertCoupon 未回填券号,反查最新一张(同事务可见)。
	if err := tx.QueryRow(ctx, `
		SELECT coupon_id FROM coupons WHERE customer_id=$1 AND template_id=$2
		ORDER BY issued_at DESC, coupon_id DESC LIMIT 1`, customerID, t.TemplateID).Scan(&couponID); err != nil {
		return "", fmt.Errorf("promotion: fetch redeemed coupon: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE coupon_codes SET status='REDEEMED', redeemed_by=$2, redeemed_at=now() WHERE code=$1`,
		code, customerID); err != nil {
		return "", fmt.Errorf("promotion: mark code redeemed: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`UPDATE coupon_templates SET issued_qty=issued_qty+1 WHERE template_id=$1`, t.TemplateID); err != nil {
		return "", fmt.Errorf("promotion: bump issued_qty: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("promotion: commit redeem: %w", err)
	}
	return couponID, nil
}

// CreateGift 生成转赠码(整券转赠,一次有效;券须为本人可用态)。
func (s *PGStore) CreateGift(ctx context.Context, couponID string, customerID int64) (string, error) {
	code := randCode("GFT")
	tag, err := s.db.Exec(ctx, `
		UPDATE coupons SET code=$3
		WHERE coupon_id=$1 AND customer_id=$2 AND status='ISSUED' AND code IS NULL
		  AND (expire_at IS NULL OR expire_at > now())`,
		couponID, customerID, code)
	if err != nil {
		return "", fmt.Errorf("promotion: create gift: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return "", &conflictError{reason: "券不可转赠: " + couponID}
	}
	return code, nil
}

// RollbackRedemption 退款回退:按流水反查核销记录,券过期作废(DISABLED)否则回 ISSUED。
func (s *PGStore) RollbackRedemption(ctx context.Context, paymentID int64) error {
	tx, err := s.db.(beginner).Begin(ctx)
	if err != nil {
		return fmt.Errorf("promotion: begin rollback tx: %w", err)
	}
	defer tx.Rollback(ctx)
	var couponID string
	err = tx.QueryRow(ctx, `
		UPDATE coupons c SET status = CASE WHEN c.expire_at IS NULL OR c.expire_at > now()
				THEN 'ISSUED' ELSE 'DISABLED' END,
			used_at=NULL, payment_id=NULL
		FROM coupon_redemptions r
		WHERE r.coupon_id=c.coupon_id AND r.payment_id=$1 AND c.status='USED'
		RETURNING c.coupon_id`, paymentID).Scan(&couponID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("promotion: rollback coupon: %w", err)
	}
	if _, err := tx.Exec(ctx,
		`DELETE FROM coupon_redemptions WHERE payment_id=$1`, paymentID); err != nil {
		return fmt.Errorf("promotion: delete redemption: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("promotion: commit rollback: %w", err)
	}
	return nil
}
