package resource

import (
	"context"
	"errors"
	"strings"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// 扩容单写侧:执行(批量建端口)/驳回(TDD,随 pg_expand.go 同提交)。

// seedExec happy-path 前置:PENDING 扩容单 EXP-1(法人 1,区域 5,预期 3)+ 设备 SPL-01(已有 2 口)。
func seedExec(t *testing.T, mock pgxmock.PgxPoolIface) {
	t.Helper()
	mock.ExpectQuery(`SELECT legal_entity_id, region_id, expected_ports, status FROM expansions`).
		WithArgs("EXP-1").
		WillReturnRows(mock.NewRows([]string{"legal_entity_id", "region_id", "expected_ports", "status"}).
			AddRow(int64(1), int64(5), int32(3), "PENDING"))
	mock.ExpectQuery(`FROM resources WHERE id`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows([]string{"id", "legal_entity_id", "code", "name", "type", "parent_id", "address_id", "status"}).
			AddRow(int64(7), int64(1), "SPL-01", "分光器01", "SPLITTER", int64(0), int64(3), "ONLINE"))
	mock.ExpectQuery(`SELECT name FROM legal_entities WHERE id`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"name"}).AddRow("主品牌·企业"))
	mock.ExpectQuery(`SELECT name FROM regions WHERE id`).
		WithArgs(int64(5)).
		WillReturnRows(mock.NewRows([]string{"name"}).AddRow("NCR"))
	rows := mock.NewRows([]string{"id", "port_code", "quad_code", "resource_id", "legal_entity_id", "legal_entity_name", "address_id", "region_id", "region_name", "order_id", "status"})
	rows.AddRow(int64(1), "P-SPL01-01", "P-SPL01-01", int64(7), int64(1), "主品牌·企业", int64(3), int64(5), "NCR", int64(0), "IDLE")
	rows.AddRow(int64(2), "P-SPL01-02", "P-SPL01-02", int64(7), int64(1), "主品牌·企业", int64(3), int64(5), "NCR", int64(0), "IDLE")
	mock.ExpectQuery(`FROM ports`).WithArgs(int64(7)).WillReturnRows(rows)
}

