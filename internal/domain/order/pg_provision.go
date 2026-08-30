package order

// 环节7 预下发配置(从 pg_workflow.go 拆出,守 300 行红线)。
// 模板解析修复:此前直接把 product_offers.id 当 provision_templates.id 传给下发任务——
// 两表 ID 空间重叠时静默套错模板(102 实测:100M 套餐全落到 TPL-TN tnet 模板),
// 新套餐无同号模板行则外键违规卡死环节7。解析规则见 adopted note(preconfig-template-resolution)。

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// PreConfigOLT 环节7 预下发配置:按订单套餐解析模板,幂等创建 provision 任务。
func (s *PGStore) PreConfigOLT(ctx context.Context, orderID int64) error {
	if s.prov == nil {
		return errors.New("order: provision task creator not wired")
	}
	if s.prof == nil {
		return errors.New("order: user profile creator not wired for provisioning")
	}
	if s.tpl == nil {
		return errors.New("order: provision template finder not wired")
	}
	var customerID, legalEntityID int64
	err := s.db.QueryRow(ctx,
		`SELECT customer_id, legal_entity_id FROM orders WHERE id = $1`, orderID).
		Scan(&customerID, &legalEntityID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrOrderNotFound
		}
		return fmt.Errorf("order: preConfigOLT select: %w", err)
	}
	lo, err := s.prof.GetLoAccountByCustomer(ctx, customerID)
	if err != nil {
		return fmt.Errorf("order: preConfigOLT get lo: %w", err)
	}
	if lo == nil {
		return fmt.Errorf("order: preConfigOLT: no lo account for customer %d", customerID)
	}
	tplID, err := s.tpl.FindTemplateForOffer(ctx, lo.OfferID, legalEntityID)
	if err != nil {
		return fmt.Errorf("order: preConfigOLT resolve template: %w", err)
	}
	taskNo := fmt.Sprintf("PRV-O%d", orderID)
	if _, err := s.prov.CreateTask(ctx, ProvisionTask{
		TaskNo: taskNo, OrderID: orderID, StageEvent: "preConfigOLT",
		LoAccountID: lo.ID, TemplateID: tplID, Status: "PENDING",
	}); err != nil {
		return fmt.Errorf("order: preConfigOLT create task: %w", err)
	}
	return s.advance(ctx, orderID, "preConfigOLT")
}
