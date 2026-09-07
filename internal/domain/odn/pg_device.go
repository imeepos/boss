package odn

import (
	"context"
	"fmt"
)

// ListSites 局点列表(指定城市)。
func (s *PGStore) ListSites(ctx context.Context, prvCode, cityPrefix string) ([]Site, error) {
	rows, err := s.db.Query(ctx, `SELECT prv_code, city_prefix, site_no,
			COALESCE(name,''), COALESCE(lat,0), COALESCE(lng,0), status, lifecycle_status
		FROM odn_site WHERE prv_code=$1 AND city_prefix=$2 AND status <> 'RETIRED'
		ORDER BY site_no`, prvCode, cityPrefix)
	if err != nil {
		return nil, fmt.Errorf("odn: list sites: %w", err)
	}
	defer rows.Close()
	out := []Site{}
	for rows.Next() {
		var st Site
		if err := rows.Scan(&st.PrvCode, &st.CityPrefix, &st.SiteNo,
			&st.Name, &st.Lat, &st.Lng, &st.Status, &st.LifecycleStatus); err != nil {
			return nil, fmt.Errorf("odn: scan site: %w", err)
		}
		out = append(out, st)
	}
	return out, rows.Err()
}

// CreateSite 新建局点(NodeCode 由规划部分配;唯一冲突→ErrDuplicate)。
func (s *PGStore) CreateSite(ctx context.Context, st Site) error {
	_, err := s.db.Exec(ctx, `INSERT INTO odn_site (prv_code, city_prefix, site_no, name, lat, lng)
		VALUES ($1,$2,$3,$4,NULLIF($5,0.0),NULLIF($6,0.0))`,
		st.PrvCode, st.CityPrefix, st.SiteNo, st.Name, st.Lat, st.Lng)
	if err != nil {
		return mapErr(fmt.Errorf("odn: create site: %w", err), ErrNotFound)
	}
	return nil
}

