package push

// 定向通知器:按主体查设备 → 逐设备发送(pkg/push 通道)→ 逐行留痕。
// 推送尽力而为:失败不影响业务事务,留痕为权威(docs/plan/push-integration.md §4)。

import (
	"context"
	"fmt"
	"log"
	"time"

	pkgpush "github.com/ymm-001/boss/internal/pkg/push"
)

// 推送留痕状态。
const (
	StatusSent         = "SENT"
	StatusFailed       = "FAILED"
	StatusLog          = "LOG" // 日志通道(开发降级)发出
	StatusSkipNoDevice = "SKIP_NO_DEVICE"
	StatusSkipDisabled = "SKIP_DISABLED"
)

// Record 单次定向推送的留痕行。
type Record struct {
	ID             int64
	SubjectType    string
	SubjectID      int64
	RegistrationID string
	Title          string
	Alert          string
	Extras         map[string]string
	Status         string
	MsgID          string
	Error          string
	CreatedAt      time.Time
}

// RecordsStore 留痕存取。
type RecordsStore interface {
	Record(ctx context.Context, r Record) error
	ListBySubject(ctx context.Context, subjectType string, subjectID int64, limit int) ([]Record, error)
}

// WorkerNotifier 师傅端定向通知(httpapi 注入点,nil 安全跳过)。
type WorkerNotifier interface {
	NotifyWorker(ctx context.Context, workerID int64, title, alert string, extras map[string]string) error
}

// Notifier 定向通知器:sender(通道)+ devices(注册表)+ records(留痕)。
type Notifier struct {
	sender  pkgpush.Sender
	devices DevicesService
	records RecordsStore
}

// NewNotifier 构造通知器。
func NewNotifier(sender pkgpush.Sender, devices DevicesService, records RecordsStore) *Notifier {
	return &Notifier{sender: sender, devices: devices, records: records}
}

// NotifyWorker 通知某师傅全部绑定设备;尽力而为,永远返回 nil(留痕承担结果)。
func (n *Notifier) NotifyWorker(ctx context.Context, workerID int64, title, alert string, extras map[string]string) error {
	ids, err := n.devices.RegistrationIDs(ctx, SubjectWorker, workerID)
	if err != nil {
		n.record(ctx, failRecord(workerID, title, alert, extras, err.Error()))
		return nil
	}
	if len(ids) == 0 {
		n.record(ctx, Record{SubjectType: SubjectWorker, SubjectID: workerID,
			Title: title, Alert: alert, Extras: extras, Status: StatusSkipNoDevice})
		return nil
	}
	for _, id := range ids {
		msgID, serr := n.sender.Send(ctx, pkgpush.Request{
			Title: title, Alert: alert, RegistrationIDs: []string{id}, Extras: extras,
		})
		status, errMsg := StatusSent, ""
		switch {
		case serr == nil && msgID == "log":
			status = StatusLog
		case serr != nil:
			status, errMsg = StatusFailed, serr.Error()
		}
		n.record(ctx, Record{
			SubjectType: SubjectWorker, SubjectID: workerID, RegistrationID: id,
			Title: title, Alert: alert, Extras: extras,
			Status: status, MsgID: msgID, Error: errMsg,
		})
	}
	return nil
}

// failRecord 设备查询失败的单行留痕。
func failRecord(workerID int64, title, alert string, extras map[string]string, errMsg string) Record {
	return Record{SubjectType: SubjectWorker, SubjectID: workerID,
		Title: title, Alert: alert, Extras: extras, Status: StatusFailed, Error: errMsg}
}

// record 留痕失败记日志(尽力而为不中断业务,但静默吞错会掩盖配置类故障)。
func (n *Notifier) record(ctx context.Context, r Record) {
	if n.records == nil {
		return
	}
	if err := n.records.Record(ctx, r); err != nil {
		log.Printf("push: record failed: %v", err)
	}
}

// TicketAssignedAlert 派单通知文案(title/alert 统一出口,指派/转派触发共用)。
func TicketAssignedAlert(ticketNo, workerName string) (string, string) {
	return "新派工单", fmt.Sprintf("工单 %s 已派给%s,请打开师傅端查看详情", ticketNo, workerName)
}
