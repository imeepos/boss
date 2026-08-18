// 业务参数(SYS 域,阶段1):欠费阈值/预占有效期/对账周期等集中配置,热更新。
// 存储 biz_params(key PK, value JSONB, description);value 统一按 JSON 字符串出入参。
package user

import (
	"context"
	"encoding/json"
	"fmt"
)

// Param 业务参数行(前端展示:value 为去引号后的标量文本)。
type Param struct {
	Key   string `json:"key"`
	Value string `json:"value"`
	Desc  string `json:"desc"`
}

// ListParams 全量业务参数(量级小不分页)。
func (s *PGStore) ListParams(ctx context.Context) ([]Param, error) {
	rows, err := s.db.Query(ctx,
		`SELECT key, value::text, COALESCE(description, '') FROM biz_params ORDER BY key`)
	if err != nil {
		return nil, fmt.Errorf("user: list params: %w", err)
	}
	defer rows.Close()
	out := make([]Param, 0)
	for rows.Next() {
		var p Param
		if err := rows.Scan(&p.Key, &p.Value, &p.Desc); err != nil {
			return nil, fmt.Errorf("user: scan param: %w", err)
		}
		p.Value = unquoteJSON(p.Value)
		out = append(out, p)
	}
	return out, rows.Err()
}

// UpdateParam 更新单个参数(不存在则插入);updatedBy 记录操作账号。
func (s *PGStore) UpdateParam(ctx context.Context, key, value string, updatedBy int64) error {
	if key == "" || len(key) > 128 {
		return ErrInvalidInput
	}
	if !json.Valid([]byte(value)) {
		// 非合法 JSON 一律按 JSON 字符串存储,保持 JSONB 约束。
		b, err := json.Marshal(value)
		if err != nil {
			return ErrInvalidInput
		}
		value = string(b)
	}
	tag, err := s.db.Exec(ctx, `
		INSERT INTO biz_params(key, value, updated_by, updated_at)
		VALUES($1, $2::jsonb, NULLIF($3,0), now())
		ON CONFLICT (key) DO UPDATE
		SET value = EXCLUDED.value, updated_by = EXCLUDED.updated_by, updated_at = now()`,
		key, value, updatedBy)
	if err != nil {
		return fmt.Errorf("user: update param: %w", err)
	}
	_ = tag
	return nil
}

// unquoteJSON JSONB 文本转标量展示:JSON 字符串去引号,其余(数字/对象)原样。
func unquoteJSON(s string) string {
	var str string
	if err := json.Unmarshal([]byte(s), &str); err == nil {
		return str
	}
	return s
}
