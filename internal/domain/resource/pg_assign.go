package resource

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
)

// ListAssignments 列出设备归属台账;resourceID=0 返回全部。
func (s *PGStore) ListAssignments(ctx context.Context, resourceID int64) ([]ResourceAssignment, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, resource_id, legal_entity_id, legal_entity_name,
		       COALESCE(address_id, 0), COALESCE(address_name, ''),
		       COALESCE(region_id, 0), COALESCE(region_name, ''),
		       COALESCE(reason, ''), COALESCE(operator_account_id, 0),
		       effective_from, effective_to
		FROM resource_assignments WHERE ($1 = 0 OR resource_id = $1) ORDER BY effective_from, id`, resourceID)
	if err != nil {
		return nil, fmt.Errorf("resource: list assignments: %w", err)
	}
	defer rows.Close()
	out := make([]ResourceAssignment, 0)
	for rows.Next() {
		var a ResourceAssignment
		var effTo pgtype.Timestamptz
		if err := rows.Scan(&a.ID, &a.ResourceID, &a.LegalEntityID, &a.LegalEntityName,
			&a.AddressID, &a.AddressName, &a.RegionID, &a.RegionName, &a.Reason, &a.OperatorAccountID,
			&a.EffectiveFrom, &effTo); err != nil {
			return nil, fmt.Errorf("resource: scan assignment: %w", err)
		}
		if effTo.Valid {
			t := effTo.Time
			a.EffectiveTo = &t
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// AppendAssignment 追加设备归属台账,返回自增 id。
func (s *PGStore) AppendAssignment(ctx context.Context, a ResourceAssignment) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO resource_assignments(resource_id, legal_entity_id, legal_entity_name,
			address_id, address_name, region_id, region_name, reason, operator_account_id,
			effective_from, effective_to)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`,
		a.ResourceID, a.LegalEntityID, a.LegalEntityName,
		idOrNil(a.AddressID), a.AddressName, idOrNil(a.RegionID), a.RegionName, a.Reason, idOrNil(a.OperatorAccountID),
		a.EffectiveFrom, a.EffectiveTo).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("resource: append assignment: %w", err)
	}
	return id, nil
}

// ListQosTemplates 列出 QoS 模板;legalEntityID=0 返回全部。
func (s *PGStore) ListQosTemplates(ctx context.Context, legalEntityID int64) ([]QosTemplate, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, legal_entity_id, code, name FROM qos_templates WHERE ($1 = 0 OR legal_entity_id = $1) ORDER BY id`, legalEntityID)
	if err != nil {
		return nil, fmt.Errorf("resource: list qos: %w", err)
	}
	defer rows.Close()
	out := make([]QosTemplate, 0)
	for rows.Next() {
		var q QosTemplate
		if err := rows.Scan(&q.ID, &q.LegalEntityID, &q.Code, &q.Name); err != nil {
			return nil, fmt.Errorf("resource: scan qos: %w", err)
		}
		out = append(out, q)
	}
	return out, rows.Err()
}

// CreateQosTemplate 新增 QoS 模板,返回自增 id。
func (s *PGStore) CreateQosTemplate(ctx context.Context, q QosTemplate) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO qos_templates(legal_entity_id, code, name) VALUES($1,$2,$3) RETURNING id`,
		q.LegalEntityID, q.Code, q.Name).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("resource: create qos: %w", err)
	}
	return id, nil
}
