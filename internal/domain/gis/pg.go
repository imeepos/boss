package gis

// PGStore GISService 的 PostgreSQL 实现(阶段8,派生聚合:无自有基表,读 addresses/resources/ports/device_metrics/quad_links)。

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// dbtx 最小数据库接口(*pgxpool.Pool / pgxmock 均满足)。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// PGStore GIS 聚合查询实现。
type PGStore struct {
	db dbtx
}

// NewPGStore 构造。
func NewPGStore(db dbtx) *PGStore { return &PGStore{db: db} }

// errLevelInvalid 非法层级(不在 1~8)。
var errLevelInvalid = errors.New("gis: level must be 1..8")

// Drill 八级下钻(见 gis.go 层级口径)。
func (s *PGStore) Drill(ctx context.Context, level int16, parentID int64) ([]Node, error) {
	switch {
	case level >= 1 && level <= 5:
		return s.drillAddresses(ctx, level, parentID)
	case level == 6:
		return s.drillOLT(ctx, parentID)
	case level == 7:
		return s.drillSplitter(ctx, parentID)
	case level == 8:
		return s.drillPorts(ctx, parentID)
	default:
		return nil, errLevelInvalid
	}
}

// drillAddresses 层 1~5:地址层级。
func (s *PGStore) drillAddresses(ctx context.Context, level int16, parentID int64) ([]Node, error) {
	rows, err := s.db.Query(ctx, `
		SELECT a.id, a.name,
		       (SELECT count(*) FROM addresses c WHERE c.parent_id = a.id) AS cnt
		FROM addresses a
		WHERE a.level = $1 AND ($2 = 0 OR a.parent_id = $2)
		ORDER BY a.id`, level, parentID)
	if err != nil {
		return nil, fmt.Errorf("gis: drill addresses: %w", err)
	}
	defer rows.Close()
	return scanNodes(rows, level)
}

// drillOLT 层 6:弱电井(=OLT 设备),parent=楼栋地址 ID。
func (s *PGStore) drillOLT(ctx context.Context, addressID int64) ([]Node, error) {
	rows, err := s.db.Query(ctx, `
		SELECT r.id, r.name,
		       (SELECT count(*) FROM resources c WHERE c.parent_id = r.id) AS cnt
		FROM resources r
		WHERE r.type = 'OLT' AND ($1 = 0 OR r.address_id = $1)
		ORDER BY r.id`, addressID)
	if err != nil {
		return nil, fmt.Errorf("gis: drill olt: %w", err)
	}
	defer rows.Close()
	return scanNodes(rows, 6)
}

// drillSplitter 层 7:分光器(=SPLITTER 设备),parent=OLT 资源 ID。
func (s *PGStore) drillSplitter(ctx context.Context, parentID int64) ([]Node, error) {
	rows, err := s.db.Query(ctx, `
		SELECT r.id, r.name,
		       (SELECT count(*) FROM ports p WHERE p.resource_id = r.id) AS cnt
		FROM resources r
		WHERE r.type = 'SPLITTER' AND r.parent_id = $1
		ORDER BY r.id`, parentID)
	if err != nil {
		return nil, fmt.Errorf("gis: drill splitter: %w", err)
	}
	defer rows.Close()
	return scanNodes(rows, 7)
}

// drillPorts 层 8:端口,parent=分光器资源 ID。
func (s *PGStore) drillPorts(ctx context.Context, resourceID int64) ([]Node, error) {
	rows, err := s.db.Query(ctx, `
		SELECT p.id, p.port_code, 0
		FROM ports p WHERE p.resource_id = $1 ORDER BY p.id`, resourceID)
	if err != nil {
		return nil, fmt.Errorf("gis: drill ports: %w", err)
	}
	defer rows.Close()
	return scanNodes(rows, 8)
}

// LevelCounts 八级数量统计。
func (s *PGStore) LevelCounts(ctx context.Context) ([]Node, error) {
	out := make([]Node, 0, 8)
	for i := int16(1); i <= 8; i++ {
		n, err := s.levelCount(ctx, i)
		if err != nil {
			return nil, err
		}
		out = append(out, n)
	}
	return out, nil
}

