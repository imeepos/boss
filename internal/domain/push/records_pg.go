package push

// records PostgreSQL 实现(迁移 000096 push_records)。

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// RecordsPGStore RecordsStore 的 PostgreSQL 实现。
type RecordsPGStore struct{ db devicesDB }

// NewRecordsPGStore 构造 PG 留痕存储。
func NewRecordsPGStore(db devicesDB) *RecordsPGStore { return &RecordsPGStore{db: db} }

// extrasJSON extras 序列化(pg 落库用);nil 返回 '{}'。
func extrasJSON(m map[string]string) []byte {
	if len(m) == 0 {
		return []byte("{}")
	}
	b, err := json.Marshal(m)
	if err != nil {
		return []byte("{}")
	}
	return b
}

// Record 追加一行留痕。
func (s *RecordsPGStore) Record(ctx context.Context, r Record) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO push_records
			(subject_type, subject_id, registration_id, title, alert, extras, status, msg_id, error)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`,
		r.SubjectType, r.SubjectID, r.RegistrationID, r.Title, r.Alert,
		extrasJSON(r.Extras), r.Status, r.MsgID, r.Error)
	if err != nil {
		return fmt.Errorf("push: record: %w", err)
	}
	return nil
}

// ListBySubject 主体留痕(新→旧,limit 上限 200)。
func (s *RecordsPGStore) ListBySubject(ctx context.Context, subjectType string, subjectID int64, limit int) ([]Record, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	rows, err := s.db.Query(ctx, `
		SELECT id, subject_type, subject_id, registration_id, title, alert, extras, status, msg_id, error, created_at
		FROM push_records WHERE subject_type = $1 AND subject_id = $2
		ORDER BY created_at DESC LIMIT $3`, subjectType, subjectID, limit)
	if err != nil {
		return nil, fmt.Errorf("push: list records: %w", err)
	}
	defer rows.Close()
	return scanRecords(rows)
}

func scanRecords(rows pgx.Rows) ([]Record, error) {
	out := []Record{}
	for rows.Next() {
		var r Record
		var extras []byte
		if err := rows.Scan(&r.ID, &r.SubjectType, &r.SubjectID, &r.RegistrationID,
			&r.Title, &r.Alert, &extras, &r.Status, &r.MsgID, &r.Error, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("push: scan record: %w", err)
		}
		_ = json.Unmarshal(extras, &r.Extras)
		out = append(out, r)
	}
	return out, rows.Err()
}

// compile-time 接口断言。
var _ RecordsStore = (*RecordsPGStore)(nil)
