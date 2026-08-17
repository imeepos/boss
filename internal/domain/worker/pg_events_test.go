package worker

import (
	"context"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

func TestPGStore_ListMaterials(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "worker_id", "group_id", "group_name", "legal_entity_id", "legal_entity_name", "region_id", "region_name", "name", "qty"}
	mock.ExpectQuery(`SELECT id, worker_id, group_id, group_name, legal_entity_id, legal_entity_name, region_id, region_name, name, qty FROM worker_materials`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), int64(1), int64(1), "装机一组", int64(1), "主品牌·企业", int64(11), "马尼拉市", "光纤跳线", int32(3)))

	s := NewPGStore(mock)
	got, err := s.ListMaterials(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListMaterials: %v", err)
	}
	if len(got) != 1 || got[0].Qty != 3 {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_AppendMaterial(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO worker_materials`).
		WithArgs(int64(1), int64(1), "装机一组", int64(1), "主品牌·企业", int64(11), "马尼拉市", "光纤跳线", int32(3)).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock)
	id, err := s.AppendMaterial(context.Background(), Material{
		WorkerID: 1, GroupID: 1, GroupName: "装机一组", LegalEntityID: 1, LegalEntityName: "主品牌·企业",
		RegionID: 11, RegionName: "马尼拉市", Name: "光纤跳线", Qty: 3,
	})
	if err != nil {
		t.Fatalf("AppendMaterial: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ListTools(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "worker_id", "group_id", "group_name", "legal_entity_id", "legal_entity_name", "region_id", "region_name", "name", "borrowed"}
	mock.ExpectQuery(`SELECT id, worker_id, group_id, group_name, legal_entity_id, legal_entity_name, region_id, region_name, name, borrowed FROM worker_tools`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), int64(1), int64(1), "装机一组", int64(1), "主品牌·企业", int64(11), "马尼拉市", "熔纤机", true))

	s := NewPGStore(mock)
	got, err := s.ListTools(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListTools: %v", err)
	}
	if len(got) != 1 || !got[0].Borrowed {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_AppendTool(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO worker_tools`).
		WithArgs(int64(1), int64(1), "装机一组", int64(1), "主品牌·企业", int64(11), "马尼拉市", "熔纤机", true).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock)
	id, err := s.AppendTool(context.Background(), Tool{
		WorkerID: 1, GroupID: 1, GroupName: "装机一组", LegalEntityID: 1, LegalEntityName: "主品牌·企业",
		RegionID: 11, RegionName: "马尼拉市", Name: "熔纤机", Borrowed: true,
	})
	if err != nil {
		t.Fatalf("AppendTool: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ListFeedbacks(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "worker_id", "worker_name", "group_id", "group_name", "legal_entity_id", "legal_entity_name", "region_id", "region_name", "ticket_id", "customer_id", "customer_name", "score", "need_review"}
	mock.ExpectQuery(`SELECT id, worker_id, worker_name, group_id, group_name`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), int64(1), "张师傅", int64(1), "装机一组", int64(1), "主品牌·企业", int64(11), "马尼拉市", int64(1001), int64(1), "王先生", int16(5), false))

	s := NewPGStore(mock)
	got, err := s.ListFeedbacks(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListFeedbacks: %v", err)
	}
	if len(got) != 1 || got[0].Score != 5 {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_AppendFeedback(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO worker_feedbacks`).
		WithArgs(int64(1), "张师傅", int64(1), "装机一组", int64(1), "主品牌·企业", int64(11), "马尼拉市", int64(1001), int64(1), "王先生", int16(5), false).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock)
	id, err := s.AppendFeedback(context.Background(), Feedback{
		WorkerID: 1, WorkerName: "张师傅", GroupID: 1, GroupName: "装机一组", LegalEntityID: 1, LegalEntityName: "主品牌·企业",
		RegionID: 11, RegionName: "马尼拉市", TicketID: 1001, CustomerID: 1, CustomerName: "王先生", Score: 5,
	})
	if err != nil {
		t.Fatalf("AppendFeedback: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ListAssetReturns(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "worker_id", "group_id", "group_name", "legal_entity_id", "legal_entity_name", "region_id", "region_name", "asset_id", "reason", "status"}
	mock.ExpectQuery(`SELECT id, worker_id, group_id, group_name, legal_entity_id, legal_entity_name, region_id, region_name, asset_id, reason, status FROM asset_returns`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), int64(1), int64(1), "装机一组", int64(1), "主品牌·企业", int64(11), "马尼拉市", int64(5), "完工回收", "PENDING"))

	s := NewPGStore(mock)
	got, err := s.ListAssetReturns(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListAssetReturns: %v", err)
	}
	if len(got) != 1 || got[0].AssetID != 5 {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_AppendAssetReturn(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO asset_returns`).
		WithArgs(int64(1), int64(1), "装机一组", int64(1), "主品牌·企业", int64(11), "马尼拉市", int64(5), "完工回收", "PENDING").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock)
	id, err := s.AppendAssetReturn(context.Background(), AssetReturn{
		WorkerID: 1, GroupID: 1, GroupName: "装机一组", LegalEntityID: 1, LegalEntityName: "主品牌·企业",
		RegionID: 11, RegionName: "马尼拉市", AssetID: 5, Reason: "完工回收", Status: "PENDING",
	})
	if err != nil {
		t.Fatalf("AppendAssetReturn: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
