package resource

import (
	"context"
	"time"
)

// ResourceAssignment 设备归属台账(每次归属公司/区域/地址的时间段)。
type ResourceAssignment struct {
	ID                int64
	ResourceID        int64
	LegalEntityID     int64
	LegalEntityName   string
	AddressID         int64 // 0=空
	AddressName       string
	RegionID          int64 // 0=空
	RegionName        string
	Reason            string
	OperatorAccountID int64 // 0=空
	EffectiveFrom     time.Time
	EffectiveTo       *time.Time // nil=至今
}

// QosTemplate QoS 模板(挂公司,LO 账号生效载体)。
type QosTemplate struct {
	ID            int64
	LegalEntityID int64
	Code          string // QoS-VIP
	Name          string // VIP
}

// ResourceAssignService 网络资源归属台账域服务口(阶段4)。
type ResourceAssignService interface {
	ListAssignments(ctx context.Context, resourceID int64) ([]ResourceAssignment, error)
	AppendAssignment(ctx context.Context, a ResourceAssignment) (int64, error)
	ListQosTemplates(ctx context.Context, legalEntityID int64) ([]QosTemplate, error)
	CreateQosTemplate(ctx context.Context, q QosTemplate) (int64, error)
}
