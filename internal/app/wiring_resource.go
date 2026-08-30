package app

// portReserver 包装 resource.PGStore 为 order.PortReserver 接口。
// *resource.PGStore 同时实现 ResourceChecker 和 PortReserver，
// 但 order.NewPGStore 的 type switch 只匹配第一个 case，
// 所以需要显式包装成不同类型以分别注入。

import (
	"context"
	"fmt"

	"github.com/ymm-001/boss/internal/domain/aaa"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/provision"
	"github.com/ymm-001/boss/internal/domain/resource"
)

type portReserver struct{ svc *resource.PGStore }

func (r portReserver) ReserveFirstAvailable(ctx context.Context, addressID, orderID int64) (int64, error) {
	return r.svc.ReserveFirstAvailable(ctx, addressID, orderID)
}

// userProfileCreator 包装 aaa.PGStore 为 order.UserProfileCreator 接口。
type userProfileCreator struct{ svc *aaa.PGStore }

func (u userProfileCreator) CreateLoAccount(ctx context.Context, lo order.LoidReq) (int64, error) {
	return u.svc.CreateLoAccount(ctx, aaa.LoAccount{
		Loid: lo.Loid, CustomerID: lo.CustomerID,
		LegalEntityID: lo.LegalEntityID, LegalEntityName: lo.LegalEntityName,
		RegionID: lo.RegionID, RegionName: lo.RegionName,
		RegionPath: lo.RegionPath, OfferID: lo.OfferID,
		Status: string(aaa.StatusActive), BillingMode: lo.BillingMode,
	})
}

func (u userProfileCreator) GetLoAccountByCustomer(ctx context.Context, customerID int64) (*order.LoidAccount, error) {
	lo, err := u.svc.GetLoAccountByCustomer(ctx, customerID)
	if err != nil {
		return nil, err
	}
	if lo == nil {
		return nil, nil
	}
	return &order.LoidAccount{ID: lo.ID, Loid: lo.Loid, OfferID: lo.OfferID}, nil
}

// provisionTaskCreator 包装 provision.PGStore 为 order.ProvisionTaskCreator 接口。
type provisionTaskCreator struct{ svc *provision.PGStore }

func (p provisionTaskCreator) CreateTask(ctx context.Context, t order.ProvisionTask) (int64, error) {
	id, err := p.svc.CreateTask(ctx, provision.Task{
		TaskNo: t.TaskNo, OrderID: t.OrderID, StageEvent: t.StageEvent,
		LoAccountID: t.LoAccountID, TemplateID: t.TemplateID, Status: t.Status,
	})
	if err != nil {
		return 0, fmt.Errorf("provision: create task: %w", err)
	}
	return id, nil
}

// FindTemplateForOffer 同一适配器实现 order.ProvisionTemplateFinder(环节7 模板解析)。
func (p provisionTaskCreator) FindTemplateForOffer(ctx context.Context, offerID, legalEntityID int64) (int64, error) {
	return p.svc.FindTemplateForOffer(ctx, offerID, legalEntityID)
}