// RetireSite 局点退役(须无在用设备挂接)。
func (s *PGStore) RetireSite(ctx context.Context, prvCode, cityPrefix string, siteNo int16) error {
	tag, err := s.db.Exec(ctx, `UPDATE odn_site SET status='RETIRED'
		WHERE prv_code=$1 AND city_prefix=$2 AND site_no=$3 AND status='ACTIVE'
		  AND NOT EXISTS (SELECT 1 FROM odn_device d WHERE d.prv_code=$1
			AND d.city_prefix=$2 AND d.site_no=$3 AND d.status='IN_USE')`,
		prvCode, cityPrefix, siteNo)
	if err != nil {
		return fmt.Errorf("odn: retire site: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// CreateDevice 新建核心链路设备:格式校验(规范 5.1)+ 归属链校验(规范 2.1)。
func (s *PGStore) CreateDevice(ctx context.Context, d Device) error {
	kind, err := ValidateDeviceCode(d.Code)
	if err != nil {
		return err
	}
	if kind != d.Kind {
		return fmt.Errorf("%w: kind %s 与编码前缀不符", ErrInvalidDeviceCode, d.Kind)
	}
	if want := RequiredParentKind(kind); want != "" {
		return s.insertChildDevice(ctx, d, want)
	}
	return s.insertTopDevice(ctx, d)
}

// insertTopDevice 顶层设备(SNW/OLT/ODF/OCC),市域唯一(SNW 全网唯一由部分索引保证)。
func (s *PGStore) insertTopDevice(ctx context.Context, d Device) error {
	var siteNo any
	if d.SiteNo > 0 {
		// E16:site_no 无 FK(复合主键无法单列 FK),域层校验局点已备案且在用。
		var siteOK bool
		err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM odn_site
			WHERE prv_code=$1 AND city_prefix=$2 AND site_no=$3 AND status='ACTIVE')`,
			d.PrvCode, d.CityPrefix, d.SiteNo).Scan(&siteOK)
		if err != nil {
			return fmt.Errorf("odn: check site: %w", err)
		}
		if !siteOK {
			return fmt.Errorf("%w: %s %s/%03d", ErrSiteMissing, d.PrvCode, d.CityPrefix, d.SiteNo)
		}
		siteNo = d.SiteNo
	}
	_, err := s.db.Exec(ctx, `INSERT INTO odn_device (code, kind, prv_code, city_prefix, site_no, name, lat, lng)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		d.Code, d.Kind, d.PrvCode, d.CityPrefix, siteNo, d.Name, d.Lat, d.Lng)
	if err != nil {
		return mapErr(fmt.Errorf("odn: create device: %w", err), ErrNotFound)
	}
	return nil
}

// insertChildDevice 子级设备(ODB/SDB/PRT/TBP):parent 必须是指定类型的在用设备且同城。
func (s *PGStore) insertChildDevice(ctx context.Context, d Device, wantKind string) error {
	var parentOK bool
	err := s.db.QueryRow(ctx, `SELECT EXISTS (SELECT 1 FROM odn_device
		WHERE id=$1 AND kind=$2 AND prv_code=$3 AND city_prefix=$4 AND status='IN_USE')`,
		d.ParentID, wantKind, d.PrvCode, d.CityPrefix).Scan(&parentOK)
	if err != nil {
		return fmt.Errorf("odn: check parent: %w", err)
	}
	if !parentOK {
		return fmt.Errorf("%w: %s 的上级必须是同城市的在用 %s", ErrBadHierarchy, d.Code, wantKind)
	}
	_, err = s.db.Exec(ctx, `INSERT INTO odn_device (code, kind, prv_code, city_prefix, parent_id, name, lat, lng)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8)`,
		d.Code, d.Kind, d.PrvCode, d.CityPrefix, d.ParentID, d.Name, d.Lat, d.Lng)
	if err != nil {
		return mapErr(fmt.Errorf("odn: create device: %w", err), ErrNotFound)
	}
	return nil
}

// ListDevices 设备列表(kind 可空;城市可空=全网)。
func (s *PGStore) ListDevices(ctx context.Context, kind, prvCode, cityPrefix string) ([]Device, error) {
	rows, err := s.db.Query(ctx, `SELECT d.id, d.code, d.kind, COALESCE(d.prv_code,''), COALESCE(d.city_prefix,''),
		COALESCE(d.site_no,0), COALESCE(d.parent_id,0), COALESCE(d.name,''), d.lat, d.lng, d.status, d.lifecycle_status,
		COALESCE(r.registration_no,''), COALESCE(r.asset_id,0), COALESCE(a.asset_code,''), COALESCE(a.status,'')
		FROM odn_device d
		LEFT JOIN odn_asset_registrations r ON r.entity_kind='DEVICE' AND r.device_id=d.id AND r.status='ACTIVE'
		LEFT JOIN assets a ON a.id=r.asset_id
		WHERE ($1='' OR d.kind=$1) AND ($2='' OR (d.prv_code=$2 AND d.city_prefix=$3))
		ORDER BY d.code LIMIT 500`, kind, prvCode, cityPrefix)
	if err != nil {
		return nil, fmt.Errorf("odn: list devices: %w", err)
	}
	defer rows.Close()
	out := []Device{}
	for rows.Next() {
		var d Device
		var regNo, assetCode, assetStatus string
		var regAssetID int64
		if err := rows.Scan(&d.ID, &d.Code, &d.Kind, &d.PrvCode, &d.CityPrefix,
			&d.SiteNo, &d.ParentID, &d.Name, &d.Lat, &d.Lng, &d.Status, &d.LifecycleStatus,
			&regNo, &regAssetID, &assetCode, &assetStatus); err != nil {
			return nil, fmt.Errorf("odn: scan device: %w", err)
		}
		if regNo != "" {
			d.AssetReg = &EntityAssetReg{RegistrationNo: regNo, AssetID: regAssetID, AssetCode: assetCode, AssetStatus: assetStatus}
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// RetireDevice 设备报废(编码永久锁定,资产编码规范红线 2)。
func (s *PGStore) RetireDevice(ctx context.Context, id int64) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE odn_device SET status='RETIRED' WHERE id=$1 AND status='IN_USE'`, id)
	if err != nil {
		return fmt.Errorf("odn: retire device: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
