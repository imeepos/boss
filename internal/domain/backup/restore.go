// 恢复执行器:读取 gzip JSONL 归档,逐行 INSERT ... ON CONFLICT DO NOTHING 追加导入。
// 语义 = 只补不删不改:冲突行跳过,既有数据不动;用于环境间配置/基础数据迁移。
// 已知限制(设计裁定):数组仅支持 text[](元素全为字符串);bytea 走 base64 往返。
package backup

import (
	"compress/gzip"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"
)

// streamLine 归档里的一行:表头(T/C/U)或数据行(R)。
type streamLine struct {
	T *string  `json:"t"`
	C []string `json:"c"`
	U []string `json:"u"`
	R []any    `json:"r"`
}

// runRestore 恢复主流程(goroutine 内执行)。
func (s *Service) runRestore(jobID int64, path string) {
	defer s.unlock()
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Minute)
	defer cancel()
	res, err := s.restore(ctx, path)
	if err != nil {
		s.store.FinishJob(ctx, jobID, StatusFailed, "", res.sizeBytes, res.tableCount, res.rowCount, err.Error())
		return
	}
	s.store.FinishJob(ctx, jobID, StatusSucceeded, "", res.sizeBytes, res.tableCount, res.rowCount, "")
}

// restore 读归档逐表回灌。
func (s *Service) restore(ctx context.Context, path string) (dumpResult, error) {
	f, err := os.Open(path)
	if err != nil {
		return dumpResult{}, fmt.Errorf("restore: open file: %w", err)
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return dumpResult{}, fmt.Errorf("restore: open gzip: %w", err)
	}
	var res dumpResult
	if fi, err := f.Stat(); err == nil {
		res.sizeBytes = fi.Size()
	}
	dec := json.NewDecoder(gz)
	dec.UseNumber() // 保 int64 精度:json.Number 以文本参数下发,由 PG 按列类型收窄
	var meta dumpMeta
	if err := dec.Decode(&meta); err != nil {
		return res, fmt.Errorf("restore: read meta: %w", err)
	}
	if meta.V != 1 {
		return res, fmt.Errorf("restore: unsupported archive version %d", meta.V)
	}
	var cur *tableHeader
	for {
		var ln streamLine
		if err := dec.Decode(&ln); err != nil {
			if err == io.EOF {
				break
			}
			return res, fmt.Errorf("restore: read line: %w", err)
		}
		if ln.T != nil { // 表头:切换当前表
			cur = &tableHeader{T: *ln.T, C: ln.C, U: ln.U}
			res.tableCount++
			continue
		}
		if cur == nil {
			return res, fmt.Errorf("restore: row before any table header")
		}
		args, err := coerceRow(cur.U, ln.R)
		if err != nil {
			return res, fmt.Errorf("restore: coerce row %s: %w", cur.T, err)
		}
		if _, err := s.store.db.Exec(ctx, insertSQL(cur.T, cur.C), args...); err != nil {
			return res, fmt.Errorf("restore: insert %s: %w", cur.T, err)
		}
		res.rowCount++
	}
	return res, nil
}

// coerceRow 按 udt 把 JSON 值转回 pgx 可编码参数。
func coerceRow(udts []string, vals []any) ([]any, error) {
	if len(udts) != len(vals) {
		return nil, fmt.Errorf("column count mismatch: want %d got %d", len(udts), len(vals))
	}
	out := make([]any, len(vals))
	for i, v := range vals {
		c, err := coerceValue(udts[i], v)
		if err != nil {
			return nil, err
		}
		out[i] = c
	}
	return out, nil
}

func coerceValue(udt string, v any) (any, error) {
	if v == nil {
		return nil, nil
	}
	switch {
	case udt == "bytea":
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("bytea value is %T, want base64 string", v)
		}
		return base64.StdEncoding.DecodeString(s)
	case len(udt) > 1 && udt[0] == '_': // 数组列:仅支持元素全字符串
		arr, ok := v.([]any)
		if !ok {
			return nil, fmt.Errorf("array column got %T", v)
		}
		out := make([]string, len(arr))
		for i, e := range arr {
			s, ok := e.(string)
			if !ok {
				return nil, fmt.Errorf("array element is %T, only string arrays supported", e)
			}
			out[i] = s
		}
		return out, nil
	default:
		return v, nil // json.Number/string/bool 由 PG 按列类型收窄
	}
}

// insertSQL 生成单行追加导入语句(冲突行静默跳过)。
func insertSQL(table string, cols []string) string {
	placeholders := ""
	for i := range cols {
		if i > 0 {
			placeholders += ","
		}
		placeholders += fmt.Sprintf("$%d", i+1)
	}
	return fmt.Sprintf(`INSERT INTO %q (%s) VALUES (%s) ON CONFLICT DO NOTHING`,
		table, quoteCols(cols), placeholders)
}
