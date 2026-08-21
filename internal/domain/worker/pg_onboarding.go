package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// nullableInt64 把 ≤0 视作空(NULL),与 worker_registrations 的可空约束保持一致。
// 师傅自助注册允许 groupId=0 / regionId=0 表示待后台审核时补正,落库为 NULL。
func nullableInt64(v int64) any {
	if v <= 0 {
		return nil
	}
	return v
}

// regCols lists columns returned by the list query.
// group_id/region_id 自迁移 000089 可空(自助注册留 NULL 待审核补正),COALESCE 兜 0 防 scan 500。
const regCols = `id, name, phone, id_card_no, COALESCE(group_id, 0), COALESCE(region_id, 0), status,
 review_note, reviewer_account_id, worker_id, submitted_at, reviewed_at`

// Submit 新建师傅注册申请(落 PENDING);submitted_at 由 DB 默认 now() 生成。
// 校验 group_id 和 region_id 存在性,防止孤儿申请。
func (s *PGStore) Submit(ctx context.Context, reg Registration) (int64, error) {
	// 关联完整性校验
	if reg.GroupID > 0 {
		ok, err := s.exists(ctx, "worker_groups", reg.GroupID)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("worker: group %d: %w", reg.GroupID, ErrForeignKeyViolation)
		}
	}
	if reg.RegionID > 0 {
		ok, err := s.exists(ctx, "regions", reg.RegionID)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("worker: region %d: %w", reg.RegionID, ErrForeignKeyViolation)
		}
	}

	var id int64
	err := s.db.QueryRow(ctx, `
INSERT INTO worker_registrations(name, phone, id_card_no, group_id, region_id)
VALUES($1,$2,$3,$4,$5) RETURNING id`,
		reg.Name, reg.Phone, reg.IDCardNo,
		nullableInt64(reg.GroupID), nullableInt64(reg.RegionID)).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("worker: submit registration: %w", err)
	}
	return id, nil
}

