package asset

// 建标签服务层测试(P2-W2-T1):法人存在性校验 + 编号/EPC 唯一冲突 40900 分类。

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"
)

func TestPGStore_CreateTag_LegalEntityMissing(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(9)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(false))

	s := NewPGStore(mock)
	_, err = s.CreateTag(context.Background(), Tag{LegalEntityID: 9, TagNo: "T-1", EpcCode: "E1", Band: "UHF"})
	if !errors.Is(err, ErrForeignKeyViolation) {
		t.Fatalf("err=%v, want ErrForeignKeyViolation", err)
	}
}

func TestPGStore_CreateTag_NoDuplicateConflict(t *testing.T) {
	cases := map[string]*pgconn.PgError{
		"tagNo": {Code: "23505", ConstraintName: "tags_tag_no_key"},
		"epc":   {Code: "23505", ConstraintName: "tags_epc_code_key"},
	}
	for name, pgErr := range cases {
		t.Run(name, func(t *testing.T) {
			mock, err := pgxmock.NewPool()
			if err != nil {
				t.Fatal(err)
			}
			defer mock.Close()

			mock.ExpectQuery(`SELECT EXISTS`).
				WithArgs(int64(1)).
				WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
			mock.ExpectBegin()
			mock.ExpectQuery(`INSERT INTO tags`).
				WithArgs(int64(1), "T-1", "E1", "UHF", nil, "", "").
				WillReturnError(pgErr)

			s := NewPGStore(mock)
			_, err = s.CreateTag(context.Background(), Tag{LegalEntityID: 1, TagNo: "T-1", EpcCode: "E1", Band: "UHF"})
			if !errors.Is(err, ErrCodeDuplicate) {
				t.Fatalf("err=%v, want ErrCodeDuplicate", err)
			}
		})
	}
}
