package geo

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

// PGStore 地理基础数据 PostgreSQL 实现。SQL 与 migrations/000038 一一对应。
type PGStore struct {
	db dbtx
}

// dbtx 最小数据库接口;*pgxpool.Pool 天然满足,单测可注入 mock。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// NewPGStore 构造 PGStore。
func NewPGStore(db *pgxpool.Pool) *PGStore { return &PGStore{db: db} }

// mapWriteErr 写操作错误归一:唯一冲突→ErrDuplicate,其余原样上抛。
func mapWriteErr(err error) error {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return ErrDuplicate
	}
	return err
}

// ListCountries 国家列表;locale 非空时联译名(缺译回落 short_name)。
func (s *PGStore) ListCountries(ctx context.Context, locale string) ([]Country, error) {
	sql := `SELECT c.alpha2, c.alpha3, c.numeric_code, c.short_name,
			COALESCE(c.full_name,''), c.status, c.continent_code,
			COALESCE(c.m49_region,''), COALESCE(c.postal_regex,''), c.is_active`
	args := []any{}
	if locale != "" {
		sql += `, COALESCE((SELECT n.name FROM geo_country_i18n n
			WHERE n.country_code = c.alpha2 AND n.locale = $1 AND n.name_type = 'STANDARD'
			LIMIT 1), c.short_name)`
		args = append(args, locale)
	} else {
		sql += `, c.short_name`
	}
	sql += ` FROM geo_country c ORDER BY c.alpha2`
	rows, err := s.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("geo: list countries: %w", err)
	}
	defer rows.Close()
	out := make([]Country, 0)
	for rows.Next() {
		var c Country
		if err := rows.Scan(&c.Alpha2, &c.Alpha3, &c.NumericCode, &c.ShortName,
			&c.FullName, &c.Status, &c.ContinentCode, &c.M49Region,
			&c.PostalRegex, &c.IsActive, &c.DisplayName); err != nil {
			return nil, fmt.Errorf("geo: scan country: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// GetCountry 国家详情:主档 + 译名 + 关联属性(时区/货币/电话码)。
func (s *PGStore) GetCountry(ctx context.Context, alpha2 string) (*CountryDetail, error) {
	d := &CountryDetail{}
	c := &d.Country
	err := s.db.QueryRow(ctx, `SELECT alpha2, alpha3, numeric_code, short_name,
		COALESCE(full_name,''), status, continent_code,
		COALESCE(m49_region,''), COALESCE(postal_regex,''), is_active, short_name
		FROM geo_country WHERE alpha2 = $1`, alpha2).
		Scan(&c.Alpha2, &c.Alpha3, &c.NumericCode, &c.ShortName,
			&c.FullName, &c.Status, &c.ContinentCode, &c.M49Region,
			&c.PostalRegex, &c.IsActive, &c.DisplayName)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("geo: get country: %w", err)
	}
	if err := s.loadCountryNames(ctx, d); err != nil {
		return nil, err
	}
	if err := s.loadCountryAttrs(ctx, d); err != nil {
		return nil, err
	}
	return d, nil
}

// loadCountryNames 译名列表(按 locale,name_type 排序)。
func (s *PGStore) loadCountryNames(ctx context.Context, d *CountryDetail) error {
	rows, err := s.db.Query(ctx, `SELECT locale, name, name_type
		FROM geo_country_i18n WHERE country_code = $1
		ORDER BY locale, name_type`, d.Alpha2)
	if err != nil {
		return fmt.Errorf("geo: list country names: %w", err)
	}
	defer rows.Close()
	d.Names = make([]CountryName, 0)
	for rows.Next() {
		var n CountryName
		if err := rows.Scan(&n.Locale, &n.Name, &n.NameType); err != nil {
			return fmt.Errorf("geo: scan country name: %w", err)
		}
		d.Names = append(d.Names, n)
	}
	return rows.Err()
}

// loadCountryAttrs 时区/货币/电话码集合。
func (s *PGStore) loadCountryAttrs(ctx context.Context, d *CountryDetail) error {
	tz, err := s.listStrings(ctx,
		`SELECT tz_name FROM country_time_zone WHERE country_code=$1 ORDER BY tz_name`, d.Alpha2)
	if err != nil {
		return err
	}
	cc, err := s.listStrings(ctx,
		`SELECT calling_code FROM country_calling_code WHERE country_code=$1 ORDER BY calling_code`, d.Alpha2)
	if err != nil {
		return err
	}
	rows, err := s.db.Query(ctx, `SELECT currency, is_primary, minor_unit
		FROM country_currency WHERE country_code=$1 ORDER BY is_primary DESC, currency`, d.Alpha2)
	if err != nil {
		return fmt.Errorf("geo: list country currencies: %w", err)
	}
	defer rows.Close()
	d.Attrs = CountryAttrs{TimeZones: tz, CallingCodes: cc, Currencies: make([]Currency, 0)}
	for rows.Next() {
		var cur Currency
		if err := rows.Scan(&cur.Currency, &cur.IsPrimary, &cur.MinorUnit); err != nil {
			return fmt.Errorf("geo: scan currency: %w", err)
		}
		d.Attrs.Currencies = append(d.Attrs.Currencies, cur)
	}
	return rows.Err()
}

// listStrings 单列字符串集合查询。
func (s *PGStore) listStrings(ctx context.Context, sql, arg string) ([]string, error) {
	rows, err := s.db.Query(ctx, sql, arg)
	if err != nil {
		return nil, fmt.Errorf("geo: list strings: %w", err)
	}
	defer rows.Close()
	out := make([]string, 0)
	for rows.Next() {
		var v string
		if err := rows.Scan(&v); err != nil {
			return nil, fmt.Errorf("geo: scan string: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