func TestPGStore_ExecuteExpansion(t *testing.T) {
	t.Run("续号补建 1 口后 DONE", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		seedExec(t, mock)
		mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(7)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(3)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`INSERT INTO ports`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(10)))
		mock.ExpectExec(`UPDATE expansions SET status = 'DONE'`).
			WithArgs("EXP-1").WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		s := NewPGStore(mock)
		res, err := s.ExecuteExpansion(context.Background(), "EXP-1", 7)
		if err != nil {
			t.Fatalf("ExecuteExpansion: %v", err)
		}
		if res.Created != 1 || res.TotalPorts != 3 || len(res.PortCodes) != 1 || res.PortCodes[0] != "P-SPL01-03" {
			t.Fatalf("res=%+v, want created=1 total=3 codes=[P-SPL01-03]", res)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("已建满直接完成不建口", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT legal_entity_id, region_id, expected_ports, status FROM expansions`).
			WithArgs("EXP-1").
			WillReturnRows(mock.NewRows([]string{"legal_entity_id", "region_id", "expected_ports", "status"}).
				AddRow(int64(1), int64(5), int32(2), "PENDING"))
		mock.ExpectQuery(`FROM resources WHERE id`).
			WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"id", "legal_entity_id", "code", "name", "type", "parent_id", "address_id", "status"}).
				AddRow(int64(7), int64(1), "SPL-01", "分光器01", "SPLITTER", int64(0), int64(3), "ONLINE"))
		mock.ExpectQuery(`SELECT name FROM legal_entities WHERE id`).
			WithArgs(int64(1)).WillReturnRows(mock.NewRows([]string{"name"}).AddRow("主品牌·企业"))
		mock.ExpectQuery(`SELECT name FROM regions WHERE id`).
			WithArgs(int64(5)).WillReturnRows(mock.NewRows([]string{"name"}).AddRow("NCR"))
		rows := mock.NewRows([]string{"id", "port_code", "quad_code", "resource_id", "legal_entity_id", "legal_entity_name", "address_id", "region_id", "region_name", "order_id", "status"})
		rows.AddRow(int64(1), "P-SPL01-01", "P-SPL01-01", int64(7), int64(1), "主品牌·企业", int64(3), int64(5), "NCR", int64(0), "IDLE")
		rows.AddRow(int64(2), "P-SPL01-02", "P-SPL01-02", int64(7), int64(1), "主品牌·企业", int64(3), int64(5), "NCR", int64(0), "IDLE")
		mock.ExpectQuery(`FROM ports`).WithArgs(int64(7)).WillReturnRows(rows)
		mock.ExpectExec(`UPDATE expansions SET status = 'DONE'`).
			WithArgs("EXP-1").WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		s := NewPGStore(mock)
		res, err := s.ExecuteExpansion(context.Background(), "EXP-1", 7)
		if err != nil {
			t.Fatalf("ExecuteExpansion: %v", err)
		}
		if res.Created != 0 || res.TotalPorts != 2 {
			t.Fatalf("res=%+v, want created=0 total=2", res)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("非 PENDING 单拒绝执行", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT legal_entity_id, region_id, expected_ports, status FROM expansions`).
			WithArgs("EXP-1").
			WillReturnRows(mock.NewRows([]string{"legal_entity_id", "region_id", "expected_ports", "status"}).
				AddRow(int64(1), int64(5), int32(3), "DONE"))
		s := NewPGStore(mock)
		if _, err := s.ExecuteExpansion(context.Background(), "EXP-1", 7); !errors.Is(err, ErrIllegalTransition) {
			t.Fatalf("err=%v, want ErrIllegalTransition", err)
		}
	})

	t.Run("设备法人不符拒绝执行", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT legal_entity_id, region_id, expected_ports, status FROM expansions`).
			WithArgs("EXP-1").
			WillReturnRows(mock.NewRows([]string{"legal_entity_id", "region_id", "expected_ports", "status"}).
				AddRow(int64(1), int64(5), int32(3), "PENDING"))
		mock.ExpectQuery(`FROM resources WHERE id`).
			WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"id", "legal_entity_id", "code", "name", "type", "parent_id", "address_id", "status"}).
				AddRow(int64(7), int64(2), "SPL-01", "分光器01", "SPLITTER", int64(0), int64(3), "ONLINE"))
		s := NewPGStore(mock)
		_, err := s.ExecuteExpansion(context.Background(), "EXP-1", 7)
		if !errors.Is(err, ErrForeignKeyViolation) {
			t.Fatalf("err=%v, want ErrForeignKeyViolation", err)
		}
	})

	t.Run("建口失败留 PENDING 可重试", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		seedExec(t, mock)
		mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(7)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(3)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`INSERT INTO ports`).WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).WillReturnError(errors.New("unique violation"))
		s := NewPGStore(mock)
		_, err := s.ExecuteExpansion(context.Background(), "EXP-1", 7)
		if err == nil || !strings.Contains(err.Error(), "expansion build port") {
			t.Fatalf("err=%v, want build port failure", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("扩容单不存在", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`FROM expansions`).
			WithArgs("EXP-X").
			WillReturnRows(mock.NewRows([]string{"legal_entity_id", "region_id", "expected_ports", "status"}))
		s := NewPGStore(mock)
		if _, err := s.ExecuteExpansion(context.Background(), "EXP-X", 7); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v, want ErrNotFound", err)
		}
	})
}

func TestPGStore_RejectExpansion(t *testing.T) {
	t.Run("PENDING→DONE 终态", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE expansions SET status = 'DONE'`).
			WithArgs("EXP-1").WillReturnResult(pgxmock.NewResult("UPDATE", 1))
		s := NewPGStore(mock)
		if err := s.RejectExpansion(context.Background(), "EXP-1"); err != nil {
			t.Fatalf("RejectExpansion: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("单号不存在", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectExec(`UPDATE expansions`).
			WithArgs("EXP-X").WillReturnResult(pgxmock.NewResult("UPDATE", 0))
		s := NewPGStore(mock)
		if err := s.RejectExpansion(context.Background(), "EXP-X"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v, want ErrNotFound", err)
		}
	})
}
