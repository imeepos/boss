package aaa

import (
	"context"
	"testing"
	"time"

	"github.com/pashagolub/pgxmock/v4"
)

// ListUnsentCdrs/MarkCdrsKafkaStatus(话单补偿,000109)形状回归。
func TestPGStore_CdrKafkaStatus(t *testing.T) {
	t.Run("未送达按 id 升序限速", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		rows := pgxmock.NewRows([]string{"id", "loid", "username", "acct_status", "session_id",
			"session_time", "input_octets", "output_octets", "nas_ip", "billing_status", "started_at"}).
			AddRow(int64(1), "LOID-1", "u1", int16(1), "s1", int32(60), int64(100), int64(200), "10.0.0.1", "UNBILLED", time.Now())
		mock.ExpectQuery(`FROM cdrs WHERE kafka_status <> 'SENT' ORDER BY id LIMIT \$1`).
			WithArgs(200).
			WillReturnRows(rows)

		s := NewPGStore(mock)
		out, err := s.ListUnsentCdrs(context.Background(), 200)
		if err != nil {
			t.Fatalf("ListUnsentCdrs: %v", err)
		}
		if len(out) != 1 || out[0].ID != 1 || out[0].Loid != "LOID-1" {
			t.Fatalf("out=%+v", out)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
	t.Run("批量回写状态/空 ids no-op", func(t *testing.T) {
		mock, err := pgxmock.NewPool()
		if err != nil {
			t.Fatal(err)
		}
		defer mock.Close()

		mock.ExpectExec(`UPDATE cdrs SET kafka_status`).
			WithArgs(pgxmock.AnyArg(), "SENT").
			WillReturnResult(pgxmock.NewResult("UPDATE", 2))

		s := NewPGStore(mock)
		if err := s.MarkCdrsKafkaStatus(context.Background(), []int64{1, 2}, "SENT"); err != nil {
			t.Fatalf("MarkCdrsKafkaStatus: %v", err)
		}
		if err := s.MarkCdrsKafkaStatus(context.Background(), nil, "SENT"); err != nil {
			t.Fatalf("empty ids should be no-op: %v", err)
		}
		if err := mock.ExpectationsWereMet(); err != nil {
			t.Fatalf("unmet: %v", err)
		}
	})
}
