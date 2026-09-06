package odn

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// 物理端口占用态(P2,T11,迁移 000200)。
const (
	PortIdle     = "IDLE"       // 空闲可分配
	PortReserved = "RESERVED"   // 已被订单预占
	PortInSvc    = "IN_SERVICE" // 在网服务中
)

// 端口领域错误。
var (
	// ErrInvalidPortState 非法端口状态转移(含并发 CAS 失败)。
	ErrInvalidPortState = errors.New("odn: invalid port state")
	// ErrNoFreePort 无空闲端口可分配。
	ErrNoFreePort = errors.New("odn: no free port")
	// ErrNoCoverageDevice 地址覆盖未挂接设备,无法定位分配目标。
	ErrNoCoverageDevice = errors.New("odn: coverage has no device")
)

// ODNPort 物理端口。
type ODNPort struct {
	ID        int64  `json:"id"`
	DeviceID  int64  `json:"deviceId"`
	PortNo    int16  `json:"portNo"`
	Status    string `json:"status"`
	OrderID   int64  `json:"orderId"`
	UpdatedAt string `json:"updatedAt"`
}

// portTransitions 允许的端口状态转移。
var portTransitions = map[string][]string{
	PortIdle:     {PortReserved, PortInSvc},
	PortReserved: {PortInSvc, PortIdle},
	PortInSvc:    {PortIdle},
}

// ValidatePortTransition 转移合法性(同态 no-op 放行)。
func ValidatePortTransition(from, to string) error {
	if from == to {
		return nil
	}
	for _, nxt := range portTransitions[from] {
		if nxt == to {
			return nil
		}
	}
	return ErrInvalidPortState
}

// PortStore 物理端口存储口(PGStore 实现)。
type PortStore interface {
	ListPorts(ctx context.Context, deviceID int64) ([]ODNPort, error)
	AllocatePort(ctx context.Context, deviceID, orderID int64) (*ODNPort, error)
	ReleasePort(ctx context.Context, portID int64) error
	ActivatePort(ctx context.Context, portID int64) error
	AllocateForAddress(ctx context.Context, addressID, orderID int64) (*ODNPort, error)
}

// AllocateForAddress 覆盖关联兑现:地址 → 覆盖设备 → 空闲端口。
// 覆盖未挂设备或无空闲端口时明确报错(下单门控的数据基础)。
func (s *PGStore) AllocateForAddress(ctx context.Context, addressID, orderID int64) (*ODNPort, error) {
	var devID int64
	err := s.db.QueryRow(ctx, "SELECT device_id FROM address_coverage WHERE address_id=$1", addressID).Scan(&devID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoCoverageDevice
	}
	if err != nil {
		return nil, fmt.Errorf("odn: port coverage read: %w", err)
	}
	if devID == 0 {
		return nil, ErrNoCoverageDevice
	}
	return s.AllocatePort(ctx, devID, orderID)
}
