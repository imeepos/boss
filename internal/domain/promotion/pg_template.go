package promotion

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const templateCols = `template_id, legal_entity_id, name, type, face_value, threshold,
	COALESCE(max_discount,0), scope_type, COALESCE(scope_ref,0), total_qty, issued_qty,
	per_customer_limit, COALESCE(valid_days,0), COALESCE(to_char(valid_from,'YYYY-MM-DD"T"HH24:MI:SSOF'),''),
	COALESCE(to_char(valid_to,'YYYY-MM-DD"T"HH24:MI:SSOF'),''), status, points_price`

// ListTemplates 模板列表(含禁用)。
func (s *PGStore) ListTemplates(ctx context.Context) ([]Template, error) {
	rows, err := s.db.Query(ctx, `SELECT `+templateCols+` FROM coupon_templates ORDER BY template_id`)
	if err != nil {
		return nil, fmt.Errorf("promotion: list templates: %w", err)
	}
	defer rows.Close()
	out := make([]Template, 0)
	for rows.Next() {
		t, err := scanTemplate(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *t)
	}
	return out, rows.Err()
}

func scanTemplate(row pgx.Row) (*Template, error) {
	var t Template
	if err := row.Scan(&t.TemplateID, &t.LegalEntityID, &t.Name, &t.Type, &t.FaceValue,
		&t.Threshold, &t.MaxDiscount, &t.ScopeType, &t.ScopeRef, &t.TotalQty, &t.IssuedQty,
		&t.PerCustomerLimit, &t.ValidDays, &t.ValidFrom, &t.ValidTo, &t.Status, &t.PointsPrice); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("promotion: scan template: %w", err)
	}
	return &t, nil
}

