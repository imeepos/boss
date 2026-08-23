package apikey

import (
	"context"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func TestPGStore_Create(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO api_keys`).
		WithArgs("worker", int64(5), "field-worker", pgxmock.AnyArg(), int64(1), "").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(10)))

	s := NewPGStore(mock)
	res, err := s.Create(context.Background(), SubjectWorker, 5, 1, "field-worker", "")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if res.ID != 10 || res.SubjectType != "worker" || res.SubjectRef != 5 || res.PlainKey == "" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if len(res.PlainKey) != 37 || res.PlainKey[:5] != "boss_" {
		t.Fatalf("bad key format: %q", res.PlainKey)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CreateInvalidSubject(t *testing.T) {
	s := NewPGStore(nil)
	if _, err := s.Create(context.Background(), "hacker", 1, 1, "x", ""); err != ErrInvalidSubject {
		t.Fatalf("expected ErrInvalidSubject, got %v", err)
	}
}

func TestPGStore_List(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "subject_type", "subject_ref", "subject_name", "name", "status", "last_used_at", "created_at", "template_code"}
	now := time.Now()
	mock.ExpectQuery(`SELECT k.id, k.subject_type`).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), "account", int64(1), "admin", "ci", int16(1), nil, now, "").
			AddRow(int64(2), "worker", int64(5), "张师傅", "field", int16(1), now, now, "partner-orders-read"))

	s := NewPGStore(mock)
	list, err := s.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 2 || list[0].SubjectType != "account" || list[1].SubjectName != "张师傅" {
		t.Fatalf("unexpected list: %+v", list)
	}
	if list[1].LastUsedAt == "" || list[0].CreatedAt == "" {
		t.Fatalf("missing timestamps: %+v", list)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_Revoke(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectExec(`UPDATE api_keys SET status=0`).
		WithArgs(int64(5)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	s := NewPGStore(mock)
	if err := s.Revoke(context.Background(), 5); err != nil {
		t.Fatalf("Revoke: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_RevokeNotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectExec(`UPDATE api_keys SET status=0`).
		WithArgs(int64(999)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	s := NewPGStore(mock)
	if err := s.Revoke(context.Background(), 999); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestPGStore_Lookup(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT subject_type, subject_ref, COALESCE\(template_code, ''\) FROM api_keys`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnRows(mock.NewRows([]string{"subject_type", "subject_ref", "template_code"}).AddRow("worker", int64(5), "partner-orders-read"))

	s := NewPGStore(mock)
	subj, err := s.Lookup(context.Background(), "abc")
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if subj.Type != "worker" || subj.Ref != 5 {
		t.Fatalf("subject=%+v", subj)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_LookupNotFound(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT subject_type, subject_ref, COALESCE\(template_code, ''\) FROM api_keys`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnError(pgx.ErrNoRows)

	s := NewPGStore(mock)
	if _, err = s.Lookup(context.Background(), "nonexistent"); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_Touch(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectExec(`UPDATE api_keys SET last_used_at`).
		WithArgs(pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	s := NewPGStore(mock)
	s.Touch(context.Background(), "abc")
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestValidSubjectType(t *testing.T) {
	for _, ok := range []string{"account", "worker", "customer"} {
		if !ValidSubjectType(ok) {
			t.Fatalf("%q should be valid", ok)
		}
	}
	for _, bad := range []string{"", "admin", "Account"} {
		if ValidSubjectType(bad) {
			t.Fatalf("%q should be invalid", bad)
		}
	}
}
