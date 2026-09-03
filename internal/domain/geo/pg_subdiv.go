package geo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// CreateCountry 新建国家(唯一冲突→ErrDuplicate)。
func (s *PGStore) CreateCountry(ctx context.Context, c Country) error {
	_, err := s.db.Exec(ctx, `INSERT INTO geo_country
		(alpha2, alpha3, numeric_code, short_name, full_name, status, continent_code, m49_region, postal_regex)
		VALUES ($1,$2,$3,$4,NULLIF($5,''),$6,$7,NULLIF($8,''),NULLIF($9,''))`,
		c.Alpha2, c.Alpha3, c.NumericCode, c.ShortName, c.FullName,
		c.Status, c.ContinentCode, c.M49Region, c.PostalRegex)
	if err != nil {
		return mapWriteErr(fmt.Errorf("geo: create country: %w", err))
	}
	return nil
}

// UpdateCountry 编辑国家主档(alpha2 为主键不可改)。
func (s *PGStore) UpdateCountry(ctx context.Context, alpha2 string, c Country) error {
	tag, err := s.db.Exec(ctx, `UPDATE geo_country SET
		alpha3=$2, numeric_code=$3, short_name=$4, full_name=NULLIF($5,''),
		status=$6, continent_code=$7, m49_region=NULLIF($8,''), postal_regex=NULLIF($9,'')
		WHERE alpha2=$1`,
		alpha2, c.Alpha3, c.NumericCode, c.ShortName, c.FullName,
		c.Status, c.ContinentCode, c.M49Region, c.PostalRegex)
	if err != nil {
		return mapWriteErr(fmt.Errorf("geo: update country: %w", err))
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetCountryActive 停用/启用国家(软删除保留)。
func (s *PGStore) SetCountryActive(ctx context.Context, alpha2 string, active bool) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE geo_country SET is_active=$2 WHERE alpha2=$1`, alpha2, active)
	if err != nil {
		return fmt.Errorf("geo: set country active: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// AddCountryName 新增国家译名;RemoveCountryName 删除。
func (s *PGStore) AddCountryName(ctx context.Context, alpha2 string, n CountryName) error {
	_, err := s.db.Exec(ctx, `INSERT INTO geo_country_i18n (country_code, locale, name, name_type)
		VALUES ($1,$2,$3,$4)`, alpha2, n.Locale, n.Name, n.NameType)
	if err != nil {
		return mapWriteErr(fmt.Errorf("geo: add country name: %w", err))
	}
	return nil
}

func (s *PGStore) RemoveCountryName(ctx context.Context, alpha2, locale, nameType string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM geo_country_i18n
		WHERE country_code=$1 AND locale=$2 AND name_type=$3`, alpha2, locale, nameType)
	if err != nil {
		return fmt.Errorf("geo: remove country name: %w", err)
	}
	return nil
}

// ReplaceCountryAttrs 整体替换时区/货币/电话码集合(单事务,先删后插)。
func (s *PGStore) ReplaceCountryAttrs(ctx context.Context, alpha2 string, attrs CountryAttrs) error {
	return s.withTx(ctx, func(tx dbtx) error {
		if _, err := tx.Exec(ctx, `DELETE FROM country_time_zone WHERE country_code=$1`, alpha2); err != nil {
			return fmt.Errorf("geo: clear time zones: %w", err)
		}
		for _, tz := range attrs.TimeZones {
			if _, err := tx.Exec(ctx,
				`INSERT INTO country_time_zone (country_code, tz_name) VALUES ($1,$2)`, alpha2, tz); err != nil {
				return mapWriteErr(fmt.Errorf("geo: insert time zone: %w", err))
			}
		}
		if err := replaceCurrencies(ctx, tx, alpha2, attrs.Currencies); err != nil {
			return err
		}
		if _, err := tx.Exec(ctx, `DELETE FROM country_calling_code WHERE country_code=$1`, alpha2); err != nil {
			return fmt.Errorf("geo: clear calling codes: %w", err)
		}
		for _, cc := range attrs.CallingCodes {
			if _, err := tx.Exec(ctx,
				`INSERT INTO country_calling_code (country_code, calling_code) VALUES ($1,$2)`, alpha2, cc); err != nil {
				return mapWriteErr(fmt.Errorf("geo: insert calling code: %w", err))
			}
		}
		return nil
	})
}

// replaceCurrencies 替换货币集合。
func replaceCurrencies(ctx context.Context, tx dbtx, alpha2 string, list []Currency) error {
	if _, err := tx.Exec(ctx, `DELETE FROM country_currency WHERE country_code=$1`, alpha2); err != nil {
		return fmt.Errorf("geo: clear currencies: %w", err)
	}
	for _, cur := range list {
		if _, err := tx.Exec(ctx, `INSERT INTO country_currency
			(country_code, currency, is_primary, minor_unit) VALUES ($1,$2,$3,$4)`,
			alpha2, cur.Currency, cur.IsPrimary, cur.MinorUnit); err != nil {
			return mapWriteErr(fmt.Errorf("geo: insert currency: %w", err))
		}
	}
	return nil
}

