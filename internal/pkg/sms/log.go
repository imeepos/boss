package sms

import (
	"context"
	"log"
)

// LogSender 开发/联调替身:不外呼,仅打印验证码;凭据未配置时的默认通道。
type LogSender struct{}

// NewLogSender 构造日志通道。
func NewLogSender() Sender { return LogSender{} }

// Send 打印目标号码与验证码(仅限开发环境,凭据为空时装配)。
func (LogSender) Send(_ context.Context, phone, code, scene string) error {
	log.Printf("sms[dev] scene=%s phone=%s code=%s", scene, phone, code)
	return nil
}
