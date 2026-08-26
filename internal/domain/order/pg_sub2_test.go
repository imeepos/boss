package order

import (
	"context"
	"errors"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

func TestPGStore_ListDismantles(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "dismantle_no", "order_id", "legal_entity_id", "legal_entity_name", "asset_id", "port_id", "status"}
	mock.ExpectQuery(`SELECT id, dismantle_no, order_id, legal_entity_id, legal_entity_name, asset_id, port_id, status FROM dismantles`).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), "DSM-20250817-001", int64(9), int64(1), "主品牌·企业", int64(5), int64(200), "PENDING"))

	s := NewPGStore(mock, stubExists{})
	got, err := s.ListDismantles(context.Background())
	if err != nil {
		t.Fatalf("ListDismantles: %v", err)
	}
	if len(got) != 1 || got[0].AssetID != 5 {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_CreateDismantle(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`SELECT EXISTS\(SELECT 1 FROM orders WHERE id = \$1\)`).
		WithArgs(int64(10)).WillReturnRows(mock.NewRows([]string{"exists"}).AddRow(true))
	mock.ExpectQuery(`INSERT INTO dismantles`).
		WithArgs("DSM-20250817-002", int64(10), int64(1), "主品牌·企业", int64(6), int64(201), "PENDING").
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock, stubExists{})
	id, err := s.CreateDismantle(context.Background(), Dismantle{
		DismantleNo: "DSM-20250817-002", OrderID: 10, LegalEntityID: 1, LegalEntityName: "主品牌·企业", AssetID: 6, PortID: 201, Status: "PENDING",
	})
	if err != nil {
		t.Fatalf("CreateDismantle: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_ListActivationCallbacks(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "order_id", "result", "retries"}
	mock.ExpectQuery(`SELECT id, order_id, result, retries FROM activation_callbacks`).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), int64(1), "FAILED", int16(2)))

	s := NewPGStore(mock, stubExists{})
	got, err := s.ListActivationCallbacks(context.Background())
	if err != nil {
		t.Fatalf("ListActivationCallbacks: %v", err)
	}
	if len(got) != 1 || got[0].Retries != 2 {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_AppendActivationCallback(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	// 幂等(000154):同订单 upsert,ON CONFLICT (order_id) DO UPDATE result/retries。
	mock.ExpectQuery(`INSERT INTO activation_callbacks`).
		WithArgs(int64(1), "SUCCESS", int16(0)).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock, stubExists{})
	id, err := s.AppendActivationCallback(context.Background(), ActivationCallback{OrderID: 1, Result: "SUCCESS"})
	if err != nil {
		t.Fatalf("AppendActivationCallback: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_RetryActivationCallback(t *testing.T) {
	t.Run("订单已DONE:仅计数", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		mock.ExpectQuery(`SELECT id, order_id, result, retries FROM activation_callbacks`).
			WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"id", "order_id", "result", "retries"}).
				AddRow(int64(7), int64(9), "SUCCESS", int16(1)))
		mock.ExpectQuery(`SELECT status FROM orders`).
			WithArgs(int64(9)).
			WillReturnRows(mock.NewRows([]string{"status"}).AddRow("DONE"))
		mock.ExpectExec(`UPDATE activation_callbacks SET retries = retries \+ 1`).
			WithArgs(int64(7)).WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		s := NewPGStore(mock, stubExists{})
		if err := s.RetryActivationCallback(context.Background(), 7); err != nil {
			t.Fatalf("RetryActivationCallback: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("订单未DONE:重放确认失败保持FAILED并计数", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		mock.ExpectQuery(`SELECT id, order_id, result, retries FROM activation_callbacks`).
			WithArgs(int64(7)).
			WillReturnRows(mock.NewRows([]string{"id", "order_id", "result", "retries"}).
				AddRow(int64(7), int64(9), "FAILED", int16(1)))
		mock.ExpectQuery(`SELECT status FROM orders`).
			WithArgs(int64(9)).
			WillReturnRows(mock.NewRows([]string{"status"}).AddRow("INSTALLING"))
		// NotifyActivation 重放:确认 LO 账号缺失 → 落 FAILED 行,不推进。
		mock.ExpectQuery(`SELECT customer_id FROM orders`).
			WithArgs(int64(9)).
			WillReturnRows(mock.NewRows([]string{"customer_id"}).AddRow(int64(3)))
		mock.ExpectQuery(`INSERT INTO activation_callbacks`).
			WithArgs(int64(9), "FAILED", int16(0)).
			WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(7)))
		mock.ExpectExec(`UPDATE activation_callbacks SET retries = retries \+ 1`).
			WithArgs(int64(7)).WillReturnResult(pgxmock.NewResult("UPDATE", 1))

		s := NewPGStore(mock, stubExists{}, &stubProfileCreator{err: errors.New("no lo")})
		if err := s.RetryActivationCallback(context.Background(), 7); err == nil {
			t.Fatal("want error when confirm still fails")
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})

	t.Run("回调不存在:ErrOrderNotFound", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()
		mock.ExpectQuery(`SELECT id, order_id, result, retries FROM activation_callbacks`).
			WithArgs(int64(99)).
			WillReturnError(pgx.ErrNoRows)

		s := NewPGStore(mock, stubExists{})
		if err := s.RetryActivationCallback(context.Background(), 99); err != ErrOrderNotFound {
			t.Fatalf("err=%v, want ErrOrderNotFound", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}

func TestPGStore_ListDispatchTransfers(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	cols := []string{"id", "ticket_id", "from_worker_id", "from_worker_name", "to_worker_id", "to_worker_name", "reason", "operator_account_id", "transferred_at"}
	mock.ExpectQuery(`SELECT id, ticket_id, COALESCE\(from_worker_id, 0\)`).
		WithArgs(int64(1)).
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), int64(1), int64(1024), "张师傅", int64(1036), "王师傅", "师傅请假", int64(9), ts))

	s := NewPGStore(mock, stubExists{})
	got, err := s.ListDispatchTransfers(context.Background(), 1)
	if err != nil {
		t.Fatalf("ListDispatchTransfers: %v", err)
	}
	if len(got) != 1 || got[0].ToWorkerID != 1036 {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

func TestPGStore_AppendDispatchTransfer(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()

	mock.ExpectQuery(`INSERT INTO dispatch_transfers`).
		WithArgs(int64(1), int64(1024), "张师傅", int64(1036), "王师傅", "师傅请假", int64(9), ts).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(2)))

	s := NewPGStore(mock, stubExists{})
	id, err := s.AppendDispatchTransfer(context.Background(), DispatchTransfer{
		TicketID: 1, FromWorkerID: 1024, FromWorkerName: "张师傅", ToWorkerID: 1036, ToWorkerName: "王师傅",
		Reason: "师傅请假", OperatorAccountID: 9, TransferredAt: ts,
	})
	if err != nil {
		t.Fatalf("AppendDispatchTransfer: %v", err)
	}
	if id != 2 {
		t.Fatalf("id=%d, want 2", id)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}