// levelCount 单级总量(1~5 地址 / 6 OLT / 7 SPLITTER / 8 端口)。
func (s *PGStore) levelCount(ctx context.Context, level int16) (Node, error) {
	var sql string
	switch {
	case level <= 5:
		sql = `SELECT count(*) FROM addresses WHERE level = $1`
	case level == 6:
		sql = `SELECT count(*) FROM resources WHERE type = 'OLT'`
	case level == 7:
		sql = `SELECT count(*) FROM resources WHERE type = 'SPLITTER'`
	default:
		sql = `SELECT count(*) FROM ports`
	}
	var n int64
	args := []any{}
	if level <= 5 {
		args = append(args, level)
	}
	if err := s.db.QueryRow(ctx, sql, args...).Scan(&n); err != nil {
		return Node{}, fmt.Errorf("gis: level count %d: %w", level, err)
	}
	return Node{Level: level, Count: n}, nil
}

// ResourceDetail 资产实时详情(合并资源 + 最新指标 + 关联用户 + 端口占用)。
func (s *PGStore) ResourceDetail(ctx context.Context, resourceID int64) (*ResourceDetail, error) {
	var d ResourceDetail
	err := s.db.QueryRow(ctx, `
		SELECT id, code, name, type, status, address_id
		FROM resources WHERE id = $1`, resourceID).
		Scan(&d.ID, &d.Code, &d.Name, &d.Type, &d.Status, &d.AddressID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("gis: resource detail: %w", err)
	}
	if err := s.fillMetrics(ctx, &d); err != nil {
		return nil, err
	}
	if err := s.fillUsage(ctx, &d); err != nil {
		return nil, err
	}
	return &d, nil
}

// fillMetrics 最新一条指标(光功率/丢包率/采集时间)。
func (s *PGStore) fillMetrics(ctx context.Context, d *ResourceDetail) error {
	var opt, pkt pgtype.Float8
	var at pgtype.Timestamptz
	err := s.db.QueryRow(ctx, `
		SELECT optical_power, packet_loss, collected_at
		FROM device_metrics WHERE resource_id = $1
		ORDER BY collected_at DESC LIMIT 1`, d.ID).
		Scan(&opt, &pkt, &at)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil // 无指标不视为错误
	}
	if err != nil {
		return fmt.Errorf("gis: metrics: %w", err)
	}
	if opt.Valid {
		v := opt.Float64
		d.OpticalPower = &v
	}
	if pkt.Valid {
		v := pkt.Float64
		d.PacketLoss = &v
	}
	if at.Valid {
		t := at.Time
		d.CollectedAt = &t
	}
	return nil
}

// fillUsage 端口占用 + 关联用户(本资源或子分光器端口 → 四码 → 客户)。
func (s *PGStore) fillUsage(ctx context.Context, d *ResourceDetail) error {
	err := s.db.QueryRow(ctx, `
		SELECT
		  (SELECT count(*) FROM ports p WHERE p.resource_id = r.id OR p.resource_id IN (SELECT id FROM resources WHERE parent_id = r.id)),
		  (SELECT count(*) FROM ports p WHERE (p.resource_id = r.id OR p.resource_id IN (SELECT id FROM resources WHERE parent_id = r.id)) AND p.status != 'IDLE'),
		  (SELECT COALESCE(c.name,'') FROM ports p
		     JOIN quad_links q ON q.port_id = p.id
		     JOIN customers c ON c.id = q.customer_id
		     WHERE p.resource_id = r.id OR p.resource_id IN (SELECT id FROM resources WHERE parent_id = r.id)
		     LIMIT 1)
		FROM resources r WHERE r.id = $1`, d.ID).
		Scan(&d.PortsTotal, &d.PortsUsed, &d.CustomerName)
	if err != nil {
		return fmt.Errorf("gis: usage: %w", err)
	}
	return nil
}

// ErrNotFound 资源不存在。
var ErrNotFound = errors.New("gis: resource not found")

// scanNodes 逐行扫描为 Node。
func scanNodes(rows pgx.Rows, level int16) ([]Node, error) {
	out := make([]Node, 0)
	for rows.Next() {
		var n Node
		if err := rows.Scan(&n.ID, &n.Name, &n.Count); err != nil {
			return nil, fmt.Errorf("gis: scan node: %w", err)
		}
		n.Level = level
		out = append(out, n)
	}
	return out, rows.Err()
}
