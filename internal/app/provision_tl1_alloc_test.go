package app

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func TestTL1Resolver_NoNMSEndpoint(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	t.Setenv("BOSS_PROVISION_TL1_ADDR", "")
	mock.ExpectQuery(`SELECT order_no, customer_id, legal_entity_id FROM orders WHERE id = \$1`).
		WithArgs(int64(9)).
		WillReturnRows(mock.NewRows([]string{"order_no", "customer_id", "legal_entity_id"}).AddRow("ORD-1", int64(77), int64(3)))
	mock.ExpectQuery(`SELECT loid FROM lo_accounts WHERE customer_id = \$1`).
		WithArgs(int64(77)).
		WillReturnRows(mock.NewRows([]string{"loid"}).AddRow("LOID-88A1"))
	mock.ExpectQuery(`SELECT content FROM provision_templates WHERE id = \$1`).
		WithArgs(int64(2)).
		WillReturnRows(mock.NewRows([]string{"content"}).AddRow([]byte(tl1Content)))
	mock.ExpectQuery(`SELECT id, resource_id, pon_frame, pon_slot, pon_port, onu_no`).WithArgs(int64(9)).
		WillReturnRows(mock.NewRows([]string{"id", "resource_id", "pon_frame", "pon_slot", "pon_port", "onu_no"}).AddRow(int64(88), int64(5), 0, 7, 5, nil))
	mock.ExpectQuery(`SELECT nms_oltid FROM resources WHERE id = \$1 AND type = 'OLT'`).
		WithArgs(int64(5)).
		WillReturnRows(mock.NewRows([]string{"nms_oltid"}).AddRow("OLT-1"))
	// provision_nms 无行(返回 no rows error),env 也缺 → FAIL
	mock.ExpectQuery(`SELECT host, port, username, pass_cipher FROM provision_nms WHERE legal_entity_id = \$1`).
		WithArgs(int64(3)).
		WillReturnError(pgx.ErrNoRows)

	r := &TL1ParamResolver{db: mock}
	_, err := r.Resolve(context.Background(), tl1Task())
	if err == nil || !strings.Contains(err.Error(), "no NMS endpoint for entity 3") {
		t.Fatalf("err=%v, want no NMS endpoint for entity 3", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestTL1Resolver_EnvEndpointFallback(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	t.Setenv("BOSS_PROVISION_TL1_ADDR", "192.168.1.99:13027")
	t.Setenv("BOSS_PROVISION_TL1_USER", "envu")
	t.Setenv("BOSS_PROVISION_TL1_PASS", "envp")
	mock.ExpectQuery(`SELECT order_no, customer_id, legal_entity_id FROM orders WHERE id = \$1`).
		WithArgs(int64(9)).
		WillReturnRows(mock.NewRows([]string{"order_no", "customer_id", "legal_entity_id"}).AddRow("ORD-1", int64(77), int64(3)))
	mock.ExpectQuery(`SELECT loid FROM lo_accounts WHERE customer_id = \$1`).
		WithArgs(int64(77)).
		WillReturnRows(mock.NewRows([]string{"loid"}).AddRow("LOID-88A1"))
	mock.ExpectQuery(`SELECT content FROM provision_templates WHERE id = \$1`).
		WithArgs(int64(2)).
		WillReturnRows(mock.NewRows([]string{"content"}).AddRow([]byte(tl1Content)))
	mock.ExpectQuery(`SELECT id, resource_id, pon_frame, pon_slot, pon_port, onu_no`).WithArgs(int64(9)).
		WillReturnRows(mock.NewRows([]string{"id", "resource_id", "pon_frame", "pon_slot", "pon_port", "onu_no"}).AddRow(int64(88), int64(5), 0, 7, 5, nil))
	mock.ExpectQuery(`SELECT nms_oltid FROM resources WHERE id = \$1 AND type = 'OLT'`).
		WithArgs(int64(5)).
		WillReturnRows(mock.NewRows([]string{"nms_oltid"}).AddRow("OLT-1"))
	mock.ExpectQuery(`SELECT host, port, username, pass_cipher FROM provision_nms WHERE legal_entity_id = \$1`).
		WithArgs(int64(3)).
		WillReturnError(pgx.ErrNoRows)
	// onu_no nil → 分配事务
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO pon_onu_alloc\(olt_resource_id`).
		WithArgs(int64(5), 0, 7, 5).
		WillReturnRows(mock.NewRows([]string{"next_no"}).AddRow(0))
	mock.ExpectExec(`UPDATE ports SET onu_no = \$1 WHERE id = \$2`).
		WithArgs(0, int64(88)).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectCommit()

	r := &TL1ParamResolver{db: mock}
	p, err := r.Resolve(context.Background(), tl1Task())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if p.Endpoint.Host != "192.168.1.99" || p.Endpoint.Port != 13027 || p.Endpoint.User != "envu" || p.Endpoint.Pass != "envp" {
		t.Fatalf("env endpoint: %+v", p.Endpoint)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestTL1Resolver_OnuNoReuse(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`SELECT order_no, customer_id, legal_entity_id FROM orders WHERE id = \$1`).
		WithArgs(int64(9)).
		WillReturnRows(mock.NewRows([]string{"order_no", "customer_id", "legal_entity_id"}).AddRow("ORD-1", int64(77), int64(3)))
	mock.ExpectQuery(`SELECT loid FROM lo_accounts WHERE customer_id = \$1`).
		WithArgs(int64(77)).
		WillReturnRows(mock.NewRows([]string{"loid"}).AddRow("LOID-88A1"))
	mock.ExpectQuery(`SELECT content FROM provision_templates WHERE id = \$1`).
		WithArgs(int64(2)).
		WillReturnRows(mock.NewRows([]string{"content"}).AddRow([]byte(tl1Content)))
	// onu_no=12 非空 → 直接复用,无分配事务
	mock.ExpectQuery(`SELECT id, resource_id, pon_frame, pon_slot, pon_port, onu_no`).WithArgs(int64(9)).
		WillReturnRows(mock.NewRows([]string{"id", "resource_id", "pon_frame", "pon_slot", "pon_port", "onu_no"}).AddRow(int64(88), int64(5), 0, 7, 5, 12))
	mock.ExpectQuery(`SELECT nms_oltid FROM resources WHERE id = \$1 AND type = 'OLT'`).
		WithArgs(int64(5)).
		WillReturnRows(mock.NewRows([]string{"nms_oltid"}).AddRow("OLT-1"))
	mock.ExpectQuery(`SELECT host, port, username, pass_cipher FROM provision_nms WHERE legal_entity_id = \$1`).
		WithArgs(int64(3)).
		WillReturnRows(mock.NewRows([]string{"host", "port", "username", "pass_cipher"}).AddRow("h", 13027, "u", ""))

	r := &TL1ParamResolver{db: mock}
	p, err := r.Resolve(context.Background(), tl1Task())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if p.ONUNo != 12 {
		t.Fatalf("onu_no=%d, want 12 (reuse)", p.ONUNo)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestTL1Resolver_OnuNoAlloc(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`SELECT order_no, customer_id, legal_entity_id FROM orders WHERE id = \$1`).
		WithArgs(int64(9)).
		WillReturnRows(mock.NewRows([]string{"order_no", "customer_id", "legal_entity_id"}).AddRow("ORD-1", int64(77), int64(3)))
	mock.ExpectQuery(`SELECT loid FROM lo_accounts WHERE customer_id = \$1`).
		WithArgs(int64(77)).
		WillReturnRows(mock.NewRows([]string{"loid"}).AddRow("LOID-88A1"))
	mock.ExpectQuery(`SELECT content FROM provision_templates WHERE id = \$1`).
		WithArgs(int64(2)).
		WillReturnRows(mock.NewRows([]string{"content"}).AddRow([]byte(tl1Content)))
	mock.ExpectQuery(`SELECT id, resource_id, pon_frame, pon_slot, pon_port, onu_no`).WithArgs(int64(9)).
		WillReturnRows(mock.NewRows([]string{"id", "resource_id", "pon_frame", "pon_slot", "pon_port", "onu_no"}).AddRow(int64(88), int64(5), 0, 7, 5, nil))
	mock.ExpectQuery(`SELECT nms_oltid FROM resources WHERE id = \$1 AND type = 'OLT'`).
		WithArgs(int64(5)).
		WillReturnRows(mock.NewRows([]string{"nms_oltid"}).AddRow("OLT-1"))
	mock.ExpectQuery(`SELECT host, port, username, pass_cipher FROM provision_nms WHERE legal_entity_id = \$1`).
		WithArgs(int64(3)).
		WillReturnRows(mock.NewRows([]string{"host", "port", "username", "pass_cipher"}).AddRow("h", 13027, "u", ""))
	// 分配事务:首配 0
	mock.ExpectBegin()
	mock.ExpectQuery(`INSERT INTO pon_onu_alloc\(olt_resource_id`).
		WithArgs(int64(5), 0, 7, 5).
		WillReturnRows(mock.NewRows([]string{"next_no"}).AddRow(0))
	mock.ExpectExec(`UPDATE ports SET onu_no = \$1 WHERE id = \$2`).
		WithArgs(0, int64(88)).WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	mock.ExpectCommit()

	r := &TL1ParamResolver{db: mock}
	p, err := r.Resolve(context.Background(), tl1Task())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if p.ONUNo != 0 {
		t.Fatalf("onu_no=%d, want 0 (first alloc)", p.ONUNo)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
