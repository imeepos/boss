package quadlink

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

var cols = []string{"id", "asset_id", "customer_id", "port_id", "address_id", "legal_entity_id", "legal_entity_name", "status"}

func TestPGStore_ListLinks(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, COALESCE\(asset_id, 0\), customer_id, port_id, address_id, legal_entity_id, legal_entity_name, status FROM quad_links`).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), int64(100), int64(1), int64(200), int64(300), int64(1), "主品牌·企业", "LINKED"))

	s := NewPGStore(mock)
	got, err := s.ListLinks(context.Background())
	if err != nil {
		t.Fatalf("ListLinks: %v", err)
	}
	if len(got) != 1 || got[0].Status != "LINKED" || got[0].CustomerID != 1 {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CreateLink(t *testing.T) {
	t.Run("正常创建 asset_id=0 不落资产", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		// 三码存在性校验(客户/端口/地址)。
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM customers WHERE id = \$1\)`).
			WithArgs(int64(2)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM ports WHERE id = \$1\)`).
			WithArgs(int64(201)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM addresses WHERE id = \$1\)`).
			WithArgs(int64(301)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		// 运营主体存在性 + 端口归属地址交叉校验(默认通过)。
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM legal_entities WHERE id = \$1\)`).
			WithArgs(int64(1)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM ports WHERE id = \$1 AND address_id = \$2\)`).
			WithArgs(int64(201), int64(301)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		// asset_id=0 → INSERT NULL(不查 assets)。
		mock.ExpectQuery(`INSERT INTO quad_links`).
			WithArgs(nil, int64(2), int64(201), int64(301), int64(1), "主品牌·企业", "LINKED").
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

		s := NewPGStore(mock)
		id, err := s.CreateLink(context.Background(), QuadLink{
			CustomerID: 2, PortID: 201, AddressID: 301, LegalEntityID: 1, LegalEntityName: "主品牌·企业", Status: "LINKED",
		})
		if err != nil {
			t.Fatalf("CreateLink: %v", err)
		}
		if id != 2 {
			t.Fatalf("id=%d, want 2", id)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("正常创建 asset_id 非零查资产存在性", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM customers WHERE id = \$1\)`).
			WithArgs(int64(2)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM ports WHERE id = \$1\)`).
			WithArgs(int64(201)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM addresses WHERE id = \$1\)`).
			WithArgs(int64(301)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM legal_entities WHERE id = \$1\)`).
			WithArgs(int64(1)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM ports WHERE id = \$1 AND address_id = \$2\)`).
			WithArgs(int64(201), int64(301)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM assets WHERE id = \$1\)`).
			WithArgs(int64(101)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`INSERT INTO quad_links`).
			WithArgs(int64(101), int64(2), int64(201), int64(301), int64(1), "主品牌·企业", "LINKED").
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

		s := NewPGStore(mock)
		id, err := s.CreateLink(context.Background(), QuadLink{
			AssetID: 101, CustomerID: 2, PortID: 201, AddressID: 301, LegalEntityID: 1, LegalEntityName: "主品牌·企业", Status: "LINKED",
		})
		if err != nil {
			t.Fatalf("CreateLink: %v", err)
		}
		if id != 2 {
			t.Fatalf("id=%d, want 2", id)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("某实体不存在则拒绝", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		// 客户存在。
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM customers WHERE id = \$1\)`).
			WithArgs(int64(2)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		// 端口 999 不存在 → 立即拒绝,后续不再查 address/INSERT。
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM ports WHERE id = \$1\)`).
			WithArgs(int64(999)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(false))

		s := NewPGStore(mock)
		_, err = s.CreateLink(context.Background(), QuadLink{
			CustomerID: 2, PortID: 999, AddressID: 301, LegalEntityID: 1, Status: "UNLINKED",
		})
		if !errors.Is(err, ErrForeignKeyViolation) {
			t.Fatalf("err=%v, want ErrForeignKeyViolation", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("customer_id 为零则立即拒绝不触DB", func(t *testing.T) {
		s := NewPGStore(nil) // 不需要 mock,不应触及 DB。
		_, err := s.CreateLink(context.Background(), QuadLink{
			CustomerID: 0, PortID: 201, AddressID: 301, LegalEntityID: 1, Status: "UNLINKED",
		})
		if !errors.Is(err, ErrForeignKeyViolation) {
			t.Fatalf("err=%v, want ErrForeignKeyViolation", err)
		}
	})

	t.Run("asset_id=0 且三码存在时成功(预绑定场景)", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM customers WHERE id = \$1\)`).
			WithArgs(int64(2)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM ports WHERE id = \$1\)`).
			WithArgs(int64(201)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM addresses WHERE id = \$1\)`).
			WithArgs(int64(301)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`INSERT INTO quad_links`).
			WithArgs(nil, int64(2), int64(201), int64(301), int64(1), "", "UNLINKED").
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(99)))

		s := NewPGStore(mock)
		id, err := s.CreateLink(context.Background(), QuadLink{
			CustomerID: 2, PortID: 201, AddressID: 301, LegalEntityID: 1, Status: "UNLINKED",
		})
		if err != nil {
			t.Fatalf("CreateLink: %v", err)
		}
		if id != 99 {
			t.Fatalf("id=%d, want 99", id)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}

func TestPGStore_GetByAsset(t *testing.T) {
	t.Run("命中", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectQuery(`SELECT id, COALESCE\(asset_id, 0\), customer_id, port_id, address_id, legal_entity_id, legal_entity_name, status FROM quad_links WHERE asset_id = \$1 AND asset_id IS NOT NULL`).
			WithArgs(int64(100)).
			WillReturnRows(mock.NewRows(cols).
				AddRow(int64(1), int64(100), int64(1), int64(200), int64(300), int64(1), "主品牌·企业", "LINKED"))

		s := NewPGStore(mock)
		q, err := s.GetByAsset(context.Background(), 100)
		if err != nil {
			t.Fatalf("GetByAsset: %v", err)
		}
		if q.CustomerID != 1 || q.PortID != 200 {
			t.Fatalf("q=%+v", q)
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

		mock.ExpectQuery(`WHERE asset_id = \$1`).
			WithArgs(int64(999)).
			WillReturnError(pgx.ErrNoRows)

		s := NewPGStore(mock)
		_, err = s.GetByAsset(context.Background(), 999)
		if !errors.Is(err, ErrNotFound) {
			t.Fatalf("err=%v, want ErrNotFound", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}

// 回归(000056 部分唯一索引): 同码多行历史时 getBy 必须取最新行(ORDER BY id DESC LIMIT 1)。
func TestPGStore_GetByLatestLifecycle(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`WHERE customer_id = \$1 AND customer_id IS NOT NULL ORDER BY id DESC LIMIT 1`).
		WithArgs(int64(7)).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(5), int64(105), int64(7), int64(205), int64(305), int64(1), "主品牌·企业", "LINKED"))

	s := NewPGStore(mock)
	q, err := s.GetByCustomer(context.Background(), 7)
	if err != nil {
		t.Fatalf("GetByCustomer: %v", err)
	}
	if q.ID != 5 {
		t.Fatalf("id=%d, want 5(最新行)", q.ID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_PurgeOrphans(t *testing.T) {
	t.Run("删除孤儿行返回条数", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectExec(`DELETE FROM quad_links ql`).
			WillReturnResult(pgxmock.NewResult("DELETE", 132))

		s := NewPGStore(mock)
		n, err := s.PurgeOrphans(context.Background())
		if err != nil {
			t.Fatalf("PurgeOrphans: %v", err)
		}
		if n != 132 {
			t.Fatalf("n=%d, want 132", n)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("无孤儿时返回 0", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectExec(`DELETE FROM quad_links ql`).
			WillReturnResult(pgxmock.NewResult("DELETE", 0))

		s := NewPGStore(mock)
		n, err := s.PurgeOrphans(context.Background())
		if err != nil {
			t.Fatalf("PurgeOrphans: %v", err)
		}
		if n != 0 {
			t.Fatalf("n=%d, want 0", n)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}
