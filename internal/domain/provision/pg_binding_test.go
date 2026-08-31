package provision

// 绑定 CRUD 单测(pgxmock):查询/绑定改绑/解绑 + 校验(同法人/禁用模板/不存在)。

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestPGStore_GetOfferBinding(t *testing.T) {
	ctx := context.Background()

	t.Run("已绑定返回绑定行", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT b\.id, b\.legal_entity_id.*FROM offer_provision_bindings b`).
			WithArgs(int64(101)).
			WillReturnRows(mock.NewRows([]string{"id", "legal_entity_id", "offer_id", "template_id", "code", "name", "remark"}).
				AddRow(int64(1), int64(1), int64(101), int64(142), "TPL-FTTH-100M", "FTTH 100M开通", ""))

		s := NewPGStore(mock)
		b, err := s.GetOfferBinding(ctx, 101)
		if err != nil || b == nil {
			t.Fatalf("binding=%+v err=%v", b, err)
		}
		if b.OfferID != 101 || b.TemplateID != 142 || b.TemplateCode != "TPL-FTTH-100M" {
			t.Fatalf("binding=%+v", b)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("未绑定返回 nil,nil", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT b\.id, b\.legal_entity_id.*FROM offer_provision_bindings b`).
			WithArgs(int64(999)).
			WillReturnRows(pgxmock.NewRows([]string{"id", "legal_entity_id", "offer_id", "template_id", "code", "name", "remark"}))

		s := NewPGStore(mock)
		b, err := s.GetOfferBinding(ctx, 999)
		if err != nil || b != nil {
			t.Fatalf("want nil/nil, got binding=%+v err=%v", b, err)
		}
	})
}

func TestPGStore_UpsertOfferBinding(t *testing.T) {
	ctx := context.Background()
	const validateSQL = `SELECT o\.legal_entity_id, t\.legal_entity_id, t\.status`

	t.Run("同法人启用模板:绑定成功", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(validateSQL).
			WithArgs(int64(101), int64(142)).
			WillReturnRows(mock.NewRows([]string{"entity", "entity", "status"}).AddRow(int64(1), int64(1), "ENABLED"))
		mock.ExpectQuery(`INSERT INTO offer_provision_bindings`).
			WithArgs(int64(1), int64(101), int64(142), "").
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(5)))

		s := NewPGStore(mock)
		id, err := s.UpsertOfferBinding(ctx, 101, 142, "")
		if err != nil || id != 5 {
			t.Fatalf("id=%d err=%v", id, err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("跨法人:拒绝绑定", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(validateSQL).
			WithArgs(int64(101), int64(152)).
			WillReturnRows(mock.NewRows([]string{"entity", "entity", "status"}).AddRow(int64(1), int64(6), "ENABLED"))

		s := NewPGStore(mock)
		_, err := s.UpsertOfferBinding(ctx, 101, 152, "")
		if err == nil || !errors.Is(err, ErrBindingInvalid) || !strings.Contains(err.Error(), "entity mismatch") {
			t.Fatalf("want ErrBindingInvalid entity mismatch, got %v", err)
		}
	})

	t.Run("禁用模板:拒绝绑定", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(validateSQL).
			WithArgs(int64(101), int64(142)).
			WillReturnRows(mock.NewRows([]string{"entity", "entity", "status"}).AddRow(int64(1), int64(1), "DISABLED"))

		s := NewPGStore(mock)
		_, err := s.UpsertOfferBinding(ctx, 101, 142, "")
		if err == nil || !errors.Is(err, ErrBindingInvalid) || !strings.Contains(err.Error(), "disabled template") {
			t.Fatalf("want ErrBindingInvalid disabled template, got %v", err)
		}
	})

	t.Run("套餐或模板不存在:外键错误", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(validateSQL).
			WithArgs(int64(999), int64(142)).
			WillReturnRows(pgxmock.NewRows([]string{"entity", "entity", "status"}))

		s := NewPGStore(mock)
		_, err := s.UpsertOfferBinding(ctx, 999, 142, "")
		if err == nil || !strings.Contains(err.Error(), "foreign key violation") {
			t.Fatalf("want foreign key violation, got %v", err)
		}
	})
}

func TestPGStore_DeleteOfferBinding(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectExec(`DELETE FROM offer_provision_bindings`).
		WithArgs(int64(101)).
		WillReturnResult(pgxmock.NewResult("DELETE", 1))

	s := NewPGStore(mock)
	if err := s.DeleteOfferBinding(context.Background(), 101); err != nil {
		t.Fatalf("DeleteOfferBinding: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
