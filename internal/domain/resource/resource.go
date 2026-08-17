package resource

import "context"

// Resource 网络设备(OLT/分光器,各公司建设的设备树)。
type Resource struct {
	ID            int64
	LegalEntityID int64
	Code          string // OLT-01/SPL-01
	Name          string
	Type          string // OLT/SPLITTER
	ParentID      int64  // 0=无上级(OLT 为根)
	AddressID     int64
	Status        string // ONLINE/OFFLINE/FAULT
}

// Port 端口(挂分光器,占用态必带订单)。
type Port struct {
	PortID          int64
	PortCode        string // P-SPL01-01
	QuadCode        string // 四码端口码
	ResourceID      int64
	LegalEntityID   int64
	LegalEntityName string
	AddressID       int64
	RegionID        int64
	RegionName      string
	OrderID         int64  // 0=空闲
	Status          string // IDLE/RESERVED/USED/DISABLED
}

// ResourceService 网络资源域服务口(阶段4):设备树/端口/预占。
type ResourceService interface {
	ListResources(ctx context.Context) ([]Resource, error)
	CreateResource(ctx context.Context, r Resource) (int64, error)
	GetResource(ctx context.Context, id int64) (*Resource, error)

	ListPorts(ctx context.Context, resourceID int64) ([]Port, error)
	CreatePort(ctx context.Context, p Port) (int64, error)
	// ReservePort 端口预占:仅 IDLE 可预占为 RESERVED,并挂订单;失败返回 ErrPortNotAvailable。
	ReservePort(ctx context.Context, portID, orderID int64) error
}
