package resource

import (
	"context"
	"fmt"
)

// ListTransfers 列出全部调拨单。
func (s *PGStore) ListTransfers(ctx context.Context) ([]Transfer, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, transfer_no, resource_id, legal_entity_id, legal_entity_name, from_region_id, to_region_id, status
		 FROM transfers ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("resource: list transfers: %w", err)
	}
	defer rows.Close()
	out := make([]Transfer, 0)
	for rows.Next() {
		var t Transfer
		if err := rows.Scan(&t.ID, &t.TransferNo, &t.ResourceID, &t.LegalEntityID, &t.LegalEntityName, &t.FromRegionID, &t.ToRegionID, &t.Status); err != nil {
			return nil, fmt.Errorf("resource: scan transfer: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// CreateTransfer 新建调拨单,返回自增 id。
// 校验 resource_id 存在性,防止孤儿调拨单。
func (s *PGStore) CreateTransfer(ctx context.Context, t Transfer) (int64, error) {
	// 关联完整性校验
	if t.ResourceID > 0 {
		ok, err := s.exists(ctx, "resources", t.ResourceID)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("resource: resource %d: %w", t.ResourceID, ErrForeignKeyViolation)
		}
	}

	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO transfers(transfer_no, resource_id, legal_entity_id, legal_entity_name, from_region_id, to_region_id, status)
		VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		t.TransferNo, t.ResourceID, t.LegalEntityID, t.LegalEntityName, t.FromRegionID, t.ToRegionID, t.Status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("resource: create transfer: %w", err)
	}
	return id, nil
}

// ListExpansions 列出全部扩容单。
func (s *PGStore) ListExpansions(ctx context.Context) ([]Expansion, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, legal_entity_id, expansion_no, region_id, expected_ports, status FROM expansions ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("resource: list expansions: %w", err)
	}
	defer rows.Close()
	out := make([]Expansion, 0)
	for rows.Next() {
		var e Expansion
		if err := rows.Scan(&e.ID, &e.LegalEntityID, &e.ExpansionNo, &e.RegionID, &e.ExpectedPorts, &e.Status); err != nil {
			return nil, fmt.Errorf("resource: scan expansion: %w", err)
		}
		out = append(out, e)
	}
	return out, rows.Err()
}

// CreateExpansion 新建扩容单,返回自增 id。
// 校验 legal_entity_id 存在性,防止孤儿扩容单。
func (s *PGStore) CreateExpansion(ctx context.Context, e Expansion) (int64, error) {
	// 关联完整性校验
	if e.LegalEntityID > 0 {
		ok, err := s.exists(ctx, "legal_entities", e.LegalEntityID)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("resource: legal entity %d: %w", e.LegalEntityID, ErrForeignKeyViolation)
		}
	}

	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO expansions(legal_entity_id, expansion_no, region_id, expected_ports, status)
		VALUES($1,$2,$3,$4,$5) RETURNING id`,
		e.LegalEntityID, e.ExpansionNo, e.RegionID, e.ExpectedPorts, e.Status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("resource: create expansion: %w", err)
	}
	return id, nil
}

// ListReserveRecords 列出预占记录;portID=0 返回全部。
func (s *PGStore) ListReserveRecords(ctx context.Context, portID int64) ([]ReserveRecord, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, port_id, order_id, status FROM reserve_records WHERE ($1 = 0 OR port_id = $1) ORDER BY id`, portID)
	if err != nil {
		return nil, fmt.Errorf("resource: list reserve records: %w", err)
	}
	defer rows.Close()
	out := make([]ReserveRecord, 0)
	for rows.Next() {
		var r ReserveRecord
		if err := rows.Scan(&r.ID, &r.PortID, &r.OrderID, &r.Status); err != nil {
			return nil, fmt.Errorf("resource: scan reserve record: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// AppendReserveRecord 追加预占记录,返回自增 id。
func (s *PGStore) AppendReserveRecord(ctx context.Context, r ReserveRecord) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO reserve_records(port_id, order_id, status) VALUES($1,$2,$3) RETURNING id`,
		r.PortID, r.OrderID, r.Status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("resource: append reserve record: %w", err)
	}
	return id, nil
}

// ListPortHistory 列出端口变更历史;portID=0 返回全部。
func (s *PGStore) ListPortHistory(ctx context.Context, portID int64) ([]PortChangeHistory, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, port_id, status, COALESCE(order_id, 0), changed_at
		 FROM port_change_history WHERE ($1 = 0 OR port_id = $1) ORDER BY changed_at, id`, portID)
	if err != nil {
		return nil, fmt.Errorf("resource: list port history: %w", err)
	}
	defer rows.Close()
	out := make([]PortChangeHistory, 0)
	for rows.Next() {
		var h PortChangeHistory
		if err := rows.Scan(&h.ID, &h.PortID, &h.Status, &h.OrderID, &h.ChangedAt); err != nil {
			return nil, fmt.Errorf("resource: scan port history: %w", err)
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// AppendPortHistory 追加端口变更历史,返回自增 id。
func (s *PGStore) AppendPortHistory(ctx context.Context, h PortChangeHistory) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO port_change_history(port_id, status, order_id, changed_at) VALUES($1,$2,$3,$4) RETURNING id`,
		h.PortID, h.Status, idOrNil(h.OrderID), h.ChangedAt).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("resource: append port history: %w", err)
	}
	return id, nil
}
