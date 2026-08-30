package provision

// 环节7 模板解析:按套餐带宽匹配同法人 ENABLED 模板,无命中回退法人默认模板并留痕。
// 规则裁定(adopted note preconfig-template-resolution):不加 offer→template 外键列,
// 用 content->>'bandwidth' 匹配——模板按公司/设备型号各异(000127 content 自定义),
// 带宽匹配让模板与套餐解耦维护,新增套餐零配置即可命中既有档位模板。

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
)

// FindTemplateForOffer 套餐 → 下发模板 ID。
// 命中顺序:带宽匹配(content->>'bandwidth' = product_offers.bandwidth)→ 法人默认(最旧 ENABLED)。
func (s *PGStore) FindTemplateForOffer(ctx context.Context, offerID, legalEntityID int64) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		SELECT t.id FROM provision_templates t
		JOIN product_offers o ON o.id = $1
		WHERE t.status = 'ENABLED' AND t.legal_entity_id = $2
		  AND t.content->>'bandwidth' = o.bandwidth
		ORDER BY t.id LIMIT 1`, offerID, legalEntityID).Scan(&id)
	if err == nil {
		return id, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf("provision: resolve template by bandwidth offer=%d: %w", offerID, err)
	}
	// 带宽无匹配:回退法人默认模板;留痕可 grep,便于排查错配。
	err = s.db.QueryRow(ctx, `
		SELECT id FROM provision_templates
		WHERE status = 'ENABLED' AND legal_entity_id = $1
		ORDER BY id LIMIT 1`, legalEntityID).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, fmt.Errorf(
			"[provision] TEMPLATE MISSING: no enabled template for legal_entity=%d (offer=%d)", legalEntityID, offerID)
	}
	if err != nil {
		return 0, fmt.Errorf("provision: resolve default template: %w", err)
	}
	log.Printf("[provision] TEMPLATE FALLBACK: offer=%d entity=%d -> template=%d (no bandwidth match)", offerID, legalEntityID, id)
	return id, nil
}
