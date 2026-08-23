package billing

import (
	"context"
	"fmt"
)

func (s *PGStore) ListCollectionTasks(ctx context.Context, status string) ([]CollectionTaskItem, error) {
	rows, err := s.db.Query(ctx, `
		SELECT t.id, t.customer_id, COALESCE(c.name, ''), t.task_type, t.priority, t.status,
		       t.due_at::text, COALESCE(a.amount, 0), COALESCE(a.days, 0), COALESCE(t.note, '')
		FROM ar_collection_tasks t
		LEFT JOIN customers c ON c.id = t.customer_id
		LEFT JOIN arrears a ON a.customer_id = t.customer_id
		WHERE ($1 = '' OR t.status = $1)
		ORDER BY CASE t.priority WHEN 'URGENT' THEN 0 WHEN 'HIGH' THEN 1 ELSE 2 END, t.due_at, t.id`, status)
	if err != nil {
		return nil, fmt.Errorf("billing: list collection tasks: %w", err)
	}
	defer rows.Close()
	out := make([]CollectionTaskItem, 0)
	for rows.Next() {
		var item CollectionTaskItem
		if err := rows.Scan(&item.ID, &item.CustomerID, &item.CustomerName, &item.TaskType, &item.Priority,
			&item.Status, &item.DueAt, &item.Amount, &item.Days, &item.Note); err != nil {
			return nil, fmt.Errorf("billing: scan collection task: %w", err)
		}
		out = append(out, item)
	}
	return out, rows.Err()
}

func (s *PGStore) UpdateCollectionTask(ctx context.Context, id int64, status, outcome, note string) error {
	tag, err := s.db.Exec(ctx, `
		UPDATE ar_collection_tasks SET status = $2, outcome = NULLIF($3, ''), note = NULLIF($4, ''),
			completed_at = CASE WHEN $2 IN ('DONE', 'FAILED') THEN now() ELSE completed_at END
		WHERE id = $1`, id, status, outcome, note)
	if err != nil {
		return fmt.Errorf("billing: update collection task: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}
