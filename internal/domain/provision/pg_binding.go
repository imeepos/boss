package provision

// 产品/套餐 ↔ 下发模板显式绑定(方案B,adopted note 2026-09-01-offer-provision-binding)。
// 后台把套餐和下发模板绑死,环节7 优先走绑定;绑定缺失才按带宽兜底,
// 杜绝"买套餐开错模板/无绑定开不了"。域边界:绑定表归 provision 域,
// product_offers 仅软引用读取(FindTemplateForOffer 同域已有此读路径)。

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// GetOfferBinding 查询套餐已绑定的下发模板;未绑定返回 (nil, nil)。
func (s *PGStore) GetOfferBinding(ctx context.Context, offerID int64) (*OfferTemplateBinding, error) {
	var b OfferTemplateBinding
	err := s.db.QueryRow(ctx, `
		SELECT b.id, b.legal_entity_id, b.offer_id, b.template_id,
		       COALESCE(t.code,''), COALESCE(t.name,''), COALESCE(b.remark,'')
		FROM offer_provision_bindings b
		LEFT JOIN provision_templates t ON t.id = b.template_id
		WHERE b.offer_id = $1`, offerID).
		Scan(&b.ID, &b.LegalEntityID, &b.OfferID, &b.TemplateID,
			&b.TemplateCode, &b.TemplateName, &b.Remark)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("provision: get offer binding: %w", err)
	}
	return &b, nil
}

// ErrBindingInvalid 绑定校验失败(跨法人/模板停用);httpapi 映射 42200 并透传原因。
var ErrBindingInvalid = errors.New("provision: invalid binding")

// UpsertOfferBinding 绑定/改绑套餐→模板(幂等,同套餐 ON CONFLICT 更新)。
// 校验:套餐与模板均存在、属同一法人、模板 ENABLED(禁用模板不可绑,防环节7 静默降级)。
func (s *PGStore) UpsertOfferBinding(ctx context.Context, offerID, templateID int64, remark string) (int64, error) {
	var offerEntity, tplEntity int64
	var tplStatus string
	err := s.db.QueryRow(ctx, `
		SELECT o.legal_entity_id, t.legal_entity_id, t.status
		FROM product_offers o, provision_templates t
		WHERE o.id = $1 AND t.id = $2`, offerID, templateID).
		Scan(&offerEntity, &tplEntity, &tplStatus)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("provision: bind offer=%d template=%d: %w", offerID, templateID, ErrForeignKeyViolation)
	}
	if err != nil {
		return 0, fmt.Errorf("provision: bind validate: %w", err)
	}
	if offerEntity != tplEntity {
		return 0, fmt.Errorf("provision: bind entity mismatch: offer entity=%d template entity=%d: %w",
			offerEntity, tplEntity, ErrBindingInvalid)
	}
	if tplStatus != "ENABLED" {
		return 0, fmt.Errorf("provision: bind disabled template: template=%d status=%s: %w",
			templateID, tplStatus, ErrBindingInvalid)
	}

	var id int64
	err = s.db.QueryRow(ctx, `
		INSERT INTO offer_provision_bindings(legal_entity_id, offer_id, template_id, remark)
		VALUES($1,$2,$3,$4)
		ON CONFLICT (offer_id) DO UPDATE SET
			template_id = EXCLUDED.template_id, remark = EXCLUDED.remark, updated_at = now()
		RETURNING id`,
		offerEntity, offerID, templateID, remark).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("provision: upsert offer binding: %w", err)
	}
	return id, nil
}

// DeleteOfferBinding 解绑套餐→模板;未绑定视为成功(幂等)。
func (s *PGStore) DeleteOfferBinding(ctx context.Context, offerID int64) error {
	if _, err := s.db.Exec(ctx,
		`DELETE FROM offer_provision_bindings WHERE offer_id = $1`, offerID); err != nil {
		return fmt.Errorf("provision: delete offer binding: %w", err)
	}
	return nil
}

// ListOfferBindings 列出全部绑定(admin 产品页一次性加载,渲染"下发模板"列)。
func (s *PGStore) ListOfferBindings(ctx context.Context) ([]OfferTemplateBinding, error) {
	rows, err := s.db.Query(ctx, `
		SELECT b.id, b.legal_entity_id, b.offer_id, b.template_id,
		       COALESCE(t.code,''), COALESCE(t.name,''), COALESCE(b.remark,'')
		FROM offer_provision_bindings b
		LEFT JOIN provision_templates t ON t.id = b.template_id
		ORDER BY b.offer_id`)
	if err != nil {
		return nil, fmt.Errorf("provision: list offer bindings: %w", err)
	}
	defer rows.Close()
	out := make([]OfferTemplateBinding, 0)
	for rows.Next() {
		var b OfferTemplateBinding
		if err := rows.Scan(&b.ID, &b.LegalEntityID, &b.OfferID, &b.TemplateID,
			&b.TemplateCode, &b.TemplateName, &b.Remark); err != nil {
			return nil, fmt.Errorf("provision: scan offer binding: %w", err)
		}
		out = append(out, b)
	}
	return out, rows.Err()
}
