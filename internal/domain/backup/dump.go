// 备份执行器:逐表流式导出为 gzip JSONL。
// 归档格式 v1:首行 {"v":1};每表一行头 {"t":表名,"c":[列名],"u":[udt 名]},
// 之后每行 {"r":[行值]}(encoding/json 序列化,时间戳 RFC3339,bytea base64)。
package backup

import (
	"compress/gzip"
	"context"
	"encoding/json"
	"fmt"
	"os"
	"time"
)

type dumpMeta struct {
	V int `json:"v"`
}

type tableHeader struct {
	T string   `json:"t"`
	C []string `json:"c"`
	U []string `json:"u"` // pg udt 名:int8/text/timestamptz/bytea/_text...
}

type rowLine struct {
	R []any `json:"r"`
}

// tableColumns 列名 + udt 名(按 ordinal 顺序)。
func (s *Service) tableColumns(ctx context.Context, table string) ([]string, []string, error) {
	rows, err := s.store.db.Query(ctx, `
		SELECT column_name, udt_name FROM information_schema.columns
		WHERE table_schema = 'public' AND table_name = $1
		ORDER BY ordinal_position`, table)
	if err != nil {
		return nil, nil, fmt.Errorf("backup: query columns %s: %w", table, err)
	}
	defer rows.Close()
	cols, udts := make([]string, 0, 16), make([]string, 0, 16)
	for rows.Next() {
		var c, u string
		if err := rows.Scan(&c, &u); err != nil {
			return nil, nil, fmt.Errorf("backup: scan column %s: %w", table, err)
		}
		cols, udts = append(cols, c), append(udts, u)
	}
	if len(cols) == 0 {
		return nil, nil, fmt.Errorf("backup: table %s not found", table)
	}
	return cols, udts, rows.Err()
}

// runBackup 备份主流程( goroutine 内执行):导出 → 收尾簿记。
func (s *Service) runBackup(jobID int64, tables []string) {
	defer s.unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	fileName := fmt.Sprintf("backup-%d-%s.jsonl.gz", jobID, time.Now().UTC().Format("20060102T150405Z"))
	path := s.jobFilePath(fileName)
	res, err := s.dump(ctx, path, tables)
	if err != nil {
		s.store.FinishJob(ctx, jobID, StatusFailed, fileName, res.sizeBytes, res.tableCount, res.rowCount, err.Error())
		os.Remove(path)
		return
	}
	s.store.FinishJob(ctx, jobID, StatusSucceeded, fileName, res.sizeBytes, res.tableCount, res.rowCount, "")
}

type dumpResult struct {
	sizeBytes  int64
	tableCount int
	rowCount   int64
}

// dump 逐表流式写出 gzip JSONL。
func (s *Service) dump(ctx context.Context, path string, tables []string) (dumpResult, error) {
	f, err := os.Create(path)
	if err != nil {
		return dumpResult{}, fmt.Errorf("backup: create file: %w", err)
	}
	defer f.Close()
	gz := gzip.NewWriter(f)
	enc := json.NewEncoder(gz)
	if err := enc.Encode(dumpMeta{V: 1}); err != nil {
		return dumpResult{}, fmt.Errorf("backup: write meta: %w", err)
	}
	var res dumpResult
	for _, t := range tables {
		cols, udts, err := s.tableColumns(ctx, t)
		if err != nil {
			return res, err
		}
		if err := enc.Encode(tableHeader{T: t, C: cols, U: udts}); err != nil {
			return res, fmt.Errorf("backup: write header %s: %w", t, err)
		}
		rows, err := s.store.db.Query(ctx,
			fmt.Sprintf(`SELECT %s FROM %q`, quoteCols(cols), t))
		if err != nil {
			return res, fmt.Errorf("backup: query %s: %w", t, err)
		}
		for rows.Next() {
			vals, err := rows.Values()
			if err != nil {
				rows.Close()
				return res, fmt.Errorf("backup: values %s: %w", t, err)
			}
			if err := enc.Encode(rowLine{R: vals}); err != nil {
				rows.Close()
				return res, fmt.Errorf("backup: write row %s: %w", t, err)
			}
			res.rowCount++
		}
		if err := rows.Err(); err != nil {
			rows.Close()
			return res, fmt.Errorf("backup: iterate %s: %w", t, err)
		}
		rows.Close()
		res.tableCount++
	}
	if err := gz.Close(); err != nil {
		return res, fmt.Errorf("backup: close gzip: %w", err)
	}
	if fi, err := f.Stat(); err == nil {
		res.sizeBytes = fi.Size()
	}
	return res, nil
}

func quoteCols(cols []string) string {
	out := ""
	for i, c := range cols {
		if i > 0 {
			out += ", "
		}
		out += `"` + c + `"`
	}
	return out
}
