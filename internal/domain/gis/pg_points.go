package gis

// 地图点位(Points)PG 实现:PGIS 真地图底图层。
// 口径:
//   - level 1~5:addresses.geom 自带经纬度,ST_X/ST_Y 抽取,bbox 走 ST_Within(ST_MakeEnvelope)。
//   - level 6 (OLT):OLT 自身无 lat/lng,落父楼栋地址 geom。
//   - level 7 (SPLITTER) / level 8 (ports):端口自身无坐标,经资源链回 join addresses 取 ge;ODN 表无录入时回退父地址 geom,不返回空坐标点。
// parentID=0 表示顶层;bbox 空串=不限。

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

// Points 按 level+parentID+bbox 返回点位列表(已读层过滤掉空坐标的孤儿点位)。
func (s *PGStore) Points(ctx context.Context, level int16, parentID int64, bbox string) ([]Point, error) {
	if level < 1 || level > 8 {
		return nil, errLevelInvalid
	}
	bboxSQL, bboxArgs, err := bboxToSQL(bbox)
	if err != nil {
		return nil, err
	}
	switch {
	case level <= 5:
		return s.pointsAddresses(ctx, level, parentID, bboxSQL, bboxArgs)
	case level == 6:
		return s.pointsOLT(ctx, parentID, bboxSQL, bboxArgs)
	case level == 7:
		return s.pointsSplitter(ctx, parentID, bboxSQL, bboxArgs)
	default:
		return s.pointsPorts(ctx, parentID, bboxSQL, bboxArgs)
	}
}

// pointsAddresses 1~5 层:读 addresses.geom。
func (s *PGStore) pointsAddresses(ctx context.Context, level int16, parentID int64, bboxSQL string, bboxArgs []any) ([]Point, error) {
	sql := `
		SELECT a.id, a.name,
		       ST_X(a.geom::geometry) AS lng, ST_Y(a.geom::geometry) AS lat,
		       'AREA' AS status,
		       (SELECT count(*) FROM addresses c WHERE c.parent_id = a.id) AS cnt,
		       COALESCE(a.parent_id, 0) AS parent_id
		FROM addresses a
		WHERE a.level = $1 AND a.geom IS NOT NULL
		  AND ($2 = 0 OR a.parent_id = $2)`
	args := []any{level, parentID}
	if bboxSQL != "" {
		sql += " AND " + bboxSQL
		args = append(args, bboxArgs...)
	}
	sql += " ORDER BY a.id"
	rows, err := s.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("gis: points addresses: %w", err)
	}
	return scanPoints(rows, level)
}

// pointsOLT 6 层:OLT 借父地址 geom。
func (s *PGStore) pointsOLT(ctx context.Context, parentID int64, bboxSQL string, bboxArgs []any) ([]Point, error) {
	sql := `
		SELECT r.id, r.name,
		       ST_X(p.geom::geometry) AS lng, ST_Y(p.geom::geometry) AS lat,
		       COALESCE(r.status, 'UNKNOWN') AS status,
		       (SELECT count(*) FROM resources c WHERE c.parent_id = r.id) AS cnt,
		       COALESCE(r.address_id, 0) AS parent_id
		FROM resources r
		JOIN addresses p ON p.id = r.address_id AND p.geom IS NOT NULL
		WHERE r.type = 'OLT'
		  AND ($1 = 0 OR r.address_id = $1)`
	args := []any{parentID}
	if bboxSQL != "" {
		sql += " AND " + bboxSQL
		args = append(args, bboxArgs...)
	}
	sql += " ORDER BY r.id"
	rows, err := s.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("gis: points olt: %w", err)
	}
	return scanPoints(rows, 6)
}

