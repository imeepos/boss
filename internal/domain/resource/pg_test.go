package resource

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func TestPGStore_ListResources(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "legal_entity_id", "code", "name", "type", "parent_id", "address_id", "status"}
	mock.ExpectQuery(`SELECT id, legal_entity_id, code, name, type, COALESCE\(parent_id, 0\), address_id, status FROM resources`).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), int64(1), "OLT-01", "望京OLT-01", "OLT", int64(0), int64(100), "ONLINE").
			AddRow(int64(2), int64(1), "SPL-01", "望京分光器-01", "SPLITTER", int64(1), int64(100), "ONLINE"))

	s := NewPGStore(mock)
	got, err := s.ListResources(context.Background())
	if err != nil {
		t.Fatalf("ListResources: %v", err)
	}
	if len(got) != 2 || got[1].ParentID != 1 {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CreateResource(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// FK validation: address exists
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(100)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	// FK validation: parent exists(T2 增:parentId 存在性防 23503)
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectQuery(`INSERT INTO resources`).
		WithArgs(int64(1), "SPL-02", "望京分光器-02", "SPLITTER", int64(1), int64(100), "ONLINE").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(3)))

	s := NewPGStore(mock)
	id, err := s.CreateResource(context.Background(), Resource{
		LegalEntityID: 1, Code: "SPL-02", Name: "望京分光器-02", Type: "SPLITTER", ParentID: 1, AddressID: 100, Status: "ONLINE",
	})
	if err != nil {
		t.Fatalf("CreateResource: %v", err)
	}
	if id != 3 {
		t.Fatalf("id=%d, want 3", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_GetResource(t *testing.T) {
	t.Run("命中", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		cols := []string{"id", "legal_entity_id", "code", "name", "type", "parent_id", "address_id", "status"}
		mock.ExpectQuery(`SELECT id, legal_entity_id, code, name, type, COALESCE\(parent_id, 0\)`).
			WithArgs(int64(1)).
			WillReturnRows(mock.NewRows(cols).
				AddRow(int64(1), int64(1), "OLT-01", "望京OLT-01", "OLT", int64(0), int64(100), "ONLINE"))

		s := NewPGStore(mock)
		r, err := s.GetResource(context.Background(), 1)
		if err != nil {
			t.Fatalf("GetResource: %v", err)
		}
		if r.Code != "OLT-01" {
			t.Fatalf("r=%+v", r)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("未命中", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT id, legal_entity_id, code`).
			WithArgs(int64(99)).
			WillReturnError(pgx.ErrNoRows)

		s := NewPGStore(mock)
		_, err = s.GetResource(context.Background(), 99)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v, want ErrNotFound", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}

func TestPGStore_ListPorts(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "port_code", "quad_code", "resource_id", "legal_entity_id", "legal_entity_name", "address_id", "region_id", "region_name", "order_id", "status"}
	mock.ExpectQuery(`SELECT id, port_code, quad_code, resource_id, legal_entity_id`).
		WithArgs(int64(2)).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), "P-SPL01-01", "P-SPL01-01", int64(2), int64(1), "主品牌·企业", int64(100), int64(11), "马尼拉市", int64(0), "IDLE"))

	s := NewPGStore(mock)
	got, err := s.ListPorts(context.Background(), 2)
	if err != nil {
		t.Fatalf("ListPorts: %v", err)
	}
	if len(got) != 1 || got[0].Status != "IDLE" || got[0].OrderID != 0 {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CreatePort(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// quad_code 唯一预查(T2 增:无 DB 索引,预查给 40900 语义)
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM ports WHERE quad_code = \$1\)`).
		WithArgs("P-SPL01-02").
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(false))
	// FK validation: resource exists
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(2)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	// FK validation: address exists
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(100)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectQuery(`INSERT INTO ports`).
		WithArgs("P-SPL01-02", "P-SPL01-02", int64(2), int64(1), "主品牌·企业", int64(100), int64(11), "马尼拉市", nil, "IDLE").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock)
	id, err := s.CreatePort(context.Background(), Port{
		PortCode: "P-SPL01-02", QuadCode: "P-SPL01-02", ResourceID: 2, LegalEntityID: 1, LegalEntityName: "主品牌·企业",
		AddressID: 100, RegionID: 11, RegionName: "马尼拉市", Status: "IDLE",
	})
	if err != nil {
		t.Fatalf("CreatePort: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ReservePort(t *testing.T) {
	t.Run("可预占", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectExec(`UPDATE ports SET status = 'RESERVED', order_id`).
			WithArgs(int64(1), int64(1001)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		s := NewPGStore(mock)
		if err := s.ReservePort(context.Background(), 1, 1001); err != nil {
			t.Fatalf("ReservePort: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("非IDLE不可预占", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectExec(`UPDATE ports SET status = 'RESERVED', order_id`).
			WithArgs(int64(1), int64(1001)).
			WillReturnResult(pgxmock.NewResult("UPDATE", 0))

		s := NewPGStore(mock)
		err = s.ReservePort(context.Background(), 1, 1001)
		if !errors.Is(err, ErrPortNotAvailable) {
			t.Fatalf("err=%v, want ErrPortNotAvailable", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}

// TestPGStore_ReserveFirstAvailable 契约:按地址预占一个空闲端口并返回 id;无空闲返回 ErrPortNotAvailable。
func TestPGStore_ReserveFirstAvailable(t *testing.T) {
	t.Run("有空闲", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`UPDATE ports SET status = 'RESERVED', order_id`).
			WithArgs(int64(100), int64(1001)).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(7)))

		s := NewPGStore(mock)
		id, err := s.ReserveFirstAvailable(context.Background(), 100, 1001)
		if err != nil {
			t.Fatalf("ReserveFirstAvailable: %v", err)
		}
		if id != 7 {
			t.Fatalf("id=%d, want 7", id)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("无空闲", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`UPDATE ports SET status = 'RESERVED', order_id`).
			WithArgs(int64(200), int64(1001)).
			WillReturnError(pgx.ErrNoRows)

		s := NewPGStore(mock)
		_, err = s.ReserveFirstAvailable(context.Background(), 200, 1001)
		if !errors.Is(err, ErrPortNotAvailable) {
			t.Fatalf("err=%v, want ErrPortNotAvailable", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}
