package provision

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/pashagolub/pgxmock/v4"
)

// TestPGStore_GetLogDetail 详情聚合:日志(指令/应答)+任务+订单+模板单查询取齐。
func TestPGStore_GetLogDetail(t *testing.T) {
	t.Run("命中:全维度取齐", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT l\.id, l\.task_id`).
			WithArgs(int64(413)).
			WillReturnRows(mock.NewRows([]string{
				"id", "task_id", "resource_id", "resource_code", "template_id", "template_code",
				"result", "retries", "commands", "device_response", "driver", "created_at",
				"id", "task_no", "order_id", "stage_event", "lo_account_id", "template_id", "status",
				"order_no", "status", "name",
				"code", "name", "status", "version", "content",
			}).AddRow(
				int64(413), int64(254), int64(0), "", int64(142), "TPL-FTTH-100M",
				"SUCCESS", int16(0), "provision apply template=142", "OK", "tl1",
				time.Date(2026, 9, 2, 16, 20, 37, 0, time.UTC),
				int64(254), "PRV-O667", int64(667), "preConfigOLT", int64(99), int64(142), "DONE",
				"ORD-20260903-000633", "INSTALLING", "家庭宽带100M",
				"TPL-FTTH-100M", "FTTH 100M开通", "ENABLED", int32(1), []byte(`{"bandwidth":"100M"}`),
			))
		d, err := NewPGStore(mock).GetLogDetail(context.Background(), 413)
		if err != nil {
			t.Fatalf("GetLogDetail: %v", err)
		}
		if d.Log.Commands != "provision apply template=142" || d.Log.DeviceResp != "OK" {
			t.Fatalf("trace=%+v", d.Log)
		}
		if d.Log.Driver != "tl1" {
			t.Fatalf("driver=%q, want tl1", d.Log.Driver)
		}
		if d.Task.TaskNo != "PRV-O667" || d.Order.OrderNo != "ORD-20260903-000633" {
			t.Fatalf("task/order=%+v %+v", d.Task, d.Order)
		}
		if d.Template.Content["bandwidth"] != "100M" {
			t.Fatalf("template content=%v", d.Template.Content)
		}
	})

	t.Run("日志不存在", func(t *testing.T) {
		mock, _ := pgxmock.NewPool()
		defer mock.Close()
		mock.ExpectQuery(`SELECT l\.id, l\.task_id`).
			WithArgs(int64(1)).
			WillReturnError(pgx.ErrNoRows)
		if _, err := NewPGStore(mock).GetLogDetail(context.Background(), 1); !errors.Is(err, ErrLogNotFound) {
			t.Fatalf("err=%v", err)
		}
	})
}