// withTx 包一层匿名事务(接口层无嵌套事务需求,保持简单)。
func (s *PGStore) withTx(ctx context.Context, fn func(tx dbtx) error) error {
	txer, ok := s.db.(interface {
		Begin(ctx context.Context) (pgx.Tx, error)
	})
	if !ok {
		return fn(s.db) // mock/测试通道:退化为直连执行
	}
	tx, err := txer.Begin(ctx)
	if err != nil {
		return fmt.Errorf("geo: begin tx: %w", err)
	}
	if err := fn(tx); err != nil {
		_ = tx.Rollback(ctx)
		return err
	}
	return tx.Commit(ctx)
}

// GetSubdivision 单查区划(挂接校验用)。
func (s *PGStore) GetSubdivision(ctx context.Context, code string) (*Subdivision, error) {
	var d Subdivision
	err := s.db.QueryRow(ctx, `SELECT code, country_code, COALESCE(parent_code,''),
		level, category, COALESCE(osm_admin_level,0), COALESCE(geonameid,0), is_active, code
		FROM geo_subdivision WHERE code = $1`, code).
		Scan(&d.Code, &d.CountryCode, &d.ParentCode, &d.Level, &d.Category,
			&d.OSMAdminLevel, &d.GeonameID, &d.IsActive, &d.DisplayName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("geo: get subdivision: %w", err)
	}
	return &d, nil
}

// ListSubdivisionNames 区划译名列表(按 locale,name_type 排序)。
func (s *PGStore) ListSubdivisionNames(ctx context.Context, code string) ([]SubdivisionName, error) {
	rows, err := s.db.Query(ctx, `SELECT locale, name, name_type
		FROM geo_subdivision_i18n WHERE subdivision_code = $1
		ORDER BY locale, name_type`, code)
	if err != nil {
		return nil, fmt.Errorf("geo: list subdivision names: %w", err)
	}
	defer rows.Close()
	out := make([]SubdivisionName, 0)
	for rows.Next() {
		var n SubdivisionName
		if err := rows.Scan(&n.Locale, &n.Name, &n.NameType); err != nil {
			return nil, fmt.Errorf("geo: scan subdivision name: %w", err)
		}
		out = append(out, n)
	}
	return out, rows.Err()
}

// CreateSubdivision 新建区划。
func (s *PGStore) CreateSubdivision(ctx context.Context, d Subdivision) error {
	_, err := s.db.Exec(ctx, `INSERT INTO geo_subdivision
		(code, country_code, parent_code, level, category, osm_admin_level, geonameid)
		VALUES ($1,$2,NULLIF($3,''),$4,$5,$6,NULLIF($7,0))`,
		d.Code, d.CountryCode, d.ParentCode, d.Level, d.Category, d.OSMAdminLevel, d.GeonameID)
	if err != nil {
		return mapWriteErr(fmt.Errorf("geo: create subdivision: %w", err))
	}
	return nil
}

// UpdateSubdivision 编辑区划(code 为主键不可改)。
func (s *PGStore) UpdateSubdivision(ctx context.Context, code string, d Subdivision) error {
	tag, err := s.db.Exec(ctx, `UPDATE geo_subdivision SET
		country_code=$2, parent_code=NULLIF($3,''), level=$4, category=$5,
		osm_admin_level=$6, geonameid=NULLIF($7,0) WHERE code=$1`,
		code, d.CountryCode, d.ParentCode, d.Level, d.Category, d.OSMAdminLevel, d.GeonameID)
	if err != nil {
		return mapWriteErr(fmt.Errorf("geo: update subdivision: %w", err))
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// SetSubdivisionActive 停用/启用区划。
func (s *PGStore) SetSubdivisionActive(ctx context.Context, code string, active bool) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE geo_subdivision SET is_active=$2 WHERE code=$1`, code, active)
	if err != nil {
		return fmt.Errorf("geo: set subdivision active: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// AddSubdivisionName / RemoveSubdivisionName 区划译名维护。
func (s *PGStore) AddSubdivisionName(ctx context.Context, code string, n SubdivisionName) error {
	_, err := s.db.Exec(ctx, `INSERT INTO geo_subdivision_i18n (subdivision_code, locale, name, name_type)
		VALUES ($1,$2,$3,$4)`, code, n.Locale, n.Name, n.NameType)
	if err != nil {
		return mapWriteErr(fmt.Errorf("geo: add subdivision name: %w", err))
	}
	return nil
}

func (s *PGStore) RemoveSubdivisionName(ctx context.Context, code, locale, nameType string) error {
	_, err := s.db.Exec(ctx, `DELETE FROM geo_subdivision_i18n
		WHERE subdivision_code=$1 AND locale=$2 AND name_type=$3`, code, locale, nameType)
	if err != nil {
		return fmt.Errorf("geo: remove subdivision name: %w", err)
	}
	return nil
}
