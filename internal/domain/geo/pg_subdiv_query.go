package geo

import (
	"context"
	"fmt"
	"strings"
)

// 区划列表截断口径(选择器懒加载下钻):默认 50,最大 200。
const (
	subdivDefaultLimit = 50
	subdivMaxLimit     = 200
)

// ListSubdivisions 区划列表:国家/locale 过滤 + parentCode 下钻 + keyword 搜索 + limit 截断。
// hasChildren 标记存在未停用子节点,前端据此渲染下钻入口;停用行本身照常返回(兼容旧读法)。
func (s *PGStore) ListSubdivisions(ctx context.Context, f SubdivisionFilter) ([]Subdivision, error) {
	sql, args := buildSubdivisionListSQL(f, clampSubdivLimit(f.Limit))
	rows, err := s.db.Query(ctx, sql, args...)
	if err != nil {
		return nil, fmt.Errorf("geo: list subdivisions: %w", err)
	}
	defer rows.Close()
	out := make([]Subdivision, 0)
	for rows.Next() {
		var d Subdivision
		if err := rows.Scan(&d.Code, &d.CountryCode, &d.ParentCode, &d.Level, &d.Category,
			&d.OSMAdminLevel, &d.GeonameID, &d.IsActive, &d.DisplayName, &d.HasChildren); err != nil {
			return nil, fmt.Errorf("geo: scan subdivision: %w", err)
		}
		out = append(out, d)
	}
	return out, rows.Err()
}

// clampSubdivLimit limit 钳制:<=0 取默认,超上限取上限。
func clampSubdivLimit(n int) int {
	if n <= 0 {
		return subdivDefaultLimit
	}
	if n > subdivMaxLimit {
		return subdivMaxLimit
	}
	return n
}

// buildSubdivisionListSQL 拼装列表查询:占位符按 args 追加顺序生成,返回 SQL 与实参。
func buildSubdivisionListSQL(f SubdivisionFilter, limit int) (string, []any) {
	args := []any{}
	selectSQL := `SELECT d.code, d.country_code, COALESCE(d.parent_code,''), d.level, d.category,
		COALESCE(d.osm_admin_level,0), COALESCE(d.geonameid,0), d.is_active`
	if f.Locale != "" {
		args = append(args, f.Locale)
		selectSQL += fmt.Sprintf(`, COALESCE((SELECT n.name FROM geo_subdivision_i18n n
			WHERE n.subdivision_code = d.code AND n.locale = $%d AND n.name_type = 'STANDARD' LIMIT 1), d.code)`, len(args))
	} else {
		selectSQL += `, d.code`
	}
	selectSQL += `, EXISTS(SELECT 1 FROM geo_subdivision c
		WHERE c.parent_code = d.code AND c.is_active) AS has_children`
	conds := []string{}
	if f.CountryCode != "" {
		args = append(args, f.CountryCode)
		conds = append(conds, fmt.Sprintf("d.country_code = $%d", len(args)))
	}
	if f.ParentCode != nil {
		if *f.ParentCode == "" {
			conds = append(conds, "d.parent_code IS NULL")
		} else {
			args = append(args, *f.ParentCode)
			conds = append(conds, fmt.Sprintf("d.parent_code = $%d", len(args)))
		}
	}
	if f.Keyword != "" {
		args = append(args, "%"+likeEscape(f.Keyword)+"%")
		pat := fmt.Sprintf("$%d", len(args))
		conds = append(conds, fmt.Sprintf(`(d.code ILIKE %[1]s OR EXISTS(SELECT 1 FROM geo_subdivision_i18n k
			WHERE k.subdivision_code = d.code AND k.name ILIKE %[1]s))`, pat))
	}
	sql := selectSQL + " FROM geo_subdivision d"
	if len(conds) > 0 {
		sql += " WHERE " + strings.Join(conds, " AND ")
	}
	args = append(args, limit)
	return sql + fmt.Sprintf(" ORDER BY d.code LIMIT $%d", len(args)), args
}

// likeEscape 转义 LIKE 通配符,防关键字中的 %/_ 被当模式(PG 默认反斜杠转义)。
func likeEscape(kw string) string {
	return strings.NewReplacer(`\`, `\\`, "%", "\\%", "_", "\\_").Replace(kw)
}
