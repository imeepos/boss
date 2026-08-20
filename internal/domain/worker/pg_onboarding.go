package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// regCols lists columns returned by the list query.
const regCols = `id, name, phone, id_card_no, group_id, region_id, status,
 review_note, reviewer_account_id, worker_id, submitted_at, reviewed_at`

// Submit 新建师傅注册申请(落 PENDING);submitted_at 由 DB 默认 now() 生成。
func (s *PGStore) Submit(ctx context.Context, reg Registration) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
INSERT INTO worker_registrations(name, phone, id_card_no, group_id, region_id)
VALUES($1,$2,$3,$4,$5) RETURNING id`,
		reg.Name, reg.Phone, reg.IDCardNo, reg.GroupID, reg.RegionID).Scan(&id)
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
func (s *PGStore) Approve(ctx context.Context, id, reviewerAccountID int64) (int64, error) {
	var (
		name     string
		groupID  int64
		regionID int64
		phone    string
		subAt    time.Time
	)
	err := s.db.QueryRow(ctx, `
SELECT name, group_id, region_id, phone, submitted_at
 FROM worker_registrations
 WHERE id = $1 AND status = $2`, id, RegStatusPending).Scan(&name, &groupID, &regionID, &phone, &subAt)
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
UPDATE worker_registrations SET status=$1, reviewer_account_id=$2, worker_id=$3, reviewed_at=now()
 WHERE id=$4 AND status=$5`,
		RegStatusApproved, reviewerAccountID, workerID, id, RegStatusPending)
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
