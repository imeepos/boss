package monthly

import (
	"context"
	"fmt"
	"strings"
)

// ImportCSV 上传导入:表头逐字节比对 → 行级校验(白名单/月份/非负)→ 逐行幂等 upsert。
// 行级错误不中断批次(逐行报告行号+原因);派生列不在 CSV 列集,外部无入口。
func (s *PGStore) ImportCSV(ctx context.Context, table string, data []byte) (ImportResult, error) {
	if _, err := specOf(table); err != nil {
		return ImportResult{}, err
	}
	recs, err := parseCSVTable(table, data)
	if err != nil {
		return ImportResult{}, err
	}
	whitelist, err := s.activeRegions(ctx)
	if err != nil {
		return ImportResult{}, err
	}
	res := ImportResult{Total: len(recs)}
	for i, rec := range recs {
		line := i + 2 // 表头占第 1 行
		row, derr := decodeRow(table, rec)
		if derr == nil && !whitelist[row.Region] {
			derr = fmt.Errorf("区域 %q 不在白名单", row.Region)
		}
		if derr == nil {
			derr = s.upsertFact(ctx, table, row)
		}
		if derr != nil {
			res.Failed++
			res.Errors = append(res.Errors, RowError{Line: line, Reason: derr.Error()})
			continue
		}
		res.Imported++
	}
	return res, nil
}

// activeRegions 活跃区域白名单集合(每次导入现查,容忍停用变更)。
func (s *PGStore) activeRegions(ctx context.Context) (map[string]bool, error) {
	rows, err := s.db.Query(ctx, `SELECT region FROM monthly_regions WHERE active`)
	if err != nil {
		return nil, fmt.Errorf("monthly: active regions: %w", err)
	}
	defer rows.Close()
	out := map[string]bool{}
	for rows.Next() {
		var r string
		if err := rows.Scan(&r); err != nil {
			return nil, fmt.Errorf("monthly: scan active region: %w", err)
		}
		out[r] = true
	}
	return out, rows.Err()
}

// ExportCSV 导出:与导入完全同格式同表头(BOM+CRLF),行序 month,region;
// month 为空导出全量。
func (s *PGStore) ExportCSV(ctx context.Context, table, month string) ([]byte, error) {
	spec, err := specOf(table)
	if err != nil {
		return nil, err
	}
	if month != "" {
		if err := ValidateMonth(month); err != nil {
			return nil, err
		}
	}
	cols := append([]string{"month", "region"}, spec.dbCols...)
	q := "SELECT " + strings.Join(cols, ", ") + " FROM " + spec.table + " WHERE ($1 = '' OR month = $1) ORDER BY month, region"
	rows, err := s.db.Query(ctx, q, month)
	if err != nil {
		return nil, fmt.Errorf("monthly: export %s: %w", spec.table, err)
	}
	defer rows.Close()
	var out []csvRow
	for rows.Next() {
		var m, region string
		v := make([]int64, len(spec.dbCols))
		src := make([]any, 0, len(v)+2)
		src = append(src, &m, &region)
		for i := range v {
			src = append(src, &v[i])
		}
		if err := rows.Scan(src...); err != nil {
			return nil, fmt.Errorf("monthly: scan export %s: %w", spec.table, err)
		}
		out = append(out, csvRow{Month: m, Region: region, Values: v})
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("monthly: export %s: %w", spec.table, err)
	}
	return buildCSV(table, out)
}
