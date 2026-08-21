package customer

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// custRegCols 注册申请列表查询列。
const custRegCols = `id, name, phone, id_card_no, legal_entity_id, address_id, region_id, status,
 review_note, reviewer_account_id, customer_id, submitted_at, reviewed_at`

// Submit 新建客户注册申请(落 PENDING);submitted_at 由 DB 默认 now() 生成。
// 校验 legal_entity_id, address_id, region_id 存在性,防止孤儿申请。
func (s *PGStore) Submit(ctx context.Context, reg Registration) (int64, error) {
	// 关联完整性校验
	if reg.LegalEntityID > 0 {
		ok, err := s.exists(ctx, "legal_entities", reg.LegalEntityID)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("customer: legal entity %d: %w", reg.LegalEntityID, ErrForeignKeyViolation)
		}
	}
	if reg.AddressID > 0 {
		ok, err := s.exists(ctx, "addresses", reg.AddressID)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("customer: address %d: %w", reg.AddressID, ErrForeignKeyViolation)
		}
	}
	if reg.RegionID > 0 {
		ok, err := s.exists(ctx, "regions", reg.RegionID)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("customer: region %d: %w", reg.RegionID, ErrForeignKeyViolation)
		}
	}

	var id int64
	err := s.db.QueryRow(ctx, `
INSERT INTO customer_registrations(name, phone, id_card_no, legal_entity_id, address_id, region_id)
VALUES($1,$2,$3,$4,$5,$6) RETURNING id`,
		reg.Name, reg.Phone, reg.IDCardNo, reg.LegalEntityID, reg.AddressID, reg.RegionID).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("customer: submit registration: %w", err)
	}
	return id, nil
}

