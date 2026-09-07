package odn

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PGStore odn 域 PostgreSQL 存储(网格分区部分)。
type PGStore struct {
	db                 *pgxpool.Pool
	permitGate         bool // F3 开工许可门控灰度(BOSS_ODN_PERMIT_GATE=on,默认关)
	acceptCoverageLink bool // F6 竣工覆盖联动开关(BOSS_ODN_ACCEPT_COVERAGE_LINK=off 可关,默认开)
}

// NewPGStore 构造 PGStore(竣工覆盖联动默认开,门控默认关)。
func NewPGStore(db *pgxpool.Pool) *PGStore {
	return &PGStore{db: db, acceptCoverageLink: true}
}

// SetPermitGate 开工许可门控开关(F3,装配层注入)。
func (s *PGStore) SetPermitGate(on bool) { s.permitGate = on }

// SetAcceptCoverageLink 竣工覆盖联动开关(F6,装配层注入)。
func (s *PGStore) SetAcceptCoverageLink(on bool) { s.acceptCoverageLink = on }

// mapErr 写操作错误归一:唯一/PK 冲突→ErrDuplicate,零行→ErrNotFound。
func mapErr(err error, notFound error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return notFound
	}
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrDuplicate
	}
	return err
}

// ListGrids 网格列表(含设施占用数与 >=80% 预警,规范 4.7)。
func (s *PGStore) ListGrids(ctx context.Context, prvCode, cityPrefix string) ([]GridUsage, error) {
	rows, err := s.db.Query(ctx, `SELECT g.prv_code, g.city_prefix, g.grid_code,
			COALESCE(g.name,''), COALESCE(g.coverage,''), g.status,
			(SELECT count(*) FROM odn_facility f WHERE f.prv_code=g.prv_code
			 AND f.city_prefix=g.city_prefix AND f.grid_code=g.grid_code)
		FROM odn_grid g
		WHERE g.prv_code=$1 AND g.city_prefix=$2 AND g.status <> 'RETIRED'
		ORDER BY g.grid_code`, prvCode, cityPrefix)
	if err != nil {
		return nil, fmt.Errorf("odn: list grids: %w", err)
	}
	defer rows.Close()
	out := []GridUsage{}
	for rows.Next() {
		var g GridUsage
		if err := rows.Scan(&g.PrvCode, &g.CityPrefix, &g.GridCode,
			&g.Name, &g.Coverage, &g.Status, &g.Facilities); err != nil {
			return nil, fmt.Errorf("odn: scan grid: %w", err)
		}
		g.Warn = g.Facilities >= GridWarnUsage
		out = append(out, g)
	}
	return out, rows.Err()
}

// CreateGrid 新建网格(城市前缀须已登记,唯一冲突→ErrDuplicate)。
func (s *PGStore) CreateGrid(ctx context.Context, g Grid) error {
	_, err := s.db.Exec(ctx, `INSERT INTO odn_grid (prv_code, city_prefix, grid_code, name, coverage, status)
		VALUES ($1,$2,$3,$4,$5,$6)`,
		g.PrvCode, g.CityPrefix, g.GridCode, g.Name, g.Coverage, g.Status)
	if err != nil {
		return mapErr(fmt.Errorf("odn: create grid: %w", err), ErrNotFound)
	}
	return nil
}

// UpdateGrid 编辑网格(网格码不可改;RETIERED 不可改回)。
func (s *PGStore) UpdateGrid(ctx context.Context, prvCode, cityPrefix string, gridCode int16, g Grid) error {
	tag, err := s.db.Exec(ctx, `UPDATE odn_grid SET name=$4, coverage=$5,
			status=(CASE WHEN $6 = 'RETIRED' THEN 'RETIRED' ELSE $6 END), updated_at=now()
		WHERE prv_code=$1 AND city_prefix=$2 AND grid_code=$3 AND status <> 'RETIRED'`,
		prvCode, cityPrefix, gridCode, g.Name, g.Coverage, g.Status)
	if err != nil {
		return fmt.Errorf("odn: update grid: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// RetireGrid 网格退役(须无在用设施,否则 ErrDuplicate 语义冲突)。
func (s *PGStore) RetireGrid(ctx context.Context, prvCode, cityPrefix string, gridCode int16) error {
	tag, err := s.db.Exec(ctx, `UPDATE odn_grid SET status='RETIRED', updated_at=now()
		WHERE prv_code=$1 AND city_prefix=$2 AND grid_code=$3
		  AND NOT EXISTS (SELECT 1 FROM odn_facility f
			WHERE f.prv_code=$1 AND f.city_prefix=$2 AND f.grid_code=$3 AND f.status='IN_USE')`,
		prvCode, cityPrefix, gridCode)
	if err != nil {
		return fmt.Errorf("odn: retire grid: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
