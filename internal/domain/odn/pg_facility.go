package odn

import (
	"context"
	"fmt"
)

// CreateFacility 新建基础设施:校验编码格式(红线 3)+ 网格归属(红线 2:须已备案)
// + 网格容量(规范 4.7:单网格 999)。
func (s *PGStore) CreateFacility(ctx context.Context, f Facility) error {
	kind, grid, _, err := ValidateFacilityCode(f.Code)
	if err != nil {
		return err
	}
	if kind != f.Kind {
		return fmt.Errorf("%w: kind %s 与编码前缀不符", ErrInvalidCode, f.Kind)
	}
	if kind == KindPole || kind == KindManhole {
		if grid != f.GridCode {
			return fmt.Errorf("%w: 编码网格段 %02d 与所属网格 %02d 不符", ErrInvalidCode, grid, f.GridCode)
		}
		return s.insertGridFacility(ctx, f)
	}
	return s.insertSeqFacility(ctx, f)
}

// insertGridFacility 电杆/人井入库(网格须 ACTIVE 且容量未满)。
func (s *PGStore) insertGridFacility(ctx context.Context, f Facility) error {
	if err := s.checkGridForInsert(ctx, f); err != nil {
		return err
	}
	_, err := s.db.Exec(ctx, `INSERT INTO odn_facility
			(code, kind, prv_code, city_prefix, grid_code, name, lat, lng)
		VALUES ($1,$2,$3,$4,$5,$6,NULLIF($7,0.0),NULLIF($8,0.0))`,
		f.Code, f.Kind, f.PrvCode, f.CityPrefix, f.GridCode, f.Name, f.Lat, f.Lng)
	if err != nil {
		return mapErr(fmt.Errorf("odn: create facility: %w", err), ErrNotFound)
	}
	return nil
}

// checkGridForInsert 校验网格已备案(红线 2)且容量未满(规范 4.3:单网格 999)。
func (s *PGStore) checkGridForInsert(ctx context.Context, f Facility) error {
	var active bool
	var used int64
	err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM odn_grid
			WHERE prv_code=$1 AND city_prefix=$2 AND grid_code=$3 AND status='ACTIVE'),
		(SELECT count(*) FROM odn_facility WHERE prv_code=$1 AND city_prefix=$2 AND grid_code=$3)`,
		f.PrvCode, f.CityPrefix, f.GridCode).Scan(&active, &used)
	if err != nil {
		return fmt.Errorf("odn: check grid: %w", err)
	}
	if !active {
		return ErrGridMissing
	}
	if used >= MaxFacilities {
		return ErrGridFull
	}
	return nil
}

// insertSeqFacility 铁塔/接头盒/终端盒入库(市域顺序,无网格)。
func (s *PGStore) insertSeqFacility(ctx context.Context, f Facility) error {
	_, err := s.db.Exec(ctx, `INSERT INTO odn_facility
			(code, kind, prv_code, city_prefix, grid_code, name, lat, lng)
		VALUES ($1,$2,$3,$4,NULL,$5,NULLIF($6,0.0),NULLIF($7,0.0))`,
		f.Code, f.Kind, f.PrvCode, f.CityPrefix, f.Name, f.Lat, f.Lng)
	if err != nil {
		return mapErr(fmt.Errorf("odn: create facility: %w", err), ErrNotFound)
	}
	return nil
}

// GetFacility 按编码查询设施。
func (s *PGStore) GetFacility(ctx context.Context, code string) (*Facility, error) {
	var f Facility
	var lat, lng *float64
	err := s.db.QueryRow(ctx, `SELECT code, kind, prv_code, city_prefix,
			COALESCE(grid_code,0), COALESCE(name,''), lat, lng, status, lifecycle_status
		FROM odn_facility WHERE code=$1`, code).
		Scan(&f.Code, &f.Kind, &f.PrvCode, &f.CityPrefix, &f.GridCode, &f.Name, &lat, &lng, &f.Status, &f.LifecycleStatus)
	if err != nil {
		return nil, mapErr(err, ErrNotFound)
	}
	if lat != nil {
		f.Lat = *lat
	}
	if lng != nil {
		f.Lng = *lng
	}
	return &f, nil
}

// ListFacilities 设施列表(kind 可空=全部;GridRef 可空=不限网格)。
func (s *PGStore) ListFacilities(ctx context.Context, kind string, gridFilter GridRef) ([]Facility, error) {
	sql := `SELECT f.code, f.kind, f.prv_code, f.city_prefix, COALESCE(f.grid_code,0),
		COALESCE(f.name,''), COALESCE(f.lat,0), COALESCE(f.lng,0), f.status, f.lifecycle_status,
		COALESCE(r.registration_no,''), COALESCE(r.asset_id,0), COALESCE(a.asset_code,''), COALESCE(a.status,'')
		FROM odn_facility f
		LEFT JOIN odn_asset_registrations r ON r.entity_kind='FACILITY' AND r.facility_code=f.code AND r.status='ACTIVE'
		LEFT JOIN assets a ON a.id=r.asset_id
		WHERE ($1='' OR f.kind=$1) AND ($2='' OR (f.prv_code=$2 AND f.city_prefix=$3 AND f.grid_code=$4))
		ORDER BY f.code LIMIT 500`
	if gridFilter.PrvCode == "" {
		gridFilter = GridRef{PrvCode: "", CityPrefix: "", GridCode: 0}
	}
	rows, err := s.db.Query(ctx, sql, kind, gridFilter.PrvCode, gridFilter.CityPrefix, gridFilter.GridCode)
	if err != nil {
		return nil, fmt.Errorf("odn: list facilities: %w", err)
	}
	defer rows.Close()
	out := []Facility{}
	for rows.Next() {
		var f Facility
		var regNo, assetCode, assetStatus string
		var regAssetID int64
		if err := rows.Scan(&f.Code, &f.Kind, &f.PrvCode, &f.CityPrefix,
			&f.GridCode, &f.Name, &f.Lat, &f.Lng, &f.Status, &f.LifecycleStatus,
			&regNo, &regAssetID, &assetCode, &assetStatus); err != nil {
			return nil, fmt.Errorf("odn: scan facility: %w", err)
		}
		if regNo != "" {
			f.AssetReg = &EntityAssetReg{RegistrationNo: regNo, AssetID: regAssetID, AssetCode: assetCode, AssetStatus: assetStatus}
		}
		out = append(out, f)
	}
	return out, rows.Err()
}

// RetireFacility 设施报废:编码永久锁定,行保留不可复用(资产编码规范红线 2)。
func (s *PGStore) RetireFacility(ctx context.Context, code string) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE odn_facility SET status='RETIRED' WHERE code=$1 AND status='IN_USE'`, code)
	if err != nil {
		return fmt.Errorf("odn: retire facility: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
