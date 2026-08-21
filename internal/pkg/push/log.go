package push

import (
	"context"
	"log"
)

// LogSender 开发/联调替身:不外呼,仅打印推送载荷;凭据未配置时的默认通道。
type LogSender struct{}

// NewLogSender 构造日志通道。
func NewLogSender() Sender { return LogSender{} }

// Send 打印标题与目标(仅限开发环境,凭据为空时装配)。
func (LogSender) Send(_ context.Context, req Request) (string, error) {
	log.Printf("push[dev] title=%q alert=%q reg=%v alias=%v extras=%v",
		req.Title, req.Alert, req.RegistrationIDs, req.Alias, req.Extras)
	return "log", nil
}
