// 型号字典读写(P1-T3,adopted note 2026-09-06-asset-tag-p1-wave):
// 形态依据 R3 调研(NetBox DeviceType/GLPI models)——UNIQUE 防重、停用不物理删、
// 引用由 assets.model_id 外键承载。冲突映射 ErrModelExists(40900 透传 reason)。

package asset

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
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

// ErrModelInactive 型号已停用,拒绝编辑(P2-W2-T1 D:先启用再改)。
var ErrModelInactive = errors.New("asset: model inactive")

// UpdateModel 编辑型号(P2-W2-T1 D):厂商/型号名/类别/料号/规格可改;
// 四元组 UNIQUE(uq_asset_models) 冲突 23505 → ErrModelExists(40900);
// 停用型号不可编辑 → ErrModelInactive(40900,先启用)。停用闸门用
// FOR UPDATE 行锁,防编辑与停用并发交错。
func (s *PGStore) UpdateModel(ctx context.Context, id int64, m AssetModel) error {
	var active bool
	err := s.db.QueryRow(ctx,
		`SELECT is_active FROM asset_models WHERE id = $1 FOR UPDATE`, id).Scan(&active)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("asset: update model %d lock: %w", id, err)
	}
	if !active {
		return fmt.Errorf("asset: model %d deactivated, enable first: %w", id, ErrModelInactive)
	}
	_, err = s.db.Exec(ctx,
		`UPDATE asset_models SET vendor = $2, model = $3, category = $4, part_number = $5, spec = $6
		 WHERE id = $1`,
		id, m.Vendor, m.Model, m.Category, m.PartNumber, m.Spec)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" && pgErr.ConstraintName == "uq_asset_models" {
			return fmt.Errorf("asset: model %s/%s: %w", m.Vendor, m.Model, ErrModelExists)
		}
		return fmt.Errorf("asset: update model %d: %w", id, err)
	}
	return nil
}

// SetModelActive 型号停用/启用(P2-W2-T1 E):is_active 置否/置真。
// 停用不物理删,已被资产引用由 assets.model_id 承载(P1-T3 契约);
// 同值重复置位幂等成功;未命中 ErrNotFound。
func (s *PGStore) SetModelActive(ctx context.Context, id int64, active bool) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE asset_models SET is_active = $2 WHERE id = $1`, id, active)
	if err != nil {
		return fmt.Errorf("asset: set model %d active=%v: %w", id, active, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
