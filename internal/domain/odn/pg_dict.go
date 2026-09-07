package odn

import (
	"context"
	"fmt"
)

// ListRegions 省级编码字典(000075 全量,只读)。
func (s *PGStore) ListRegions(ctx context.Context) ([]RegionOption, error) {
	rows, err := s.db.Query(ctx,
		`SELECT prv_code, spec_name FROM odn_region_code ORDER BY prv_code`)
	if err != nil {
		return nil, fmt.Errorf("odn: list regions: %w", err)
	}
	defer rows.Close()
	out := []RegionOption{}
	for rows.Next() {
		var r RegionOption
		if err := rows.Scan(&r.PrvCode, &r.Name); err != nil {
			return nil, fmt.Errorf("odn: scan region: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// ListCities 指定省的城市前缀字典。
func (s *PGStore) ListCities(ctx context.Context, prvCode string) ([]CityOption, error) {
	rows, err := s.db.Query(ctx,
		`SELECT city_prefix, spec_name FROM odn_city_code WHERE prv_code=$1 ORDER BY city_prefix`,
		prvCode)
	if err != nil {
		return nil, fmt.Errorf("odn: list cities %s: %w", prvCode, err)
	}
	defer rows.Close()
	out := []CityOption{}
	for rows.Next() {
		var ct CityOption
		if err := rows.Scan(&ct.CityPrefix, &ct.Name); err != nil {
			return nil, fmt.Errorf("odn: scan city: %w", err)
		}
		out = append(out, ct)
	}
	return out, rows.Err()
}
