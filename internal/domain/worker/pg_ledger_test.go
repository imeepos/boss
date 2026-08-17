package worker

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

func TestPGStore_ListMemberships(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "worker_id", "group_id", "group_name", "legal_entity_id", "legal_entity_name", "region_id", "region_name", "reason", "operator_account_id", "effective_from", "effective_to"}
	mock.ExpectQuery(`SELECT id, worker_id, group_id, group_name, legal_entity_id, legal_entity_name`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), int64(1), int64(1), "装机一组", int64(1), "主品牌·企业", int64(11), "马尼拉市", "", int64(0), ts, ts).
			AddRow(int64(2), int64(1), int64(2), "抢修组", int64(1), "主品牌·企业", int64(11), "马尼拉市", "支援", int64(9), ts, nil))

	s := NewPGStore(mock)
	got, err := s.ListMemberships(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListMemberships: %v", err)
	}
	if len(got) != 2 || got[0].EffectiveTo == nil || got[1].EffectiveTo != nil {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_AppendMembership(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	var nilTime *time.Time
	mock.ExpectQuery(`INSERT INTO worker_group_memberships`).
		WithArgs(int64(1), int64(2), "抢修组", int64(1), "主品牌·企业", int64(11), "马尼拉市", "支援", int64(9), ts, nilTime).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(3)))

	s := NewPGStore(mock)
	id, err := s.AppendMembership(context.Background(), Membership{
		WorkerID: 1, GroupID: 2, GroupName: "抢修组", LegalEntityID: 1, LegalEntityName: "主品牌·企业",
		RegionID: 11, RegionName: "马尼拉市", Reason: "支援", OperatorAccountID: 9, EffectiveFrom: ts,
	})
	if err != nil {
		t.Fatalf("AppendMembership: %v", err)
	}
	if id != 3 {
		t.Fatalf("id=%d, want 3", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_GetSettings(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT id, worker_id, accepting, radius_km, accept_types, updated_at FROM worker_settings`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows([]string{"id", "worker_id", "accepting", "radius_km", "accept_types", "updated_at"}).
			AddRow(int64(1), int64(1), true, int16(5), "新装宽带,宽带变更", ts))

	s := NewPGStore(mock)
	st, err := s.GetSettings(context.Background(), 1)
	if err != nil {
		t.Fatalf("GetSettings: %v", err)
	}
	if !st.Accepting || st.RadiusKm != 5 {
		t.Fatalf("st=%+v", st)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_UpsertSettings(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO worker_settings`).
		WithArgs(int64(1), false, int16(3), "拆机", &ts).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(1)))

	s := NewPGStore(mock)
	id, err := s.UpsertSettings(context.Background(), Settings{
		WorkerID: 1, Accepting: false, RadiusKm: 3, AcceptTypes: "拆机", UpdatedAt: &ts,
	})
	if err != nil {
		t.Fatalf("UpsertSettings: %v", err)
	}
	if id != 1 {
		t.Fatalf("id=%d, want 1", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ListMessages(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "worker_id", "level", "title", "content", "sent_at", "read"}
	mock.ExpectQuery(`SELECT id, worker_id, level, title, content, sent_at, read FROM worker_messages`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), int64(1), "URGENT", "台风应急", "批量复测任务已下发", ts, false))

	s := NewPGStore(mock)
	got, err := s.ListMessages(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListMessages: %v", err)
	}
	if len(got) != 1 || got[0].Level != "URGENT" {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_SendMessage(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO worker_messages`).
		WithArgs(int64(1), "INFO", "配置下发", "全部预下发成功", ts, false).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock)
	id, err := s.SendMessage(context.Background(), Message{
		WorkerID: 1, Level: "INFO", Title: "配置下发", Content: "全部预下发成功", SentAt: ts,
	})
	if err != nil {
		t.Fatalf("SendMessage: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
