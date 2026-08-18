package user

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// geoCountry 根节点行返回国家锚点,非根行锚点由树根继承、强制忽略。
func (r AddressRow) geoCountry(level int8) string {
	if level == 1 {
		return r.CountryCode
	}
	return ""
}

// geoAdmin 同 geoCountry,仅根节点生效。
func (r AddressRow) geoAdmin(level int8) string {
	if level == 1 {
		return r.AdminCode
	}
	return ""
}

// SetAddressGeo 挂接/改挂国家与一级行政区锚点;仅 level=1 根节点允许(迁移 000040 约束兜底)。
func (s *PGStore) SetAddressGeo(ctx context.Context, id int64, countryCode, adminCode string) error {
	var level int8
	err := s.db.QueryRow(ctx, `SELECT level FROM addresses WHERE id = $1`, id).Scan(&level)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("user: set address geo query: %w", err)
	}
	if level != 1 {
		return ErrInvalidInput
	}
	tag, err := s.db.Exec(ctx,
		`UPDATE addresses SET country_code = NULLIF($2,''), admin_code = NULLIF($3,'') WHERE id = $1`,
		id, countryCode, adminCode)
	if err != nil {
		return fmt.Errorf("user: set address geo: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// ListUnlinkedRoots 未挂国家锚点的根节点清单(回填工作台,走 000040 部分索引)。
func (s *PGStore) ListUnlinkedRoots(ctx context.Context) ([]Address, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, 0, level, name, '', ''
		FROM addresses
		WHERE level = 1 AND country_code IS NULL
		ORDER BY path`)
	if err != nil {
		return nil, fmt.Errorf("user: list unlinked roots: %w", err)
	}
	defer rows.Close()
	out := make([]Address, 0)
	for rows.Next() {
		var a Address
		if err := rows.Scan(&a.ID, &a.ParentID, &a.Level, &a.Name, &a.CountryCode, &a.AdminCode); err != nil {
			return nil, fmt.Errorf("user: scan unlinked root: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
