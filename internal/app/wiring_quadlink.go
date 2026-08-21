package app

// quadLinkPrebinder 把 quadlink.QuadLinkService.CreateLink 适配为 order.QuadLinkPrebinder.CreateLink。
// app 装配层做转换,order 域不 import quadlink 域(依赖倒置)。

import (
	"context"

	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/quadlink"
)

type quadLinkPrebinder struct{ svc quadlink.QuadLinkService }

func (p quadLinkPrebinder) CreateLink(ctx context.Context, q order.QuadLinkBindReq) (int64, error) {
	return p.svc.CreateLink(ctx, quadlink.QuadLink{
		AssetID:         q.AssetID,
		CustomerID:      q.CustomerID,
		PortID:          q.PortID,
		AddressID:       q.AddressID,
		LegalEntityID:   q.LegalEntityID,
		LegalEntityName: q.LegalEntityName,
		Status:          q.Status,
	})
}
