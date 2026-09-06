package odn

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// coverageSelect 覆盖关联查询列(地址覆盖关联,000197)。
const coverageSelect = "c.id, c.address_id, COALESCE(c.facility_code,''), " +
	"COALESCE(c.device_id,0), c.status, COALESCE(c.note,''), " +
	"to_char(c.updated_at,'YYYY-MM-DD HH24:MI:SS')"

// UpsertCoverage 保存地址覆盖关联(一址一覆盖,ON CONFLICT 幂等)。
// 失败留 [odn-coverage] 留痕(红线:下游副作用失败禁止静默)。
func (s *PGStore) UpsertCoverage(ctx context.Context, c Coverage) error {
	if err := ValidateCoverage(c.Status, c.FacilityCode, c.DeviceID); err != nil {
		return err
	}
	var facilityArg, deviceArg any
	if c.FacilityCode != "" {
		facilityArg = c.FacilityCode
	}
	if c.DeviceID > 0 {
		deviceArg = c.DeviceID
	}
	const q = `INSERT INTO address_coverage
		(address_id, facility_code, device_id, status, note)
		VALUES ($1,$2,$3,$4,$5)
		ON CONFLICT (address_id) DO UPDATE SET
		facility_code=EXCLUDED.facility_code, device_id=EXCLUDED.device_id,
		status=EXCLUDED.status, note=EXCLUDED.note, updated_at=now()`
	if _, err := s.db.Exec(ctx, q, c.AddressID, facilityArg, deviceArg, c.Status, c.Note); err != nil {
		log.Printf("[odn-coverage] UPSERT FAILED address=%d status=%s: %v",
			c.AddressID, c.Status, err)
		return fmt.Errorf("odn: upsert coverage: %w", mapCoverageErr(err))
	}
	return nil
}

// GetCoverageByAddress 单地址覆盖查询;无记录返回 (nil,nil) 供前端判"未登记"。
func (s *PGStore) GetCoverageByAddress(ctx context.Context, addressID int64) (*Coverage, error) {
	var c Coverage
	q := `SELECT ` + coverageSelect + ` FROM address_coverage c WHERE c.address_id=$1`
	err := s.db.QueryRow(ctx, q, addressID).
		Scan(&c.ID, &c.AddressID, &c.FacilityCode, &c.DeviceID, &c.Status, &c.Note, &c.UpdatedAt)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("odn: get coverage: %w", err)
	}
	return &c, nil
}

// ListCoverage 覆盖关联列表(近更优先,联地址名供 admin 表格)。
func (s *PGStore) ListCoverage(ctx context.Context, limit int) ([]Coverage, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	q := `SELECT ` + coverageSelect + `, COALESCE(a.name,'')
		FROM address_coverage c LEFT JOIN addresses a ON a.id = c.address_id
		ORDER BY c.updated_at DESC LIMIT $1`
	rows, err := s.db.Query(ctx, q, limit)
	if err != nil {
		return nil, fmt.Errorf("odn: list coverage: %w", err)
	}
	defer rows.Close()
	out := []Coverage{}
	for rows.Next() {
		var c Coverage
		if err := rows.Scan(&c.ID, &c.AddressID, &c.FacilityCode, &c.DeviceID,
			&c.Status, &c.Note, &c.UpdatedAt, &c.AddressName); err != nil {
			return nil, fmt.Errorf("odn: scan coverage: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// ResolveLatLng 就近设施命中(平面平方距离排序,球面距离算半径阈值)。
// 无任何在用坐标设施时返回 UNSERVED 空结果,不报错(P1 判据允许无数据)。
func (s *PGStore) ResolveLatLng(ctx context.Context, lat, lng float64) (*CoverageResolved, error) {
	var code, name string
	var flat, flng float64
	q := `SELECT code, COALESCE(name,''), lat, lng FROM odn_facility
		WHERE status='IN_USE' AND lat IS NOT NULL AND lng IS NOT NULL
		ORDER BY (lat-$1)*(lat-$1)+(lng-$2)*(lng-$2)
		LIMIT 1`
	err := s.db.QueryRow(ctx, q, lat, lng).Scan(&code, &name, &flat, &flng)
	if errors.Is(err, pgx.ErrNoRows) {
		return &CoverageResolved{Status: CovUnserved}, nil
	}
	if err != nil {
		return nil, fmt.Errorf("odn: resolve coverage: %w", err)
	}
	dist := haversineM(lat, lng, flat, flng)
	return &CoverageResolved{
		Status:       ResolveStatus(dist),
		FacilityCode: code,
		FacilityName: name,
		Lat:          flat,
		Lng:          flng,
		DistanceM:    dist,
	}, nil
}

// mapCoverageErr 覆盖关联写错误归一:FK 23503 → ErrCoverageTarget(目标不存在)。
func mapCoverageErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23503" {
		return ErrCoverageTarget
	}
	return err
}
