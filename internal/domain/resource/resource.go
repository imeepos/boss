package resource

import "context"

// Resource 网络设备(OLT/分光器,各公司建设的设备树)。
type Resource struct {
	ID            int64  `json:"id"`
	LegalEntityID int64  `json:"legalEntityId"`
	Code          string `json:"code"` // OLT-01/SPL-01
	Name          string `json:"name"`
	Type          string `json:"type"`     // OLT/SPLITTER
	ParentID      int64  `json:"parentId"` // 0=无上级(OLT 为根)
	AddressID     int64  `json:"addressId"`
	Status        string `json:"status"` // ONLINE/OFFLINE/FAULT
}

// Port 端口(挂分光器,占用态必带订单)。
type Port struct {
	PortID          int64  `json:"portId"`
	PortCode        string `json:"portCode"` // P-SPL01-01
	QuadCode        string `json:"quadCode"` // 四码端口码
	ResourceID      int64  `json:"resourceId"`
	LegalEntityID   int64  `json:"legalEntityId"`
	LegalEntityName string `json:"legalEntityName"`
	AddressID       int64  `json:"addressId"`
	RegionID        int64  `json:"regionId"`
	RegionName      string `json:"regionName"`
	OrderID         int64  `json:"orderId"` // 0=空闲
	Status          string `json:"status"`  // IDLE/RESERVED/USED/DISABLED
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
	// ReserveFirstAvailable 在目标地址找一个空闲端口并预占给订单;返回端口ID;无空闲返回 ErrPortNotAvailable。
	ReserveFirstAvailable(ctx context.Context, addressID, orderID int64) (int64, error)
	// ReleasePortByOrder 端口释放(取消/超时回滚):回收挂在本订单上的 RESERVED 端口;无匹配返回 ErrPortNotAvailable。
	ReleasePortByOrder(ctx context.Context, orderID int64) error
	// Check 资源核查(环节2):目标地址是否有空闲端口;options 为空闲端口码列表。
	Check(ctx context.Context, addressID int64) (available bool, options []string, err error)
}
