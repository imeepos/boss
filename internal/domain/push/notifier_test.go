package push

// Notifier 语义测试:SENT/LOG/FAILED/SKIP_NO_DEVICE 留痕与文案出口。

import (
	"context"
	"errors"
	"testing"

	pkgpush "github.com/ymm-001/boss/internal/pkg/push"
)

type fakeSender struct {
	msgIDs []string
	err    error
	reqs   []pkgpush.Request
}

func (f *fakeSender) Send(_ context.Context, req pkgpush.Request) (string, error) {
	f.reqs = append(f.reqs, req)
	if f.err != nil {
		return "", f.err
	}
	id := "1000"
	if len(f.msgIDs) > 0 {
		id = f.msgIDs[len(f.reqs)-1]
	}
	return id, nil
}

func newNotifierForTest(sender pkgpush.Sender) (*Notifier, *RecordsMemoryStore, *DevicesMemoryStore) {
	devices := NewDevicesMemoryStore()
	records := NewRecordsMemoryStore()
	return NewNotifier(sender, devices, records), records, devices
}

func TestNotifierSentAndSkip(t *testing.T) {
	ctx := context.Background()
	n, records, devices := newNotifierForTest(&fakeSender{msgIDs: []string{"m1", "m2"}})
	_ = devices.RegisterDevice(ctx, SubjectWorker, 7, "1507bfd3f9ac1e045a", "jpush")
	_ = devices.RegisterDevice(ctx, SubjectWorker, 7, "1507bfd3f9ac1e045b", "jpush")

	if err := n.NotifyWorker(ctx, 7, "新派工单", "工单 TK-1 已派给您", map[string]string{"ticketNo": "TK-1"}); err != nil {
		t.Fatal(err)
	}
	rows, _ := records.ListBySubject(ctx, SubjectWorker, 7, 10)
	if len(rows) != 2 || rows[0].Status != StatusSent || rows[1].Status != StatusSent {
		t.Fatalf("records=%+v", rows)
	}
	if rows[0].RegistrationID == "" || rows[0].Extras["ticketNo"] != "TK-1" {
		t.Fatalf("record fields=%+v", rows[0])
	}

	// 无设备主体 → SKIP_NO_DEVICE 单行。
	if err := n.NotifyWorker(ctx, 8, "t", "a", nil); err != nil {
		t.Fatal(err)
	}
	rows8, _ := records.ListBySubject(ctx, SubjectWorker, 8, 10)
	if len(rows8) != 1 || rows8[0].Status != StatusSkipNoDevice {
		t.Fatalf("skip record=%+v", rows8)
	}
}

func TestNotifierLogAndFailed(t *testing.T) {
	ctx := context.Background()
	// 日志通道(msg_id=log)→ LOG。
	n, records, devices := newNotifierForTest(logSenderStub{})
	_ = devices.RegisterDevice(ctx, SubjectWorker, 7, "1507bfd3f9ac1e045a", "jpush")
	_ = n.NotifyWorker(ctx, 7, "t", "a", nil)
	rows, _ := records.ListBySubject(ctx, SubjectWorker, 7, 10)
	if len(rows) != 1 || rows[0].Status != StatusLog {
		t.Fatalf("log record=%+v", rows)
	}

	// 发送失败 → FAILED 且不中断。
	n2, records2, devices2 := newNotifierForTest(&fakeSender{err: errors.New("jpush 502")})
	_ = devices2.RegisterDevice(ctx, SubjectWorker, 9, "1507bfd3f9ac1e045a", "jpush")
	if err := n2.NotifyWorker(ctx, 9, "t", "a", nil); err != nil {
		t.Fatal(err)
	}
	rows2, _ := records2.ListBySubject(ctx, SubjectWorker, 9, 10)
	if len(rows2) != 1 || rows2[0].Status != StatusFailed || rows2[0].Error == "" {
		t.Fatalf("failed record=%+v", rows2)
	}
}

type logSenderStub struct{}

func (logSenderStub) Send(context.Context, pkgpush.Request) (string, error) { return "log", nil }

func TestTicketAssignedAlert(t *testing.T) {
	title, alert := TicketAssignedAlert("TK-9", "王师傅")
	if title != "新派工单" || alert != "工单 TK-9 已派给王师傅,请打开师傅端查看详情" {
		t.Fatalf("alert=%q %q", title, alert)
	}
}
