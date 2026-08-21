package worker

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func TestPGStore_Submit(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// FK validation: group exists
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(6)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	// FK validation: region exists
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(4)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectQuery(`INSERT INTO worker_registrations\(name, phone, id_card_no, group_id, region_id\)`).
		WithArgs("王师傅", "13800000001", "110101199001011234", int64(6), int64(4)).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(7)))

	s := NewPGStore(mock)
	id, err := s.Submit(context.Background(), Registration{
		Name: "王师傅", Phone: "13800000001", IDCardNo: "110101199001011234",
		GroupID: 6, RegionID: 4,
	})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if id != 7 {
		t.Fatalf("id=%d, want 7", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// TestPGStore_SubmitAllowsEmpty 师傅端自助注册允许 groupId/regionId=0,以 NULL 落库等待后台补正。
func TestPGStore_SubmitAllowsEmpty(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO worker_registrations\(name, phone, id_card_no, group_id, region_id\)`).
		WithArgs("李师傅", "13900000002", "110101199001011235", nil, nil).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(8)))

	s := NewPGStore(mock)
	id, err := s.Submit(context.Background(), Registration{
		Name: "李师傅", Phone: "13900000002", IDCardNo: "110101199001011235",
	})
	if err != nil {
		t.Fatalf("Submit empty: %v", err)
	}
	if id != 8 {
		t.Fatalf("id=%d, want 8", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ListRegistrations(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "name", "phone", "id_card_no", "group_id", "region_id",
		"status", "review_note", "reviewer_account_id", "worker_id", "submitted_at", "reviewed_at"}
	subAt := time.Date(2026, 8, 19, 8, 0, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT id, name, phone, id_card_no, group_id, region_id, status, review_note, reviewer_account_id, worker_id, submitted_at, reviewed_at FROM worker_registrations`).
		WithArgs("PENDING").
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), "王师傅", "13800000001", "110101199001011234", int64(6), int64(4),
				"PENDING", nil, nil, nil, subAt, nil))

	s := NewPGStore(mock)
	got, err := s.ListRegistrations(context.Background(), "PENDING")
	if err != nil {
		t.Fatalf("ListRegistrations: %v", err)
	}
	if len(got) != 1 {
		t.Fatalf("len=%d, want 1", len(got))
	}
	r := got[0]
	if r.Status != "PENDING" || r.Name != "王师傅" || r.GroupID != 6 || r.RegionID != 4 {
		t.Fatalf("row=%+v", r)
	}
	if r.SubmittedAt.IsZero() {
		t.Fatalf("submittedAt should be set")
	}
	if r.ReviewerAccountID != 0 || r.WorkerID != 0 {
		t.Fatalf("nullable fields should be zero, got rev=%d wid=%d", r.ReviewerAccountID, r.WorkerID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_Approve(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	subAt := time.Date(2026, 8, 19, 8, 0, 0, 0, time.UTC)
	// 1) 读待审核行
	mock.ExpectQuery(`SELECT name, phone, submitted_at FROM worker_registrations`).
		WithArgs(int64(1), RegStatusPending).
		WillReturnRows(mock.NewRows([]string{"name", "phone", "submitted_at"}).
			AddRow("王师傅", "13800000001", subAt))
	// 2) 建 workers 主档
	mock.ExpectQuery(`INSERT INTO workers\(staff_no, name, group_id, region_id, phone, status, joined_at, password_hash\)`).
		WithArgs(pgxmock.AnyArg(), "王师傅", int64(6), int64(4), "13800000001", subAt).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(9)))
	// 3) 乐观锁更新(回填 worker_id + 校正 group_id/region_id)
	mock.ExpectExec(`UPDATE worker_registrations SET status=\$1, reviewer_account_id=\$2, worker_id=\$3, group_id=\$4, region_id=\$5, reviewed_at=now\(\)`).
		WithArgs(RegStatusApproved, int64(103), int64(9), int64(6), int64(4), int64(1), RegStatusPending).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	s := NewPGStore(mock)
	wid, err := s.Approve(context.Background(), 1, 103, 6, 4)
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if wid != 9 {
		t.Fatalf("workerId=%d, want 9", wid)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// TestPGStore_ApproveMissingGroup 审核时未补 group/region 应返回 ErrInvalidReviewFields。
func TestPGStore_ApproveMissingGroup(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	s := NewPGStore(mock)
	if _, err := s.Approve(context.Background(), 1, 103, 0, 4); !errors.Is(err, ErrInvalidReviewFields) {
		t.Fatalf("err=%v, want ErrInvalidReviewFields", err)
	}
	if _, err := s.Approve(context.Background(), 1, 103, 6, 0); !errors.Is(err, ErrInvalidReviewFields) {
		t.Fatalf("err=%v, want ErrInvalidReviewFields", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ApproveConflict(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// 状态非 PENDING → 无行
	mock.ExpectQuery(`SELECT name, phone, submitted_at FROM worker_registrations`).
		WithArgs(int64(1), RegStatusPending).
		WillReturnError(pgx.ErrNoRows)

	s := NewPGStore(mock)
	_, err = s.Approve(context.Background(), 1, 103, 6, 4)
	if !errors.Is(err, ErrRegistrationConflict) {
		t.Fatalf("err=%v, want ErrRegistrationConflict", err)
	}
}

func TestPGStore_Reject(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectExec(`UPDATE worker_registrations`).
		WithArgs(RegStatusRejected, "证件存疑", int64(103), int64(1), RegStatusPending).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	s := NewPGStore(mock)
	if err := s.Reject(context.Background(), 1, 103, "证件存疑"); err != nil {
		t.Fatalf("Reject: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_SubmitRealName(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO verifications\(subject_type, subject_id, method, real_name, id_card_no, result, verified_at\)`).
		WithArgs(int64(9), "证件OCR", "王师傅", "110101199001011234", RealNamePending, pgxmock.AnyArg()).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(3)))

	s := NewPGStore(mock)
	id, err := s.SubmitRealName(context.Background(), WorkerRealNameVerification{
		WorkerID: 9, Method: "证件OCR", RealName: "王师傅", IDCardNo: "110101199001011234", Result: RealNamePending,
	})
	if err != nil {
		t.Fatalf("SubmitRealName: %v", err)
	}
	if id != 3 {
		t.Fatalf("id=%d, want 3", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_GetLatest(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "worker_id", "method", "real_name", "id_card_no", "result", "verified_at", "operator_account_id", "operator_name"}
	va := time.Date(2026, 8, 19, 8, 1, 0, 0, time.UTC)
	mock.ExpectQuery(`SELECT id, subject_id, method, real_name, id_card_no, result, verified_at, operator_account_id, operator_name FROM verifications`).
		WithArgs(int64(9)).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(3), int64(9), "证件OCR", "王师傅", "110101199001011234", RealNamePass, va, int64(103), "admin"))

	s := NewPGStore(mock)
	v, err := s.GetLatest(context.Background(), 9)
	if err != nil {
		t.Fatalf("GetLatest: %v", err)
	}
	if v.Result != RealNamePass || v.OperatorAccountID != 103 || v.OperatorName != "admin" {
		t.Fatalf("v=%+v", v)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_GetLatestNotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, subject_id, method, real_name, id_card_no, result, verified_at, operator_account_id, operator_name FROM verifications`).
		WithArgs(int64(9)).
		WillReturnError(pgx.ErrNoRows)

	s := NewPGStore(mock)
	_, err = s.GetLatest(context.Background(), 9)
	if !errors.Is(err, ErrRealNameNotFound) {
		t.Fatalf("err=%v, want ErrRealNameNotFound", err)
	}
}

func TestPGStore_Verify(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectExec(`UPDATE verifications SET result=\$1, operator_account_id=\$2, operator_name=\$3, verified_at=now\(\)`).
		WithArgs(RealNamePass, int64(103), "admin", int64(9), RealNamePending).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	s := NewPGStore(mock)
	if err := s.Verify(context.Background(), 9, RealNamePass, "admin", 103); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
