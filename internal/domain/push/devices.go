// Package push 移动端推送通道:通道 Sender(pkg)与设备注册表(devices)。
// 设计契约:docs/plan/push-integration.md §4、docs/plan/push-config-design.md。
package push

import (
	"context"
	"errors"
)

// 设备主体类型(与 apikey 主体词对齐)。
const (
	SubjectUser   = "user"
	SubjectWorker = "worker"
)

// ErrInvalidDevice 设备注册入参非法(registrationID/主体不合法)。
var ErrInvalidDevice = errors.New("push: invalid device")

// Device 已注册设备的查询形态。
type Device struct {
	ID             int64
	SubjectType    string
	SubjectID      int64
	RegistrationID string
	Vendor         string
}

// DevicesService 设备注册表:App 启动/登录后上报 RegistrationID,发送链路按主体反查。
type DevicesService interface {
	// RegisterDevice 幂等注册:同 RegistrationID 已存在时改绑到新主体(设备换账号)并刷新活跃时间。
	RegisterDevice(ctx context.Context, subjectType string, subjectID int64, registrationID, vendor string) error
	// RegistrationIDs 某主体当前绑定的全部设备(推送定向目标)。
	RegistrationIDs(ctx context.Context, subjectType string, subjectID int64) ([]string, error)
}

// validRegistrationID JPush RegistrationID 形态:10-64 位字母数字。
func validRegistrationID(id string) bool {
	if len(id) < 10 || len(id) > 64 {
		return false
	}
	for _, r := range id {
		if !(r >= '0' && r <= '9' || r >= 'a' && r <= 'z' || r >= 'A' && r <= 'Z') {
			return false
		}
	}
	return true
}

// validDevice 入参校验:主体枚举 + 主体 ID + RegistrationID 形态。
func validDevice(subjectType string, subjectID int64, registrationID string) bool {
	if subjectType != SubjectUser && subjectType != SubjectWorker {
		return false
	}
	return subjectID > 0 && validRegistrationID(registrationID)
}
