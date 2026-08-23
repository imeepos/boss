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
	var ok bool
	err = s.db.QueryRow(ctx, `SELECT EXISTS(SELECT 1 FROM regions WHERE path = $1 AND (legal_entity_id=$2 OR legal_entity_id=0))`, regionPath, entityID).Scan(&ok)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("partner: region outside enterprise")
	}
	_, err = s.db.Exec(ctx, `UPDATE accounts SET region_scope=NULLIF($1,'')::ltree, updated_at=now() WHERE id=$2 AND legal_entity_id=$3`, regionPath, adminAccountID, entityID)
	return err
}
