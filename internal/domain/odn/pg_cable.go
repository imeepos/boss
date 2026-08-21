package odn

import (
	"context"
	"fmt"
)

// CreateSegment 新建光缆段落:端点按规范 5.2 定向后入库(同对端点唯一)。
func (s *PGStore) CreateSegment(ctx context.Context, code1, code2, name string) (*Segment, error) {
	a, b, err := OrderEndpoints(code1, code2)
	if err != nil {
		return nil, err
	}
	var seg Segment
	err = s.db.QueryRow(ctx, `INSERT INTO odn_cable_segment (a_code, b_code, name)
		VALUES ($1,$2,NULLIF($3,''))
		ON CONFLICT (a_code, b_code) DO UPDATE SET name = EXCLUDED.name
		RETURNING id, a_code, b_code, COALESCE(name,'')`, a, b, name).
		Scan(&seg.ID, &seg.ACode, &seg.BCode, &seg.Name)
	if err != nil {
		return nil, mapErr(fmt.Errorf("odn: create segment: %w", err), ErrNotFound)
	}
	return &seg, nil
}

// ListSegments 段落列表;endpoint 非空时过滤含该端点的段落。
func (s *PGStore) ListSegments(ctx context.Context, endpoint string) ([]Segment, error) {
	rows, err := s.db.Query(ctx, `SELECT id, a_code, b_code, COALESCE(name,'')
		FROM odn_cable_segment
		WHERE ($1='' OR a_code=$1 OR b_code=$1)
		ORDER BY id LIMIT 500`, endpoint)
	if err != nil {
		return nil, fmt.Errorf("odn: list segments: %w", err)
	}
	defer rows.Close()
	out := []Segment{}
	for rows.Next() {
		var seg Segment
		if err := rows.Scan(&seg.ID, &seg.ACode, &seg.BCode, &seg.Name); err != nil {
			return nil, fmt.Errorf("odn: scan segment: %w", err)
		}
		out = append(out, seg)
	}
	return out, rows.Err()
}

// AddFiber 段落内新增光缆(G01~G99 顺序号,重复→ErrDuplicate)。
func (s *PGStore) AddFiber(ctx context.Context, segID int64, f Fiber) error {
	_, err := s.db.Exec(ctx, `INSERT INTO odn_fiber (segment_id, g_no, kind)
		VALUES ($1,$2,$3)`, segID, f.GNo, f.Kind)
	if err != nil {
		return mapErr(fmt.Errorf("odn: add fiber: %w", err), ErrNotFound)
	}
	return nil
}

// ListFibers 段落光缆列表。
func (s *PGStore) ListFibers(ctx context.Context, segID int64) ([]Fiber, error) {
	rows, err := s.db.Query(ctx, `SELECT segment_id, g_no, COALESCE(kind,'')
		FROM odn_fiber WHERE segment_id=$1 ORDER BY g_no`, segID)
	if err != nil {
		return nil, fmt.Errorf("odn: list fibers: %w", err)
	}
	defer rows.Close()
	out := []Fiber{}
	for rows.Next() {
		var f Fiber
		if err := rows.Scan(&f.SegmentID, &f.GNo, &f.Kind); err != nil {
			return nil, fmt.Errorf("odn: scan fiber: %w", err)
		}
		out = append(out, f)
	}
	return out, rows.Err()
}