// CreateTemplate 新建模板(直接 ENABLED;停用走 disable 端点)。
func (s *PGStore) CreateTemplate(ctx context.Context, t Template) (int64, error) {
	defaults(&t)
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO coupon_templates(legal_entity_id, name, type, face_value, threshold,
			max_discount, scope_type, scope_ref, total_qty, per_customer_limit,
			valid_days, valid_from, valid_to, status, points_price)
		VALUES($1,$2,$3,$4,$5,NULLIF($6,0),$7,NULLIF($8,0),$9,$10,NULLIF($11,0),
			NULLIF($12,'')::timestamptz,NULLIF($13,'')::timestamptz,'ENABLED',$14)
		RETURNING template_id`,
		t.LegalEntityID, t.Name, t.Type, t.FaceValue, t.Threshold, t.MaxDiscount,
		t.ScopeType, t.ScopeRef, t.TotalQty, t.PerCustomerLimit,
		t.ValidDays, t.ValidFrom, t.ValidTo, t.PointsPrice).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("promotion: create template: %w", err)
	}
	return id, nil
}

// defaults 模板缺省值规整。
func defaults(t *Template) {
	if t.ScopeType == "" {
		t.ScopeType = "ALL"
	}
	if t.PerCustomerLimit <= 0 {
		t.PerCustomerLimit = 1
	}
	if t.ValidDays > 0 {
		t.ValidFrom, t.ValidTo = "", ""
	}
}

// DisableTemplate 停用模板(再发新券被拒,已发券不受影响)。
func (s *PGStore) DisableTemplate(ctx context.Context, templateID int64) error {
	return s.execAffected(ctx, "disable template",
		`UPDATE coupon_templates SET status='DISABLED' WHERE template_id=$1 AND status='ENABLED'`, templateID)
}

// Issue 按模板批量发券:模板行锁防超发,逐客户校验限领,快照写入 coupons。
func (s *PGStore) Issue(ctx context.Context, templateID int64, customerIDs []int64) (int, error) {
	tx, err := s.db.(beginner).Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("promotion: begin issue tx: %w", err)
	}
	defer tx.Rollback(ctx)

	t, err := scanTemplate(tx.QueryRow(ctx, `SELECT `+templateCols+` FROM coupon_templates
		WHERE template_id=$1 AND status='ENABLED' FOR UPDATE`, templateID))
	if err != nil {
		return 0, err
	}

	issued := 0
	for _, cid := range customerIDs {
		if t.TotalQty > 0 && t.IssuedQty+int64(issued+1) > t.TotalQty {
			return 0, &conflictError{reason: "超出发行总量"}
		}
		var held int
		if err := tx.QueryRow(ctx,
			`SELECT count(*) FROM coupons WHERE template_id=$1 AND customer_id=$2`,
			templateID, cid).Scan(&held); err != nil {
			return 0, fmt.Errorf("promotion: count held: %w", err)
		}
		if held >= t.PerCustomerLimit {
			continue
		}
		if _, err := insertCoupon(ctx, tx, *t, cid, SourceAdmin); err != nil {
			return 0, err
		}
		issued++
	}
	if _, err := tx.Exec(ctx,
		`UPDATE coupon_templates SET issued_qty=issued_qty+$2 WHERE template_id=$1`,
		templateID, issued); err != nil {
		return 0, fmt.Errorf("promotion: bump issued_qty: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("promotion: commit issue tx: %w", err)
	}
	return issued, nil
}

// insertCoupon 单券落库(快照自模板),返回券号。
func insertCoupon(ctx context.Context, tx pgx.Tx, t Template, cid int64, source string) (string, error) {
	couponID := randCode("CPN")
	expire := ""
	if t.ValidDays > 0 {
		expire = time.Now().AddDate(0, 0, t.ValidDays).Format(time.RFC3339)
	} else if t.ValidTo != "" {
		expire = t.ValidTo
	}
	if _, err := tx.Exec(ctx, `
		INSERT INTO coupons(coupon_id, customer_id, name, amount, template_id, type,
			face_value, threshold, max_discount, scope_type, scope_ref, source, expire_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,NULLIF($9,0),$10,NULLIF($11,0),$12,NULLIF($13,'')::timestamptz)`,
		couponID, cid, t.Name, t.FaceValue, t.TemplateID, t.Type,
		t.FaceValue, t.Threshold, t.MaxDiscount, t.ScopeType, t.ScopeRef, source, expire); err != nil {
		return "", fmt.Errorf("promotion: insert coupon: %w", err)
	}
	return couponID, nil
}

// IssueToCustomer 向单客户按模板发一张券(带来源,邀请奖励/积分兑换等场景),
// 同样受总量/限领约束;返回券号。
func (s *PGStore) IssueToCustomer(ctx context.Context, templateID, cid int64, source string) (string, error) {
	tx, err := s.db.(beginner).Begin(ctx)
	if err != nil {
		return "", fmt.Errorf("promotion: begin issue-one tx: %w", err)
	}
	defer tx.Rollback(ctx)

	t, err := scanTemplate(tx.QueryRow(ctx, `SELECT `+templateCols+` FROM coupon_templates
		WHERE template_id=$1 AND status='ENABLED' FOR UPDATE`, templateID))
	if err != nil {
		return "", err
	}
	if t.TotalQty > 0 && t.IssuedQty+1 > t.TotalQty {
		return "", &conflictError{reason: "超出发行总量"}
	}
	var held int
	if err := tx.QueryRow(ctx,
		`SELECT count(*) FROM coupons WHERE template_id=$1 AND customer_id=$2`,
		templateID, cid).Scan(&held); err != nil {
		return "", fmt.Errorf("promotion: count held: %w", err)
	}
	if held >= t.PerCustomerLimit {
		return "", &conflictError{reason: "已达每人限领数量"}
	}
	couponID, err := insertCoupon(ctx, tx, *t, cid, source)
	if err != nil {
		return "", err
	}
	if _, err := tx.Exec(ctx,
		`UPDATE coupon_templates SET issued_qty=issued_qty+1 WHERE template_id=$1`, templateID); err != nil {
		return "", fmt.Errorf("promotion: bump issued_qty: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return "", fmt.Errorf("promotion: commit issue-one tx: %w", err)
	}
	return couponID, nil
}

// CreateCodes 按模板生成兑换码批次。
func (s *PGStore) CreateCodes(ctx context.Context, templateID int64, count int) ([]CouponCode, error) {
	var exists int
	if err := s.db.QueryRow(ctx,
		`SELECT count(*) FROM coupon_templates WHERE template_id=$1 AND status='ENABLED'`,
		templateID).Scan(&exists); err != nil {
		return nil, fmt.Errorf("promotion: check template: %w", err)
	}
	if exists == 0 {
		return nil, ErrNotFound
	}
	out := make([]CouponCode, 0, count)
	for i := 0; i < count; i++ {
		cc := CouponCode{Code: randCode("RDM"), TemplateID: templateID, Status: "UNUSED"}
		if err := s.db.QueryRow(ctx, `
			INSERT INTO coupon_codes(code, template_id) VALUES($1,$2) RETURNING code_id`,
			cc.Code, templateID).Scan(&cc.CodeID); err != nil {
			return nil, fmt.Errorf("promotion: insert code: %w", err)
		}
		out = append(out, cc)
	}
	return out, nil
}

// ListCodes 模板下兑换码列表。
func (s *PGStore) ListCodes(ctx context.Context, templateID int64) ([]CouponCode, error) {
	rows, err := s.db.Query(ctx, `
		SELECT code_id, code, template_id, status, COALESCE(redeemed_by,0)
		FROM coupon_codes WHERE template_id=$1 ORDER BY code_id`, templateID)
	if err != nil {
		return nil, fmt.Errorf("promotion: list codes: %w", err)
	}
	defer rows.Close()
	out := make([]CouponCode, 0)
	for rows.Next() {
		var cc CouponCode
		if err := rows.Scan(&cc.CodeID, &cc.Code, &cc.TemplateID, &cc.Status, &cc.RedeemedBy); err != nil {
			return nil, fmt.Errorf("promotion: scan code: %w", err)
		}
		out = append(out, cc)
	}
	return out, rows.Err()
}

// PointsPrice 模板积分兑换价(0=不可积分兑换)。
func (s *PGStore) PointsPrice(ctx context.Context, templateID int64) (int64, error) {
	var price int64
	err := s.db.QueryRow(ctx,
		`SELECT points_price FROM coupon_templates WHERE template_id=$1`, templateID).Scan(&price)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrNotFound
		}
		return 0, fmt.Errorf("promotion: points price: %w", err)
	}
	return price, nil
}
