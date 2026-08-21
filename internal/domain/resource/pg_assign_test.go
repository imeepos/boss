package resource

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

var tsTo = ts.Add(24 * time.Hour)

// TestPGStore_ListAssignments 契约:按设备过滤归属台账;resourceID=0 返回全部。
func TestPGStore_ListAssignments(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, resource_id, legal_entity_id, legal_entity_name, COALESCE\(address_id, 0\), COALESCE\(address_name, ''\), COALESCE\(region_id, 0\), COALESCE\(region_name, ''\), COALESCE\(reason, ''\), COALESCE\(operator_account_id, 0\), effective_from, effective_to`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{
			"id", "resource_id", "legal_entity_id", "legal_entity_name", "address_id", "address_name",
			"region_id", "region_name", "reason", "operator_account_id", "effective_from", "effective_to",
		}).AddRow(int64(1), int64(1), int64(1), "Acme Ltd", int64(100), "Manila", int64(11), "root.luzon.ncr.manila",
			"调拨", int64(3), ts, tsTo))

	s := NewPGStore(mock)
	got, err := s.ListAssignments(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListAssignments: %v", err)
	}
	if len(got) != 1 || got[0].LegalEntityName != "Acme Ltd" || got[0].EffectiveTo == nil || !got[0].EffectiveTo.Equal(tsTo) {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_AppendAssignment 契约:追加归属台账并返回自增 id;0 值写 NULL。
func TestPGStore_AppendAssignment(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO resource_assignments`).
		WithArgs(int64(1), int64(1), "Acme Ltd", nil, "Manila", int64(11), "root.luzon.ncr.manila", "调拨", nil, ts, &tsTo).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(7)))

	s := NewPGStore(mock)
	id, err := s.AppendAssignment(context.Background(), ResourceAssignment{
		ResourceID: 1, LegalEntityID: 1, LegalEntityName: "Acme Ltd",
		AddressName: "Manila", RegionID: 11, RegionName: "root.luzon.ncr.manila", Reason: "调拨",
		EffectiveFrom: ts, EffectiveTo: &tsTo,
	})
	if err != nil {
		t.Fatalf("AppendAssignment: %v", err)
	}
	if id != 7 {
		t.Fatalf("id=%d, want 7", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_ListQosTemplates 契约:按公司过滤 QoS 模板;legalEntityID=0 返回全部。
func TestPGStore_ListQosTemplates(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, legal_entity_id, code, name FROM qos_templates`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"id", "legal_entity_id", "code", "name"}).
			AddRow(int64(1), int64(1), "QoS-VIP", "VIP"))

	s := NewPGStore(mock)
	got, err := s.ListQosTemplates(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListQosTemplates: %v", err)
	}
	if len(got) != 1 || got[0].Code != "QoS-VIP" || got[0].Name != "VIP" {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}

// TestPGStore_CreateQosTemplate 契约:新增 QoS 模板并返回自增 id。
func TestPGStore_CreateQosTemplate(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// FK 校验:legal entity 存在。
	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM legal_entities WHERE id = \$1\)`).
		WithArgs(int64(1)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`INSERT INTO qos_templates`).
		WithArgs(int64(1), "QoS-VIP", "VIP").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(1)))

	s := NewPGStore(mock)
	id, err := s.CreateQosTemplate(context.Background(), QosTemplate{LegalEntityID: 1, Code: "QoS-VIP", Name: "VIP"})
	if err != nil {
		t.Fatalf("CreateQosTemplate: %v", err)
	}
	if id != 1 {
		t.Fatalf("id=%d, want 1", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet expectations: %v", err)
	}
}
