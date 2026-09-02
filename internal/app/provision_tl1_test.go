package app

import (
	"context"
	"strings"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"

	"github.com/ymm-001/boss/internal/domain/provision"
	"github.com/ymm-001/boss/internal/pkg/secretbox"
)

// tl1Content 测试用模板 content(含 tr069 无 svlan,验证 HasSVLAN=false)。
const tl1Content = `{"onuType":"Internet","services":{"internet":{"svlan":1113,"cvlan":1,"uv":100,"scos":0,"ccos":0},"tr069":{"cvlan":1000,"uv":1000,"scos":6,"ccos":6}}}`

func mustSeal(t *testing.T, s string) string {
	t.Helper()
	c, err := secretbox.Seal(s)
	if err != nil {
		t.Fatalf("seal: %v", err)
	}
	return c
}

// tl1Task 构造测试任务。
func tl1Task() provision.Task {
	return provision.Task{ID: 1, TaskNo: "TASK-1", OrderID: 9, TemplateID: 2}
}

func TestTL1Resolver_FullChain(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	svc := tl1Content
	enc := mustSeal(t, "tl1pass")
	mock.ExpectQuery(`SELECT order_no, customer_id, legal_entity_id FROM orders WHERE id = \$1`).
		WithArgs(int64(9)).
		WithArgs(int64(9)).
		WillReturnRows(mock.NewRows([]string{"order_no", "customer_id", "legal_entity_id"}).AddRow("ORD-1", int64(77), int64(3)))
	mock.ExpectQuery(`SELECT loid FROM lo_accounts WHERE customer_id = \$1`).
		WithArgs(int64(77)).
		WithArgs(int64(77)).
		WillReturnRows(mock.NewRows([]string{"loid"}).AddRow("LOID-88A1"))
	mock.ExpectQuery(`SELECT content FROM provision_templates WHERE id = \$1`).
		WithArgs(int64(2)).
		WithArgs(int64(2)).
		WillReturnRows(mock.NewRows([]string{"content"}).AddRow([]byte(svc)))
	mock.ExpectQuery(`SELECT id, resource_id, pon_frame, pon_slot, pon_port, onu_no`).
		WithArgs(int64(9)).
		WillReturnRows(mock.NewRows([]string{"id", "resource_id", "pon_frame", "pon_slot", "pon_port", "onu_no"}).AddRow(int64(88), int64(5), 0, 7, 5, 6))
	mock.ExpectQuery(`SELECT nms_oltid FROM resources WHERE id = \$1 AND type = 'OLT'`).
		WithArgs(int64(5)).
		WithArgs(int64(5)).
		WillReturnRows(mock.NewRows([]string{"nms_oltid"}).AddRow("OLT-1"))
	mock.ExpectQuery(`SELECT host, port, username, pass_cipher FROM provision_nms WHERE legal_entity_id = \$1`).
		WithArgs(int64(3)).
		WithArgs(int64(3)).
		WillReturnRows(mock.NewRows([]string{"host", "port", "username", "pass_cipher"}).AddRow("192.168.0.1", 13027, "admin", enc))

	r := &TL1ParamResolver{db: mock}
	p, err := r.Resolve(context.Background(), tl1Task())
	if err != nil {
		t.Fatalf("Resolve: %v", err)
	}
	if p.Desc != "PRV-ORD-1" || p.ONUID != "LOID-88A1" || p.OLTID != "OLT-1" || p.PONID != "NA-0-7-5" || p.ONUNo != 6 || p.ONUType != "Internet" {
		t.Fatalf("params mismatch: %+v", p)
	}
	if p.Endpoint.Host != "192.168.0.1" || p.Endpoint.Port != 13027 || p.Endpoint.User != "admin" || p.Endpoint.Pass != "tl1pass" {
		t.Fatalf("endpoint: %+v", p.Endpoint)
	}
	if len(p.Services) != 2 {
		t.Fatalf("services len=%d, want 2", len(p.Services))
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestTL1Resolver_ServicesMapping(t *testing.T) {
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
	// onu_no nil → 走分配事务
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
	// services 键序不定,按名查找断言
	svcByName := map[string]struct {
		hasSVLAN         bool
		svlan, cvlan, uv int
	}{
		"internet": {true, 1113, 1, 100},
		"tr069":    {false, 0, 1000, 1000},
	}
	for _, s := range p.Services {
		want, ok := svcByName[s.Name]
		if !ok {
			t.Fatalf("unexpected service %q", s.Name)
		}
		if s.HasSVLAN != want.hasSVLAN || s.SVLAN != want.svlan || s.CVLAN != want.cvlan || s.UV != want.uv {
			t.Fatalf("svc %s mismatch: %+v", s.Name, s)
		}
		if s.ONUID != "LOID-88A1" || s.OLTID != "OLT-1" || s.PONID != "NA-0-7-5" {
			t.Fatalf("svc %s ctx mismatch: %+v", s.Name, s)
		}
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestTL1Resolver_LOAccountMissing(t *testing.T) {
	mock, _ := pgxmock.NewPool()
	defer mock.Close()
	mock.ExpectQuery(`SELECT order_no, customer_id, legal_entity_id FROM orders WHERE id = \$1`).
		WithArgs(int64(9)).
		WillReturnRows(mock.NewRows([]string{"order_no", "customer_id", "legal_entity_id"}).AddRow("ORD-1", int64(77), int64(3)))
	mock.ExpectQuery(`SELECT loid FROM lo_accounts WHERE customer_id = \$1`).
		WithArgs(int64(77)).
		WillReturnError(pgx.ErrNoRows)

	r := &TL1ParamResolver{db: mock}
	_, err := r.Resolve(context.Background(), tl1Task())
	if err == nil || !strings.Contains(err.Error(), "lo account missing") {
		t.Fatalf("err=%v, want lo account missing", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestTL1Resolver_PortMissingPON(t *testing.T) {
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
	// pon_frame 为 NULL → 缺 PON 定位
	mock.ExpectQuery(`SELECT id, resource_id, pon_frame, pon_slot, pon_port, onu_no`).WithArgs(int64(9)).
		WillReturnRows(mock.NewRows([]string{"id", "resource_id", "pon_frame", "pon_slot", "pon_port", "onu_no"}).AddRow(int64(88), int64(5), nil, 7, 5, nil))

	r := &TL1ParamResolver{db: mock}
	_, err := r.Resolve(context.Background(), tl1Task())
	if err == nil || !strings.Contains(err.Error(), "port missing PON positioning") {
		t.Fatalf("err=%v, want port missing PON positioning", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestTL1Resolver_OLTMissingNMSID(t *testing.T) {
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
	// nms_oltid 空串 → OLT 缺标识
	mock.ExpectQuery(`SELECT nms_oltid FROM resources WHERE id = \$1 AND type = 'OLT'`).
		WithArgs(int64(5)).
		WillReturnRows(mock.NewRows([]string{"nms_oltid"}).AddRow(""))

	r := &TL1ParamResolver{db: mock}
	_, err := r.Resolve(context.Background(), tl1Task())
	if err == nil || !strings.Contains(err.Error(), "OLT missing nms_oltid") {
		t.Fatalf("err=%v, want OLT missing nms_oltid", err)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
