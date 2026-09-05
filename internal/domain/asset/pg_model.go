// 型号字典读写(P1-T3,adopted note 2026-09-06-asset-tag-p1-wave):
// 形态依据 R3 调研(NetBox DeviceType/GLPI models)——UNIQUE 防重、停用不物理删、
// 引用由 assets.model_id 外键承载。冲突映射 ErrModelExists(40900 透传 reason)。

package asset

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ListModels 全量型号(含停用;管理端下拉与列表用,量级小不分页)。
func (s *PGStore) ListModels(ctx context.Context) ([]AssetModel, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, vendor, model, category, part_number, spec, is_active, created_at
		 FROM asset_models ORDER BY is_active DESC, vendor, model`)
	if err != nil {
		return nil, fmt.Errorf("asset: list models: %w", err)
	}
	defer rows.Close()
	out := make([]AssetModel, 0)
	for rows.Next() {
		var m AssetModel
		if err := rows.Scan(&m.ID, &m.Vendor, &m.Model, &m.Category, &m.PartNumber, &m.Spec, &m.IsActive, &m.CreatedAt); err != nil {
			return nil, fmt.Errorf("asset: scan model: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// CreateModel 建型号;四元组唯一,冲突返回 ErrModelExists。
func (s *PGStore) CreateModel(ctx context.Context, m AssetModel) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO asset_models(vendor, model, category, part_number, spec)
		 VALUES($1,$2,$3,$4,$5)
		 ON CONFLICT (vendor, model, category, part_number) DO NOTHING
		 RETURNING id`,
		m.Vendor, m.Model, m.Category, m.PartNumber, m.Spec).Scan(&id)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, ErrModelExists
	}
	if err != nil {
		return 0, fmt.Errorf("asset: create model: %w", err)
	}
	return id, nil
}
