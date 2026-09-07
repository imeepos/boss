package aaa

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ErrDuplicate LOID 唯一冲突(管理端建号;DB lo_accounts.loid UNIQUE 兜底并发窗口)。
var ErrDuplicate = errors.New("aaa: duplicate loid")

// ErrOfferNotPublished 产品非 PUBLISHED 态,不可开号(引用存在但状态不符,40900 语义)。
var ErrOfferNotPublished = errors.New("aaa: offer not published")

// LoAccountAdminService 管理端建号扩展(POST /lo-accounts);环节 6 走既有 CreateLoAccount 不受影响。
type LoAccountAdminService interface {
	CreateLoAccountChecked(ctx context.Context, a LoAccount) (int64, error)
}

// CreateLoAccountChecked 管理端建号前置校验版:
// loid 预查唯一(DB UNIQUE 兜底并发)、offer 须 PUBLISHED、qos_template 软引用存在、
// 法人缺省兜底平台总公司(is_platform,000077;区域缺省 0/空名对齐导入裁定)。
// 校验通过后复用既有 CreateLoAccount(customer/offer/legal 存在性 + billing 继承)。
func (s *PGStore) CreateLoAccountChecked(ctx context.Context, a LoAccount) (int64, error) {
	var loidTaken bool
	if err := s.db.QueryRow(ctx, "SELECT EXISTS(SELECT 1 FROM lo_accounts WHERE loid = $1)", a.Loid).Scan(&loidTaken); err != nil {
		return 0, fmt.Errorf("aaa: check loid: %w", err)
	}
	if loidTaken {
		return 0, fmt.Errorf("aaa: loid %s: %w", a.Loid, ErrDuplicate)
	}
	if a.OfferID > 0 {
		var status string
		err := s.db.QueryRow(ctx, "SELECT status FROM product_offers WHERE id = $1", a.OfferID).Scan(&status)
		if errors.Is(err, pgx.ErrNoRows) {
			return 0, fmt.Errorf("aaa: offer %d: %w", a.OfferID, ErrForeignKeyViolation)
		}
		if err != nil {
			return 0, fmt.Errorf("aaa: check offer: %w", err)
		}
		if status != "PUBLISHED" {
			return 0, fmt.Errorf("aaa: offer %d status %s: %w", a.OfferID, status, ErrOfferNotPublished)
		}
	}
	if a.QosTemplateID > 0 {
		ok, err := s.exists(ctx, "qos_templates", a.QosTemplateID)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("aaa: qos_template %d: %w", a.QosTemplateID, ErrForeignKeyViolation)
		}
	}
	if a.LegalEntityID <= 0 {
		id, name, err := s.platformEntity(ctx)
		if err != nil {
			return 0, err
		}
		a.LegalEntityID, a.LegalEntityName = id, name
	}
	return s.CreateLoAccount(ctx, a)
}

// platformEntity 取平台总公司(is_platform 全库唯一,000077);未配置即兜底链断裂。
func (s *PGStore) platformEntity(ctx context.Context) (int64, string, error) {
	var id int64
	var name string
	err := s.db.QueryRow(ctx, "SELECT id, name FROM legal_entities WHERE is_platform = TRUE LIMIT 1").Scan(&id, &name)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", fmt.Errorf("aaa: platform legal entity missing: %w", ErrForeignKeyViolation)
	}
	if err != nil {
		return 0, "", fmt.Errorf("aaa: platform legal entity: %w", err)
	}
	return id, name, nil
}
