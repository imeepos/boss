package apikey

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
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
		WithArgs(int64(1), "ci-pipeline", pgxmock.AnyArg(), int64(1)).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(10)))

	s := NewPGStore(mock)
	createdBy := int64(1)
	res, err := s.Create(context.Background(), 1, createdBy, "ci-pipeline")
	if err != nil {
		t.Fatalf("Create: %v", err)
	}
	if res.ID != 10 || res.AccountID != 1 || res.Name != "ci-pipeline" || res.PlainKey == "" {
		t.Fatalf("unexpected result: %+v", res)
	}
	if len(res.PlainKey) != 37 || res.PlainKey[:5] != "boss_" {
		t.Fatalf("bad key format: %q", res.PlainKey)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_List(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "account_id", "real_name", "name", "status", "last_used_at", "expires_at", "created_at"}
	now := time.Now()
	mock.ExpectQuery(`SELECT k.id, k.account_id, a.real_name, k.name, k.status, k.last_used_at, k.expires_at, k.created_at`).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), int64(1), "admin", "ci-pipeline", int16(1), now, nil, now))

	s := NewPGStore(mock)
	list, err := s.List(context.Background())
	if err != nil {
		t.Fatalf("List: %v", err)
	}
	if len(list) != 1 || list[0].ID != 1 || list[0].AccountName != "admin" || list[0].Name != "ci-pipeline" {
		t.Fatalf("unexpected list: %+v", list)
	}
	if list[0].LastUsedAt == "" || list[0].CreatedAt == "" {
		t.Fatalf("missing timestamps: %+v", list[0])
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

	mock.ExpectExec(`UPDATE api_keys SET status=0 WHERE`).
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

	mock.ExpectExec(`UPDATE api_keys SET status=0 WHERE`).
		WithArgs(int64(999)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))

	s := NewPGStore(mock)
	if err := s.Revoke(context.Background(), 999); err != ErrNotFound {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_Lookup(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	h := sha256.Sum256([]byte("boss_test_key"))
	hash := hex.EncodeToString(h[:])

	mock.ExpectQuery(`SELECT account_id FROM api_keys`).
		WithArgs(hash).
		WillReturnRows(mock.NewRows([]string{"account_id"}).AddRow(int64(3)))

	s := NewPGStore(mock)
	aid, err := s.Lookup(context.Background(), hash)
	if err != nil {
		t.Fatalf("Lookup: %v", err)
	}
	if aid != 3 {
		t.Fatalf("account_id=%d want 3", aid)
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

	h := sha256.Sum256([]byte("nonexistent"))
	hash := hex.EncodeToString(h[:])
	mock.ExpectQuery(`SELECT account_id FROM api_keys`).
		WithArgs(hash).
		WillReturnError(pgx.ErrNoRows)

	s := NewPGStore(mock)
	_, err = s.Lookup(context.Background(), hash)
	if err != ErrNotFound {
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

	h := sha256.Sum256([]byte("boss_test_key"))
	hash := hex.EncodeToString(h[:])
	mock.ExpectExec(`UPDATE api_keys SET last_used_at`).
		WithArgs(hash).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	s := NewPGStore(mock)
	// Touch 不应返回错误(尽力而为)
	s.Touch(context.Background(), hash)
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}