// ListRegistrations 按状态过滤列出,空=全部;未审核在前,近期在后。
func (s *PGStore) ListRegistrations(ctx context.Context, status string) ([]Registration, error) {
	rows, err := s.db.Query(ctx, `
SELECT `+regCols+` FROM worker_registrations
 WHERE ($1 = '' OR status = $1)
 ORDER BY submitted_at DESC`, status)
	if err != nil {
		return nil, fmt.Errorf("worker: list registrations: %w", err)
	}
	defer rows.Close()
	out := make([]Registration, 0)
	for rows.Next() {
		var r Registration
		var (
			reviewedAt      pgtype.Timestamptz
			reviewerAccount pgtype.Int8
			workerID        pgtype.Int8
			reviewNote      pgtype.Text
		)
		if err := rows.Scan(&r.ID, &r.Name, &r.Phone, &r.IDCardNo, &r.GroupID, &r.RegionID,
			&r.Status, &reviewNote, &reviewerAccount, &workerID,
			&r.SubmittedAt, &reviewedAt); err != nil {
			return nil, fmt.Errorf("worker: scan registration: %w", err)
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
		if workerID.Valid {
			r.WorkerID = workerID.Int64
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ApproveRegistration 审核通过:校验 PENDING,创建 workers 主档,更新申请回填 worker_id。
// 返回新创建的 worker_id;status 冲突返回 ErrRegistrationConflict。
// 乐观锁:UPDATE ... WHERE status=PENDING 原子置状态,避免并发双重通过。
// 班组/区域由 caller 显式传入(groupID/regionID > 0),允许审核时纠正师傅登记时未填的字段。
func (s *PGStore) Approve(ctx context.Context, id, reviewerAccountID, groupID, regionID int64) (int64, error) {
	if groupID <= 0 || regionID <= 0 {
		return 0, ErrInvalidReviewFields
	}
	var (
		name  string
		phone string
		subAt time.Time
	)
	row := s.db.QueryRow(ctx, `
SELECT name, phone, submitted_at
 FROM worker_registrations
 WHERE id = $1 AND status = $2`, id, RegStatusPending)
	err := row.Scan(&name, &phone, &subAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, ErrRegistrationConflict
		}
		return 0, fmt.Errorf("worker: approve reg read: %w", err)
	}

	staffNo := fmt.Sprintf("WK-%d", subAt.UnixNano())
	var workerID int64
	err = s.db.QueryRow(ctx, `
INSERT INTO workers(staff_no, name, group_id, region_id, phone, status, joined_at, password_hash)
VALUES($1,$2,$3,$4,$5,1,$6,'') RETURNING id`,
		staffNo, name, groupID, regionID, phone, subAt).Scan(&workerID)
	if err != nil {
		return 0, fmt.Errorf("worker: approve create worker: %w", err)
	}

	res, err := s.db.Exec(ctx, `
UPDATE worker_registrations
 SET status=$1, reviewer_account_id=$2, worker_id=$3, group_id=$4, region_id=$5, reviewed_at=now()
 WHERE id=$6 AND status=$7`,
		RegStatusApproved, reviewerAccountID, workerID, groupID, regionID, id, RegStatusPending)
	if err != nil {
		return 0, fmt.Errorf("worker: approve reg update: %w", err)
	}
	if res.RowsAffected() == 0 {
		return 0, ErrRegistrationConflict
	}
	return workerID, nil
}

// RejectRegistration 驳回待审核申请;仅作用于 PENDING。
func (s *PGStore) Reject(ctx context.Context, id, reviewerAccountID int64, note string) error {
	res, err := s.db.Exec(ctx, `
UPDATE worker_registrations
 SET status=$1, review_note=$2, reviewer_account_id=$3, reviewed_at=now()
 WHERE id=$4 AND status=$5`,
		RegStatusRejected, note, reviewerAccountID, id, RegStatusPending)
	if err != nil {
		return fmt.Errorf("worker: reject registration: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrRegistrationConflict
	}
	return nil
}

// SubmitRealName 提交(or replace)师傅实名核验资料。
func (s *PGStore) SubmitRealName(ctx context.Context, v WorkerRealNameVerification) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
INSERT INTO verifications(subject_type, subject_id, method, real_name, id_card_no, result, verified_at)
VALUES('worker',$1,$2,$3,$4,$5,$6) RETURNING id`,
		v.WorkerID, v.Method, v.RealName, v.IDCardNo, v.Result, v.VerifiedAt).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("worker: submit real name: %w", err)
	}
	return id, nil
}

// GetLatestWorkerRealName 取师傅最新实名核验记录。
func (s *PGStore) GetLatest(ctx context.Context, workerID int64) (*WorkerRealNameVerification, error) {
	row := s.db.QueryRow(ctx, `
SELECT id, subject_id, method, real_name, id_card_no, result, verified_at, operator_account_id, operator_name
 FROM verifications
 WHERE subject_type = 'worker' AND subject_id = $1
 ORDER BY verified_at DESC
 LIMIT 1`, workerID)
	var v WorkerRealNameVerification
	var op pgtype.Int8
	var opName pgtype.Text
	if err := row.Scan(&v.ID, &v.WorkerID, &v.Method, &v.RealName, &v.IDCardNo,
		&v.Result, &v.VerifiedAt, &op, &opName); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrRealNameNotFound
		}
		return nil, fmt.Errorf("worker: get latest real name: %w", err)
	}
	if op.Valid {
		v.OperatorAccountID = op.Int64
	}
	if opName.Valid {
		v.OperatorName = opName.String
	}
	return &v, nil
}

// VerifyWorkerRealName 后台核验:仅作用于 PENDING 记录。
// 采用原子更新(受影响行>0),避免并发双重通过。
func (s *PGStore) Verify(ctx context.Context, workerID int64, result, operatorName string, operatorAccountID int64) error {
	res, err := s.db.Exec(ctx, `
UPDATE verifications
 SET result=$1, operator_account_id=$2, operator_name=$3, verified_at=now()
 WHERE subject_type='worker' AND subject_id=$4 AND result=$5`,
		result, operatorAccountID, operatorName, workerID, RealNamePending)
	if err != nil {
		return fmt.Errorf("worker: verify real name: %w", err)
	}
	if res.RowsAffected() == 0 {
		return ErrRealNameConflict
	}
	return nil
}
