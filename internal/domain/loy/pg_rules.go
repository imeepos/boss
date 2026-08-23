package loy

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// lifetimeEarned 累计获得积分(正向流水合计,含已过期),等级匹配口径。
func (s *PGStore) lifetimeEarned(ctx context.Context, customerID int64) (int64, error) {
	var earned int64
	err := s.db.QueryRow(ctx,
		`SELECT COALESCE(SUM(delta),0) FROM loy_point_entries
		 WHERE customer_id=$1 AND delta>0`, customerID).Scan(&earned)
	if err != nil {
		return 0, fmt.Errorf("loy: lifetime earned: %w", err)
	}
	return earned, nil
}

// ListLevels 等级列表(按门槛升序)。
func (s *PGStore) ListLevels(ctx context.Context) ([]Level, error) {
	rows, err := s.db.Query(ctx,
		`SELECT level_id, name, min_points, status FROM loy_levels
		 ORDER BY min_points ASC, level_id ASC`)
	if err != nil {
		return nil, fmt.Errorf("loy: list levels: %w", err)
	}
	defer rows.Close()
	out := make([]Level, 0)
	for rows.Next() {
		var l Level
		if err := rows.Scan(&l.LevelID, &l.Name, &l.MinPoints, &l.Status); err != nil {
			return nil, fmt.Errorf("loy: scan level: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}

// CreateLevel 新建等级。
func (s *PGStore) CreateLevel(ctx context.Context, l Level) (int64, error) {
	var id int64
	if err := s.db.QueryRow(ctx,
		`INSERT INTO loy_levels(name, min_points) VALUES($1,$2) RETURNING level_id`,
		l.Name, l.MinPoints).Scan(&id); err != nil {
		return 0, fmt.Errorf("loy: create level: %w", err)
	}
	return id, nil
}

// DisableLevel 停用等级。
func (s *PGStore) DisableLevel(ctx context.Context, levelID int64) error {
	ct, err := s.db.Exec(ctx,
		`UPDATE loy_levels SET status='DISABLED' WHERE level_id=$1`, levelID)
	if err != nil {
		return fmt.Errorf("loy: disable level: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// TierOf 客户当前等级:累计获得积分最高达标档。
func (s *PGStore) TierOf(ctx context.Context, customerID int64) (*Level, error) {
	earned, err := s.lifetimeEarned(ctx, customerID)
	if err != nil {
		return nil, err
	}
	var l Level
	err = s.db.QueryRow(ctx,
		`SELECT level_id, name, min_points, status FROM loy_levels
		 WHERE status='ENABLED' AND min_points <= $1
		 ORDER BY min_points DESC LIMIT 1`, earned).Scan(
		&l.LevelID, &l.Name, &l.MinPoints, &l.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("loy: tier of: %w", err)
	}
	return &l, nil
}

// ListTasks 任务列表。
func (s *PGStore) ListTasks(ctx context.Context) ([]Task, error) {
	rows, err := s.db.Query(ctx,
		`SELECT task_id, code, name, points, period, status FROM loy_tasks
		 ORDER BY task_id ASC`)
	if err != nil {
		return nil, fmt.Errorf("loy: list tasks: %w", err)
	}
	defer rows.Close()
	out := make([]Task, 0)
	for rows.Next() {
		var t Task
		if err := rows.Scan(&t.TaskID, &t.Code, &t.Name, &t.Points, &t.Period, &t.Status); err != nil {
			return nil, fmt.Errorf("loy: scan task: %w", err)
		}
		out = append(out, t)
	}
	return out, rows.Err()
}

// CreateTask 新建任务。
func (s *PGStore) CreateTask(ctx context.Context, t Task) (int64, error) {
	var id int64
	if err := s.db.QueryRow(ctx,
		`INSERT INTO loy_tasks(code, name, points, period) VALUES($1,$2,$3,$4)
		 RETURNING task_id`, t.Code, t.Name, t.Points, t.Period).Scan(&id); err != nil {
		return 0, fmt.Errorf("loy: create task: %w", err)
	}
	return id, nil
}

// DisableTask 停用任务。
func (s *PGStore) DisableTask(ctx context.Context, taskID int64) error {
	ct, err := s.db.Exec(ctx,
		`UPDATE loy_tasks SET status='DISABLED' WHERE task_id=$1`, taskID)
	if err != nil {
		return fmt.Errorf("loy: disable task: %w", err)
	}
	if ct.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// periodKeyOf 当前周期键:ONE_TIME 固定 ”,DAILY/MONTHLY 按本地时间截断。
func periodKeyOf(period string, now time.Time) string {
	switch period {
	case "DAILY":
		return now.Format("2006-01-02")
	case "MONTHLY":
		return now.Format("2006-01")
	}
	return ""
}

// taskOf 取任务定义。
func (s *PGStore) taskOf(ctx context.Context, taskID int64) (*Task, error) {
	var t Task
	err := s.db.QueryRow(ctx,
		`SELECT task_id, code, name, points, period, status FROM loy_tasks WHERE task_id=$1`,
		taskID).Scan(&t.TaskID, &t.Code, &t.Name, &t.Points, &t.Period, &t.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("loy: task of: %w", err)
	}
	return &t, nil
}

// CompleteTask 完成任务发积分:completions 唯一约束保周期幂等,重复返回 ErrConflict。
func (s *PGStore) CompleteTask(ctx context.Context, customerID, taskID int64) (int64, error) {
	t, err := s.taskOf(ctx, taskID)
	if err != nil {
		return 0, err
	}
	if t.Status != "ENABLED" {
		return 0, fmt.Errorf("%w: 任务已停用", ErrConflict)
	}
	key := periodKeyOf(t.Period, time.Now())

	tx, err := s.db.(beginner).Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("loy: begin task tx: %w", err)
	}
	defer tx.Rollback(ctx)
	ct, err := tx.Exec(ctx,
		`INSERT INTO loy_task_completions(task_id, customer_id, period_key, points)
		 VALUES($1,$2,$3,$4)`, taskID, customerID, key, t.Points)
	if err != nil || ct.RowsAffected() == 0 {
		return 0, fmt.Errorf("%w: 周期内已完成", ErrConflict)
	}
	after, err := applyDelta(ctx, tx, customerID, t.Points)
	if err != nil {
		return 0, err
	}
	if err := insertEntry(ctx, tx, customerID, t.Points, after, ReasonTask, taskID); err != nil {
		return 0, err
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("loy: commit task tx: %w", err)
	}
	return after, nil
}

// TaskStatus 客户各任务最近完成时间(周期键匹配当前周期)。
func (s *PGStore) TaskStatus(ctx context.Context, customerID int64) (map[int64]string, error) {
	rows, err := s.db.Query(ctx,
		`SELECT task_id, to_char(created_at,'YYYY-MM-DD"T"HH24:MI:SSOF')
		 FROM loy_task_completions c
		 WHERE customer_id=$1 AND period_key = CASE (SELECT period FROM loy_tasks t WHERE t.task_id=c.task_id)
		     WHEN 'DAILY' THEN to_char(now(),'YYYY-MM-DD')
		     WHEN 'MONTHLY' THEN to_char(now(),'YYYY-MM')
		     ELSE '' END`, customerID)
	if err != nil {
		return nil, fmt.Errorf("loy: task status: %w", err)
	}
	defer rows.Close()
	out := make(map[int64]string)
	for rows.Next() {
		var id int64
		var at string
		if err := rows.Scan(&id, &at); err != nil {
			return nil, fmt.Errorf("loy: scan completion: %w", err)
		}
		out[id] = at
	}
	return out, rows.Err()
}
