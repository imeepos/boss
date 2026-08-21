package customer

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestPGStore_SubmitRegistration(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// FK validation: legal entity exists
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	// FK validation: address exists
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(100)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	// FK validation: region exists
	mock.ExpectQuery(`SELECT EXISTS`).
		WithArgs(int64(4)).
		WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))

	mock.ExpectQuery(`INSERT INTO customer_registrations\(name, phone, id_card_no, legal_entity_id, address_id, region_id\)`).
		WithArgs("张先生", "13800001234", "110101199001011234", int64(1), int64(100), int64(4)).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(9)))

	s := NewPGStore(mock)
	id, err := s.Submit(context.Background(), Registration{
		Name: "张先生", Phone: "13800001234", IDCardNo: "110101199001011234",
		LegalEntityID: 1, AddressID: 100, RegionID: 4,
	})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if id != 9 {
		t.Fatalf("id=%d, want 9", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ApproveRegistration_CreatesCustomer(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// 1) 读申请
	mock.ExpectQuery(`SELECT name, phone, id_card_no, legal_entity_id, address_id, region_id FROM customer_registrations WHERE id = \$1 AND status = \$2`).
		WithArgs(int64(5), RegStatusPending).
		WillReturnRows(mock.NewRows([]string{"name", "phone", "id_card_no", "legal_entity_id", "address_id", "region_id"}).
			AddRow("张先生", "13800001234", "110101199001011234", int64(1), int64(100), int64(4)))
	// 2) 区域名快照
	mock.ExpectQuery(`SELECT name FROM regions WHERE id = \$1`).
		WithArgs(int64(4)).
		WillReturnRows(mock.NewRows([]string{"name"}).AddRow("马尼拉市"))
	// 3) 建客户主档
	mock.ExpectQuery(`INSERT INTO customers\(name, phone, id_type, id_no, real_name_status, service_status, address_id, legal_entity_id, region_id, region_name\)`).
		WithArgs("张先生", "13800001234", "110101199001011234",
			int64(100), int64(1), int64(4), "马尼拉市").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(88)))
	// 4) 回填申请
	mock.ExpectExec(`UPDATE customer_registrations SET status=\$1, reviewer_account_id=\$2, customer_id=\$3, reviewed_at=now\(\) WHERE id=\$4 AND status=\$5`).
		WithArgs(RegStatusApproved, int64(1000), int64(88), int64(5), RegStatusPending).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	s := NewPGStore(mock)
	customerID, err := s.Approve(context.Background(), 5, 1000)
	if err != nil {
		t.Fatalf("Approve: %v", err)
	}
	if customerID != 88 {
		t.Fatalf("customerID=%d, want 88", customerID)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_Verify_PassSyncsStatus(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectExec(`UPDATE verifications SET result=\$1, reject_reason=\$2, operator_account_id=\$3, operator_name=\$4, verified_at=now\(\) WHERE subject_type='customer' AND subject_id=\$5 AND result=\$6`).
		WithArgs(RealNamePass, "", int64(1000), "admin", int64(88), RealNamePending).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	// PASS 同步 customers.real_name_status=VERIFIED
	mock.ExpectExec(`UPDATE customers SET real_name_status='VERIFIED' WHERE id=\$1`).
		WithArgs(int64(88)).WillReturnResult(pgxmock.NewResult("UPDATE", 1))

	s := NewPGStore(mock)
	if err := s.Verify(context.Background(), 88, RealNamePass, "", "admin", 1000); err != nil {
		t.Fatalf("Verify: %v", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
