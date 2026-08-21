// Package push 移动端推送通道抽象:用户端/师傅端 App 通知经聚合商下发。
// 当前唯一生产通道为极光 JPush(REST v3 直调);凭据未配置时降级日志通道。
package push

import (
	"context"
	"errors"
)

// ErrDisabled 通道被管理员停用(push.enabled=false)。
var ErrDisabled = errors.New("push: channel disabled")

// Request 单次推送:Alias 与 RegistrationIDs 二选一,同时给出以 RegistrationIDs 为准。
type Request struct {
	Title           string
	Alert           string
	Alias           []string
	RegistrationIDs []string
	Extras          map[string]string
}

// Sender 推送发送通道;返回服务商消息 ID。
type Sender interface {
	Send(ctx context.Context, req Request) (string, error)
}

// ChannelConfig 通道配置(resolve 返回的明文形态)。
type ChannelConfig struct {
	Enabled        bool
	Provider       string
	AppKey         string
	MasterSecret   string
	APIURL         string
	ApnsProduction bool
	LiveTimeSec    int64
}