// ListRegistrations 按状态过滤列出,空=全部;未审核在前,近期在后。
func (s *PGStore) ListRegistrations(ctx context.Context, status string) ([]Registration, error) {
	rows, err := s.db.Query(ctx, `
SELECT `+custRegCols+` FROM customer_registrations
 WHERE ($1 = '' OR status = $1)
 ORDER BY submitted_at DESC`, status)
	if err != nil {
		return nil, fmt.Errorf("customer: list registrations: %w", err)
	}
	defer rows.Close()
	out := make([]Registration, 0)
	for rows.Next() {
		var r Registration
		var (
			reviewedAt      pgtype.Timestamptz
			reviewerAccount pgtype.Int8
			customerID      pgtype.Int8
			reviewNote      pgtype.Text
		)
		if err := rows.Scan(&r.ID, &r.Name, &r.Phone, &r.IDCardNo, &r.LegalEntityID, &r.AddressID,
			&r.RegionID, &r.Status, &reviewNote, &reviewerAccount, &customerID,
			&r.SubmittedAt, &reviewedAt); err != nil {
			return nil, fmt.Errorf("customer: scan registration: %w", err)
		}
		if reviewedAt.Valid {
			t := reviewedAt.Time
			r.ReviewedAt = &t
		}
		if reviewNote.Valid {
			r.ReviewNote = reviewNote.String
		}
		if reviewerAccount.Valid {
			r.ReviewerAccountID = reviewerAccount.Int64
		}
		if customerID.Valid {
			r.CustomerID = customerID.Int64
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Approve 审核通过:校验 PENDING,创建 customers 主档,更新申请回填 customer_id。
// 返回新创建的 customer_id;status 冲突返回 ErrRegistrationConflict。
// 乐观锁:UPDATE ... WHERE status=PENDING 原子置状态,避免并发双重通过。
func (s *PGStore) Approve(ctx context.Context, id, reviewerAccountID int64) (int64, error) {
	var (
		name          string
		phone         string
		idCardNo      string
		legalEntityID int64
		addressID     int64
		regionID      int64
	)
	err := s.db.QueryRow(ctx, `
SELECT name, phone, id_card_no, legal_entity_id, address_id, region_id
 FROM customer_registrations
 WHERE id = $1 AND status = $2`, id, RegStatusPending).Scan(&name, &phone, &idCardNo, &legalEntityID, &addressID, &regionID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrRegistrationConflict
		}
		return 0, fmt.Errorf("customer: approve reg read: %w", err)
	}

	// 经营区域名快照(改名不改历史)。
	var regionName string
	if err := s.db.QueryRow(ctx, `SELECT name FROM regions WHERE id = $1`, regionID).Scan(&regionName); err != nil {
		return 0, fmt.Errorf("customer: approve region name: %w", err)
	}

	var customerID int64
	err = s.db.QueryRow(ctx, `
INSERT INTO customers(name, phone, id_type, id_no, real_name_status, service_status,
                      address_id, legal_entity_id, region_id, region_name)
VALUES($1,$2,'身份证',$3,'PENDING','ACTIVE',$4,$5,$6,$7) RETURNING id`,
		name, phone, idCardNo, addressID, legalEntityID, regionID, regionName).Scan(&customerID)
	if err != nil {
		return 0, fmt.Errorf("customer: approve create customer: %w", err)
	}

	res, err := s.db.Exec(ctx, `
UPDATE customer_registrations SET status=$1, reviewer_account_id=$2, customer_id=$3, reviewed_at=now()
 WHERE id=$4 AND status=$5`,
		RegStatusApproved, reviewerAccountID, customerID, id, RegStatusPending)
	if err != nil {
		return 0, fmt.Errorf("customer: approve reg update: %w", err)
	}
	if res.RowsAffected() == 0 {
		return 0, ErrRegistrationConflict
	}
	return customerID, nil
}

// Reject 驳回待审核申请;仅作用于 PENDING。
func (s *PGStore) Reject(ctx context.Context, id, reviewerAccountID int64, note string) error {
	res, err := s.db.Exec(ctx, `
UPDATE customer_registrations
 SET status=$1, review_note=$2, reviewer_account_id=$3, reviewed_at=now()
 WHERE id=$4 AND status=$5`,
		RegStatusRejected, note, reviewerAccountID, id, RegStatusPending)
	if err != nil {
		return fmt.Errorf("customer: reject registration: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrRegistrationConflict
	}
	return nil
}

// SubmitRealName 提交客户实名核验资料。
func (s *PGStore) SubmitRealName(ctx context.Context, v CustomerRealNameVerification) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
INSERT INTO verifications(subject_type, subject_id, method, real_name, id_card_no, result,
                          id_card_front_id, id_card_back_id)
VALUES('customer',$1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		v.CustomerID, v.Method, v.RealName, v.IDCardNo, v.Result,
		v.IDCardFrontID, v.IDCardBackID).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("customer: submit real name: %w", err)
	}
	return id, nil
}

// GetLatest 取客户最新实名核验记录。
func (s *PGStore) GetLatest(ctx context.Context, customerID int64) (*CustomerRealNameVerification, error) {
	row := s.db.QueryRow(ctx, `
SELECT id, subject_id, method, real_name, id_card_no, result, reject_reason,
       id_card_front_id, id_card_back_id, verified_at, operator_account_id, operator_name
 FROM verifications
 WHERE subject_type = 'customer' AND subject_id = $1
 ORDER BY verified_at DESC
 LIMIT 1`, customerID)
	var v CustomerRealNameVerification
	var op pgtype.Int8
	var opName pgtype.Text
	if err := row.Scan(&v.ID, &v.CustomerID, &v.Method, &v.RealName, &v.IDCardNo,
		&v.Result, &v.RejectReason, &v.IDCardFrontID, &v.IDCardBackID,
		&v.VerifiedAt, &op, &opName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRealNameNotFound
		}
		return nil, fmt.Errorf("customer: get latest real name: %w", err)
	}
	if op.Valid {
		v.OperatorAccountID = op.Int64
	}
	if opName.Valid {
		v.OperatorName = opName.String
	}
	return &v, nil
}

// Verify 后台核验:仅作用于 PENDING 记录;PASS 同步 customers.real_name_status=VERIFIED,FAIL 记驳回原因。
// 采用原子更新(受影响行>0),避免并发双重通过。
func (s *PGStore) Verify(ctx context.Context, customerID int64, result, reason, operatorName string, operatorAccountID int64) error {
	res, err := s.db.Exec(ctx, `
UPDATE verifications
 SET result=$1, reject_reason=$2, operator_account_id=$3, operator_name=$4, verified_at=now()
 WHERE subject_type='customer' AND subject_id=$5 AND result=$6`,
		result, reason, operatorAccountID, operatorName, customerID, RealNamePending)
	if err != nil {
		return fmt.Errorf("customer: verify real name: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrRealNameConflict
	}
	if result == RealNamePass {
		if _, err := s.db.Exec(ctx,
			`UPDATE customers SET real_name_status='VERIFIED' WHERE id=$1`, customerID); err != nil {
			return fmt.Errorf("customer: verify sync status: %w", err)
		}
	}
	return nil
}

// compile-time: PGStore 满足 OnboardingService。
var _ OnboardingService = (*PGStore)(nil)
