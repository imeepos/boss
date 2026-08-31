package provision

// 环节7 模板解析(方案B,adopted note 2026-09-01-offer-provision-binding):
// 显式绑定优先 → 带宽兜底 → 显性失败。
// 原 2026-08-30 裁定只按带宽匹配 + 回退法人最旧模板,实测无带宽套餐(IPTV/云存储)
// 全部回退到测试模板、有套餐无模板法人卡死环节7。现改为:后台绑定走绑定,
// 无绑定才按带宽猜,都命中不了直接报错禁止静默降级。

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// FindTemplateForOffer 套餐 → 下发模板 ID。
// 命中顺序:①显式绑定(offer_provision_bindings,须模板 ENABLED);
// ②带宽兜底(同法人 ENABLED 模板 content->>'bandwidth' = 套餐带宽,无带宽套餐不参与);
// ③显性失败(禁止回退法人任意模板——那是开错配置的根源)。
func (s *PGStore) FindTemplateForOffer(ctx context.Context, offerID, legalEntityID int64) (int64, error) {
	// ①显式绑定:套餐在后台绑了模板就走绑定,不再猜测。
	if id, ok, err := s.boundTemplateID(ctx, offerID); err != nil {
		return 0, err
	} else if ok {
		return id, nil
	}

	// ②带宽兜底:无带宽套餐(IPTV/增值包)不参与,须显式绑定。
	var id int64
	err := s.db.QueryRow(ctx, `
		SELECT t.id FROM provision_templates t
		JOIN product_offers o ON o.id = $1
		WHERE t.status = 'ENABLED' AND t.legal_entity_id = $2
		  AND o.bandwidth <> '' AND t.content->>'bandwidth' = o.bandwidth
		ORDER BY t.id LIMIT 1`, offerID, legalEntityID).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("provision: resolve template by bandwidth offer=%d: %w", offerID, err)
	}

	// ③显性失败:既不绑也没档位模板,禁止静默降级到任意模板。
	return 0, fmt.Errorf(
		"[provision] TEMPLATE UNRESOLVED: offer=%d entity=%d (no binding, no bandwidth match)", offerID, legalEntityID)
}

// boundTemplateID 取套餐显式绑定的模板 ID。
// 绑定行存在但模板被禁用 → 显性报错(配置错误),不静默落到带宽兜底。
func (s *PGStore) boundTemplateID(ctx context.Context, offerID int64) (int64, bool, error) {
	var id int64
	var status string
	err := s.db.QueryRow(ctx, `
		SELECT t.id, t.status FROM offer_provision_bindings b
		JOIN provision_templates t ON t.id = b.template_id
		WHERE b.offer_id = $1`, offerID).Scan(&id, &status)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, false, nil
	}
	if err != nil {
		return 0, false, fmt.Errorf("provision: get offer binding: %w", err)
	}
	if status != "ENABLED" {
		return 0, false, fmt.Errorf(
			"[provision] TEMPLATE BOUND BUT DISABLED: offer=%d template=%d status=%s", offerID, id, status)
	}
	return id, true, nil
}
