package aaa

// T2 管理端建号域级测试:loid 预查/offer PUBLISHED 门禁/qos 软引用/平台法人兜底。

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestPGStore_CreateLoAccountChecked(t *testing.T) {
	t.Run("成功:显式法人", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lo_accounts WHERE loid = \$1\)`).
			WithArgs("LOID-NEW").WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(false))
		mock.ExpectQuery(`SELECT status FROM product_offers WHERE id = \$1`).
			WithArgs(int64(2)).WillReturnRows(mock.NewRows([]string{"status"}).AddRow("PUBLISHED"))
		mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(1)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(4)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(2)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(1)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`INSERT INTO lo_accounts`).
			WithArgs("LOID-NEW", int64(4), int64(1), "主品牌", int64(11), "马尼拉", nil, int64(2), int64(1), "ACTIVE", "POSTPAID").
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(9)))
		id, err := NewPGStore(mock).CreateLoAccountChecked(context.Background(), LoAccount{
			Loid: "LOID-NEW", CustomerID: 4, LegalEntityID: 1, LegalEntityName: "主品牌",
			RegionID: 11, RegionName: "马尼拉", OfferID: 2, QosTemplateID: 1, Status: "ACTIVE",
		})
		if err != nil || id != 9 {
			t.Fatalf("id=%d err=%v", id, err)
		}
	})

	t.Run("loid 撞库 → ErrDuplicate", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lo_accounts WHERE loid = \$1\)`).
			WithArgs("LOID-DUP").WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		_, err := NewPGStore(mock).CreateLoAccountChecked(context.Background(), LoAccount{
			Loid: "LOID-DUP", CustomerID: 4, OfferID: 2, QosTemplateID: 1,
		})
		if !errors.Is(err, ErrDuplicate) {
			t.Fatalf("err=%v, want ErrDuplicate", err)
		}
	})

	t.Run("offer 非 PUBLISHED → ErrOfferNotPublished", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lo_accounts WHERE loid = \$1\)`).
			WithArgs("LOID-2").WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(false))
		mock.ExpectQuery(`SELECT status FROM product_offers WHERE id = \$1`).
			WithArgs(int64(2)).WillReturnRows(mock.NewRows([]string{"status"}).AddRow("DRAFT"))
		_, err := NewPGStore(mock).CreateLoAccountChecked(context.Background(), LoAccount{
			Loid: "LOID-2", CustomerID: 4, OfferID: 2, QosTemplateID: 1,
		})
		if !errors.Is(err, ErrOfferNotPublished) {
			t.Fatalf("err=%v, want ErrOfferNotPublished", err)
		}
	})

	t.Run("qos 模板缺失 → ErrForeignKeyViolation", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lo_accounts WHERE loid = \$1\)`).
			WithArgs("LOID-3").WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(false))
		mock.ExpectQuery(`SELECT status FROM product_offers WHERE id = \$1`).
			WithArgs(int64(2)).WillReturnRows(mock.NewRows([]string{"status"}).AddRow("PUBLISHED"))
		mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(99)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(false))
		_, err := NewPGStore(mock).CreateLoAccountChecked(context.Background(), LoAccount{
			Loid: "LOID-3", CustomerID: 4, OfferID: 2, QosTemplateID: 99,
		})
		if !errors.Is(err, ErrForeignKeyViolation) {
			t.Fatalf("err=%v, want ErrForeignKeyViolation", err)
		}
	})

	t.Run("法人缺省兜底平台总公司", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM lo_accounts WHERE loid = \$1\)`).
			WithArgs("LOID-4").WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(false))
		mock.ExpectQuery(`SELECT status FROM product_offers WHERE id = \$1`).
			WithArgs(int64(2)).WillReturnRows(mock.NewRows([]string{"status"}).AddRow("PUBLISHED"))
		mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(1)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`SELECT id, name FROM legal_entities WHERE is_platform = TRUE LIMIT 1`).
			WillReturnRows(mock.NewRows([]string{"id", "name"}).AddRow(int64(9), "平台总公司"))
		mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(4)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(2)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`SELECT EXISTS`).WithArgs(int64(9)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
		mock.ExpectQuery(`INSERT INTO lo_accounts`).
			WithArgs("LOID-4", int64(4), int64(9), "平台总公司", int64(0), "", nil, int64(2), int64(1), "ACTIVE", "POSTPAID").
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(10)))
		id, err := NewPGStore(mock).CreateLoAccountChecked(context.Background(), LoAccount{
			Loid: "LOID-4", CustomerID: 4, OfferID: 2, QosTemplateID: 1, Status: "ACTIVE",
		})
		if err != nil || id != 10 {
			t.Fatalf("id=%d err=%v", id, err)
		}
	})
}