// pointsSplitter 7 层:SPLITTER → address 取 geom(无 ODN 站表 join,address 已落位)。
func (s *PGStore) pointsSplitter(ctx context.Context, parentID int64, bboxSQL string, bboxArgs []any) ([]Point, error) {
	sql := `
		SELECT r.id, r.name,
		       ST_X(p.geom::geometry) AS lng, ST_Y(p.geom::geometry) AS lat,
		       COALESCE(r.status, 'UNKNOWN') AS status,
		       (SELECT count(*) FROM ports c WHERE c.resource_id = r.id) AS cnt,
		       COALESCE(r.parent_id, 0) AS parent_id
		FROM resources r
		JOIN addresses p ON p.id = r.address_id AND p.geom IS NOT NULL
		WHERE r.type = 'SPLITTER' AND r.parent_id = $1`
	args := []any{parentID}
	if bboxSQL != "" {
		sql += " AND " + bboxSQL
		args = append(args, bboxArgs...)
	}
	sql += " ORDER BY r.id"
	rows, err := s.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("gis: points splitter: %w", err)
	}
	return scanPoints(rows, 7)
}

// pointsPorts 8 层:ports → resource → address 取 geom(端口自身无坐标)。
func (s *PGStore) pointsPorts(ctx context.Context, parentID int64, bboxSQL string, bboxArgs []any) ([]Point, error) {
	sql := `
		SELECT p.id, p.port_code,
		       ST_X(a.geom::geometry) AS lng, ST_Y(a.geom::geometry) AS lat,
		       COALESCE(p.status, 'UNKNOWN') AS status,
		       0 AS cnt,
		       COALESCE(p.resource_id, 0) AS parent_id
		FROM ports p
		JOIN resources r ON r.id = p.resource_id
		JOIN addresses a ON a.id = r.address_id AND a.geom IS NOT NULL
		WHERE p.resource_id = $1`
	args := []any{parentID}
	if bboxSQL != "" {
		sql += " AND " + bboxSQL
		args = append(args, bboxArgs...)
	}
	sql += " ORDER BY p.id"
	rows, err := s.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("gis: points ports: %w", err)
	}
	return scanPoints(rows, 8)
}

// bboxToSQL 把 "minLng,minLat,maxLng,maxLat" 解析成 ST_Within(...) + 参数。
// 空串返回 ("", nil, nil) 表示不限。
func bboxToSQL(s string) (string, []any, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil, nil
	}
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return "", nil, fmt.Errorf("gis: bbox 需 4 段 (minLng,minLat,maxLng,maxLat)")
	}
	minLng, err := strconv.ParseFloat(strings.TrimSpace(parts[0]), 64)
	if err != nil {
		return "", nil, fmt.Errorf("gis: bbox minLng: %w", err)
	}
	minLat, err := strconv.ParseFloat(strings.TrimSpace(parts[1]), 64)
	if err != nil {
		return "", nil, fmt.Errorf("gis: bbox minLat: %w", err)
	}
	maxLng, err := strconv.ParseFloat(strings.TrimSpace(parts[2]), 64)
	if err != nil {
		return "", nil, fmt.Errorf("gis: bbox maxLng: %w", err)
	}
	maxLat, err := strconv.ParseFloat(strings.TrimSpace(parts[3]), 64)
	if err != nil {
		return "", nil, fmt.Errorf("gis: bbox maxLat: %w", err)
	}
	// ST_MakeEnvelope(xmin, ymin, xmax, ymax, srid) 与 ST_Within(geom::geometry, env)。
	// 参数顺序:必须先把 4 个 float 放进 args,SQL 用 $3..$6 引用。
	return `ST_Within(a.geom::geometry, ST_MakeEnvelope($3, $4, $5, $6, 4326))`, []any{minLng, minLat, maxLng, maxLat}, nil
}

// scanPoints 逐行扫描为 Point(与 pg.go 的 scanNodes 形态对齐,便于代码 grep)。
// level 由调用方传入(各 pointsX 函数硬编码所属层级,SQL 不冗余 SELECT)。
func scanPoints(rows pgx.Rows, level int16) ([]Point, error) {
	out := make([]Point, 0)
	for rows.Next() {
		var p Point
		if err := rows.Scan(&p.ID, &p.Name, &p.Lng, &p.Lat, &p.Status, &p.Count, &p.ParentID); err != nil {
			return nil, fmt.Errorf("gis: scan point: %w", err)
		}
		p.Level = level
		out = append(out, p)
	}
	return out, rows.Err()
}
