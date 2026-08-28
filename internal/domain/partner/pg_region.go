package partner

import (
	"context"
	"fmt"
)

func (s *PGStore) GetRegionScope(ctx context.Context, accountID int64) (RegionScope, error) {
	var scope RegionScope
	err := s.db.QueryRow(ctx, `SELECT COALESCE(legal_entity_id,0), COALESCE(region_scope::text,'') FROM accounts WHERE id=$1`, accountID).Scan(&scope.LegalEntityID, &scope.RegionPath)
	if err != nil {
		return scope, fmt.Errorf("partner: get region scope: %w", err)
	}
	if scope.LegalEntityID == 0 {
		return scope, ErrNotPartner
	}
	return scope, nil
}

func (s *PGStore) SetRegionScope(ctx context.Context, adminAccountID int64, regionPath string) error {
	entityID, err := s.entityOfAccount(ctx, adminAccountID)
	if err != nil {
		return err
	}
	if entityID == 0 {
		return ErrNotPartner
	}
	// 清空 scope 是合法操作:regionPath 为空时跳过归属校验,直接落 NULL。
	// COALESCE 兜公共区域 legal_entity_id 为 NULL 的行:=$2/=0 均匹配不上会误判越界。
	if regionPath != "" {
		var ok bool
		err = s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM regions WHERE path = $1 AND COALESCE(legal_entity_id,0) IN (0,$2))`, regionPath, entityID).Scan(&ok)
		if err != nil {
			return err
		}
		if !ok {
			return ErrRegionOutsideEnterprise
		}
	}
	_, err = s.db.Exec(ctx, `UPDATE accounts SET region_scope=NULLIF($1,'')::ltree, updated_at=now() WHERE id=$2 AND legal_entity_id=$3`, regionPath, adminAccountID, entityID)
	return err
}
