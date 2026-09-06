package resource

import (
	"context"
	"errors"
	"testing"

	"github.com/pashagolub/pgxmock/v4"
)

// 容量聚合:两分光器 87.50%/12.50%,倒序排列;口径 USED/(USED+IDLE)。
func TestPGStore_Capacity_Aggregate(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	cols := []string{"id", "code", "name", "type", "total_ports", "used_ports", "usage_rate"}
	mock.ExpectQuery(`FROM resources r LEFT JOIN ports p`).
		WithArgs("").
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), "acc_cap-SPL-H", "高分光器", "SPLITTER", int64(8), int64(7), 87.50).
			AddRow(int64(2), "acc_cap-SPL-L", "低分光器", "SPLITTER", int64(8), int64(1), 12.50))

	s := NewPGStore(mock)
	got, err := s.Capacity(context.Background(), "", OrderUsageDesc)
	if err != nil {
		t.Fatalf("Capacity: %v", err)
	}
	if len(got) != 2 || got[0].UsageRate != 87.50 || got[0].UsedPorts != 7 || got[0].TotalPorts != 8 {
		t.Fatalf("got=%+v", got)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// 参数白名单:非法 dim/order 拒绝且不打库。
func TestPGStore_Capacity_ParamGuard(t *testing.T) {
	s := NewPGStore(nil)
	if _, err := s.Capacity(context.Background(), "ROUTER", ""); err == nil {
		t.Fatal("dim=ROUTER 应拒绝")
	}
	if _, err := s.Capacity(context.Background(), "", "id"); err == nil {
		t.Fatal("order=id 应拒绝")
	}
}

// 状态变化才告警:越限无告警(产生)/越限已有告警(跳过)/回落仍有 OPEN(关闭)。
func TestPGStore_CapacityAlertScan_StateChange(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	cols := []string{"id", "code", "name", "type", "total_ports", "used_ports", "usage_rate"}
	mock.ExpectQuery(`FROM resources r LEFT JOIN ports p`).
		WithArgs("").
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(1), "SPL-A", "A", "SPLITTER", int64(8), int64(7), 87.50).
			AddRow(int64(2), "SPL-B", "B", "SPLITTER", int64(8), int64(7), 87.50).
			AddRow(int64(3), "SPL-C", "C", "SPLITTER", int64(8), int64(2), 25.00))
	sink := &fakeCapacitySink{has: map[int64]bool{1: true, 3: true}}

	s := NewPGStore(mock)
	res, err := s.CapacityAlertScan(context.Background(), CapacityWarnThresholdPct, sink)
	if err != nil {
		t.Fatalf("CapacityAlertScan: %v", err)
	}
	if res.Scanned != 3 || res.Created != 1 || res.Resolved != 1 {
		t.Fatalf("res=%+v", res)
	}
	if len(sink.created) != 1 || sink.created[0] != 2 {
		t.Fatalf("created=%v", sink.created)
	}
	if len(sink.closed) != 1 || sink.closed[0] != 3 {
		t.Fatalf("closed=%v", sink.closed)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatalf("unmet: %v", err)
	}
}

// 失败路径:写告警失败必须向上返回错误(调用方落 FAILED 日志),禁止静默吞错。
func TestPGStore_CapacityAlertScan_SinkFail(t *testing.T) {
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	defer mock.Close()
	cols := []string{"id", "code", "name", "type", "total_ports", "used_ports", "usage_rate"}
	mock.ExpectQuery(`FROM resources r LEFT JOIN ports p`).
		WithArgs("").
		WillReturnRows(mock.NewRows(cols).
			AddRow(int64(9), "SPL-X", "X", "SPLITTER", int64(4), int64(4), 100.00))
	sink := &fakeCapacitySink{has: map[int64]bool{}, createErr: errors.New("alarm store down")}

	s := NewPGStore(mock)
	if _, err := s.CapacityAlertScan(context.Background(), CapacityWarnThresholdPct, sink); err == nil {
		t.Fatal("写告警失败应返回错误")
	}
}

// fakeCapacitySink 容量预警口测试桩。
type fakeCapacitySink struct {
	has         map[int64]bool
	created     []int64
	closed      []int64
	createErr   error
	lastContent string
}

func (f *fakeCapacitySink) HasOpenCapacityAlarm(_ context.Context, id int64) (bool, error) {
	return f.has[id], nil
}

func (f *fakeCapacitySink) CreateCapacityAlarm(_ context.Context, id int64, content string) error {
	if f.createErr != nil {
		return f.createErr
	}
	f.created = append(f.created, id)
	f.lastContent = content
	return nil
}

func (f *fakeCapacitySink) CloseCapacityAlarms(_ context.Context, id int64) (int64, error) {
	f.closed = append(f.closed, id)
	return 1, nil
}
