package resource

import (
	"context"
	"time"
)

// ResourceAssignment 设备归属台账(每次归属公司/区域/地址的时间段)。
type ResourceAssignment struct {
	ID                int64      `json:"id"`
	ResourceID        int64      `json:"resourceId"`
	LegalEntityID     int64      `json:"legalEntityId"`
	LegalEntityName   string     `json:"legalEntityName"`
	AddressID         int64      `json:"addressId"` // 0=空
	AddressName       string     `json:"addressName"`
	RegionID          int64      `json:"regionId"` // 0=空
	RegionName        string     `json:"regionName"`
	Reason            string     `json:"reason"`
	OperatorAccountID int64      `json:"operatorAccountId"` // 0=空
	EffectiveFrom     time.Time  `json:"effectiveFrom"`
	EffectiveTo       *time.Time `json:"effectiveTo,omitempty"` // nil=至今
}

// QosTemplate QoS 模板(挂公司,LO 账号生效载体)。
type QosTemplate struct {
	ID            int64  `json:"id"`
	LegalEntityID int64  `json:"legalEntityId"`
	Code          string `json:"code"` // QoS-VIP
	Name          string `json:"name"` // VIP
}

// ResourceAssignService 网络资源归属台账域服务口(阶段4)。
type ResourceAssignService interface {
	ListAssignments(ctx context.Context, resourceID int64) ([]ResourceAssignment, error)
	AppendAssignment(ctx context.Context, a ResourceAssignment) (int64, error)
	ListQosTemplates(ctx context.Context, legalEntityID int64) ([]QosTemplate, error)
	CreateQosTemplate(ctx context.Context, q QosTemplate) (int64, error)
}
