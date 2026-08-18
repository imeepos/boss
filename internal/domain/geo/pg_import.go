package geo

import (
	"context"
	"fmt"
)

// Import 批量导入:单事务内逐条 upsert(主键/唯一键冲突则更新,不删既有数据)。
// 顺序:国家 → 国家译名 → 区划 → 区划译名,保证外键先就位。
func (s *PGStore) Import(ctx context.Context, data ImportData) (ImportCounts, error) {
	var n ImportCounts
	err := s.withTx(ctx, func(tx dbtx) error {
		for _, c := range data.Countries {
			if err := upsertCountry(ctx, tx, c); err != nil {
				return err
			}
			n.Countries++
		}
		for _, r := range data.CountryNames {
			if err := upsertCountryName(ctx, tx, r); err != nil {
				return err
			}
			n.CountryNames++
		}
		for _, d := range data.Subdivisions {
			if err := upsertSubdiv(ctx, tx, d); err != nil {
				return err
			}
			n.Subdivisions++
		}
		for _, r := range data.SubdivisionNames {
			if err := upsertSubdivName(ctx, tx, r); err != nil {
				return err
			}
			n.SubdivisionNames++
		}
		return nil
	})
	if err != nil {
		return ImportCounts{}, err
	}
	return n, nil
}

// upsertCountry 国家 upsert(is_active 不覆盖,停用状态由页面维护)。
func upsertCountry(ctx context.Context, tx dbtx, c Country) error {
	_, err := tx.Exec(ctx, `INSERT INTO geo_country
		(alpha2, alpha3, numeric_code, short_name, full_name, status, continent_code, m49_region, postal_regex)
		VALUES ($1,$2,$3,$4,NULLIF($5,''),$6,$7,NULLIF($8,''),NULLIF($9,''))
		ON CONFLICT (alpha2) DO UPDATE SET
			alpha3=EXCLUDED.alpha3, numeric_code=EXCLUDED.numeric_code,
			short_name=EXCLUDED.short_name, full_name=EXCLUDED.full_name,
			status=EXCLUDED.status, continent_code=EXCLUDED.continent_code,
			m49_region=EXCLUDED.m49_region, postal_regex=EXCLUDED.postal_regex`,
		c.Alpha2, c.Alpha3, c.NumericCode, c.ShortName, c.FullName,
		c.Status, c.ContinentCode, c.M49Region, c.PostalRegex)
	if err != nil {
		return mapWriteErr(fmt.Errorf("geo: import country %s: %w", c.Alpha2, err))
	}
	return nil
}

// upsertCountryName 国家译名 upsert(唯一键 (country_code, locale, name_type))。
func upsertCountryName(ctx context.Context, tx dbtx, r CountryNameRow) error {
	_, err := tx.Exec(ctx, `INSERT INTO geo_country_i18n (country_code, locale, name, name_type)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (country_code, locale, name_type) DO UPDATE SET name=EXCLUDED.name`,
		r.CountryCode, r.Name.Locale, r.Name.Name, r.Name.NameType)
	if err != nil {
		return mapWriteErr(fmt.Errorf("geo: import country name %s/%s: %w", r.CountryCode, r.Name.Locale, err))
	}
	return nil
}

// upsertSubdiv 区划 upsert(is_active 不覆盖)。
func upsertSubdiv(ctx context.Context, tx dbtx, d Subdivision) error {
	_, err := tx.Exec(ctx, `INSERT INTO geo_subdivision
		(code, country_code, parent_code, level, category, osm_admin_level, geonameid)
		VALUES ($1,$2,NULLIF($3,''),$4,$5,$6,NULLIF($7,0))
		ON CONFLICT (code) DO UPDATE SET
			country_code=EXCLUDED.country_code, parent_code=EXCLUDED.parent_code,
			level=EXCLUDED.level, category=EXCLUDED.category,
			osm_admin_level=EXCLUDED.osm_admin_level, geonameid=EXCLUDED.geonameid`,
		d.Code, d.CountryCode, d.ParentCode, d.Level, d.Category, d.OSMAdminLevel, d.GeonameID)
	if err != nil {
		return mapWriteErr(fmt.Errorf("geo: import subdivision %s: %w", d.Code, err))
	}
	return nil
}

// upsertSubdivName 区划译名 upsert(唯一键 (subdivision_code, locale, name_type))。
func upsertSubdivName(ctx context.Context, tx dbtx, r SubdivisionNameRow) error {
	_, err := tx.Exec(ctx, `INSERT INTO geo_subdivision_i18n (subdivision_code, locale, name, name_type)
		VALUES ($1,$2,$3,$4)
		ON CONFLICT (subdivision_code, locale, name_type) DO UPDATE SET name=EXCLUDED.name`,
		r.SubdivisionCode, r.Name.Locale, r.Name.Name, r.Name.NameType)
	if err != nil {
		return mapWriteErr(fmt.Errorf("geo: import subdivision name %s/%s: %w", r.SubdivisionCode, r.Name.Locale, err))
	}
	return nil
}
