package gis

// ODN 无源物理层点位(ODNPoints)PG 实现:设施/局点/设备自带 lat/lng,直接作地图图层。
// 与 Points(地址/逻辑资源)独立:ODN 实体无 geom 列,bbox 用数值区间过滤(lat/lng BETWEEN)。
// level 语义:9=设施(facility)、10=局点(site)、11=设备(device)、
// 12=勘测打点(survey,W7 000223)、13=施工进度点(progress,W7 000224),与前端图层开关一一对应。

import (
	"context"
	"fmt"
	"strconv"
	"strings"
)

// ODNPoints 按 entity+bbox 返回 ODN 点位(已过滤无坐标行)。
// entity 非法返回 errODNEntityInvalid;bbox 空串=不限。
func (s *PGStore) ODNPoints(ctx context.Context, entity, bbox string) ([]Point, error) {
	sql, args, err := odnPointsQuery(entity, bbox)
	if err != nil {
		return nil, err
	}
	rows, err := s.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("gis: odn points %s: %w", entity, err)
	}
	defer rows.Close()
	out := make([]Point, 0)
	for rows.Next() {
		var p Point
		if err := rows.Scan(&p.ID, &p.Level, &p.Name, &p.Lng, &p.Lat, &p.Status, &p.Count, &p.ParentID); err != nil {
			return nil, fmt.Errorf("gis: scan odn point: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// ErrODNEntityInvalid 非法 ODN 实体类型。
var ErrODNEntityInvalid = fmt.Errorf("gis: entity must be facility|site|device|survey|progress")

// odnPointsQuery 组装 ODN 点位 SQL(按 entity 分发)与参数。
func odnPointsQuery(entity, bbox string) (string, []any, error) {
	bboxSQL, bboxArgs, err := odnBboxToSQL(bbox)
	if err != nil {
		return "", nil, err
	}
	switch entity {
	case "facility":
		return `SELECT row_number() OVER () AS id, 9 AS level, f.code AS name,
				f.lng, f.lat, f.status, 0 AS count, 0 AS parent_id
			FROM odn_facility f
			WHERE f.lat IS NOT NULL AND f.lng IS NOT NULL AND f.status = 'IN_USE'` +
			bboxSQL + ` ORDER BY f.code`, bboxArgs, nil
	case "site":
		return `SELECT row_number() OVER () AS id, 10 AS level,
				s.city_prefix || lpad(s.site_no::text, 3, '0') AS name,
				s.lng, s.lat, s.status, 0 AS count, 0 AS parent_id
			FROM odn_site s
			WHERE s.lat IS NOT NULL AND s.lng IS NOT NULL AND s.status = 'ACTIVE'` +
			bboxSQL + ` ORDER BY s.city_prefix, s.site_no`, bboxArgs, nil
	case "device":
		return `SELECT d.id, 11 AS level, d.code AS name,
				d.lng, d.lat, d.status, 0 AS count, COALESCE(d.site_no, 0) AS parent_id
			FROM odn_device d
			WHERE d.lat IS NOT NULL AND d.lng IS NOT NULL AND d.status = 'IN_USE'` +
			bboxSQL + ` ORDER BY d.code`, bboxArgs, nil
	case "survey":
		return `SELECT r.id, 12 AS level, t.task_no AS name,
				r.lng, r.lat, r.suggestion AS status, 0 AS count, r.task_id AS parent_id
			FROM survey_task_reports r JOIN survey_tasks t ON t.id = r.task_id
			WHERE r.lat IS NOT NULL AND r.lng IS NOT NULL` +
			bboxSQL + ` ORDER BY r.id`, bboxArgs, nil
	case "progress":
		return `SELECT p.id, 13 AS level, p.facility_code AS name,
				p.lng, p.lat, pr.status AS status, 0 AS count, p.project_id AS parent_id
			FROM construction_progress p JOIN construction_projects pr ON pr.id = p.project_id
			WHERE p.lat IS NOT NULL AND p.lng IS NOT NULL` +
			bboxSQL + ` ORDER BY p.id`, bboxArgs, nil
	default:
		return "", nil, ErrODNEntityInvalid
	}
}

// odnBboxToSQL 把 "minLng,minLat,maxLng,maxLat" 解析成 lat/lng 数值区间过滤。
// 空串返回 ("", nil, nil);ODN 表无 geom 列,不能复用 bboxToSQL(ST_Within)。
func odnBboxToSQL(s string) (string, []any, error) {
	s = strings.TrimSpace(s)
	if s == "" {
		return "", nil, nil
	}
	parts := strings.Split(s, ",")
	if len(parts) != 4 {
		return "", nil, fmt.Errorf("gis: bbox 需 4 段 (minLng,minLat,maxLng,maxLat)")
	}
	nums := make([]float64, 4)
	for i, raw := range parts {
		v, err := strconv.ParseFloat(strings.TrimSpace(raw), 64)
		if err != nil {
			return "", nil, fmt.Errorf("gis: bbox 段 %d: %w", i+1, err)
		}
		nums[i] = v
	}
	// 占位符从 $1 起(调用方无前置参数);minLng,minLat,maxLng,maxLat。
	return ` AND lng BETWEEN $1 AND $3 AND lat BETWEEN $2 AND $4`,
		[]any{nums[0], nums[1], nums[2], nums[3]}, nil
}
