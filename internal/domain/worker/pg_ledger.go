package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

// ListMemberships 列出师傅班组归属台账;workerID=0 返回全部。
func (s *PGStore) ListMemberships(ctx context.Context, workerID int64) ([]Membership, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, worker_id, group_id, group_name, legal_entity_id, legal_entity_name,
		       region_id, COALESCE(region_name, ''), COALESCE(reason, ''), COALESCE(operator_account_id, 0),
		       effective_from, effective_to
		FROM worker_group_memberships WHERE ($1 = 0 OR worker_id = $1) ORDER BY effective_from, id`, workerID)
	if err != nil {
		return nil, fmt.Errorf("worker: list memberships: %w", err)
	}
	defer rows.Close()
	out := make([]Membership, 0)
	for rows.Next() {
		var m Membership
		var effTo pgtype.Timestamptz
		if err := rows.Scan(&m.ID, &m.WorkerID, &m.GroupID, &m.GroupName, &m.LegalEntityID, &m.LegalEntityName,
			&m.RegionID, &m.RegionName, &m.Reason, &m.OperatorAccountID, &m.EffectiveFrom, &effTo); err != nil {
			return nil, fmt.Errorf("worker: scan membership: %w", err)
		}
		if effTo.Valid {
			t := effTo.Time
			m.EffectiveTo = &t
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// AppendMembership 追加班组归属台账,返回自增 id。
func (s *PGStore) AppendMembership(ctx context.Context, m Membership) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO worker_group_memberships(worker_id, group_id, group_name, legal_entity_id, legal_entity_name, region_id, region_name, reason, operator_account_id, effective_from, effective_to)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`,
		m.WorkerID, m.GroupID, m.GroupName, m.LegalEntityID, m.LegalEntityName, m.RegionID, m.RegionName,
		m.Reason, idOrNil(m.OperatorAccountID), m.EffectiveFrom, m.EffectiveTo).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("worker: append membership: %w", err)
	}
	return id, nil
}

// GetSettings 按师傅查接单设置;未命中返回 ErrNotFound。
func (s *PGStore) GetSettings(ctx context.Context, workerID int64) (*Settings, error) {
	var st Settings
	var up pgtype.Timestamptz
	err := s.db.QueryRow(ctx,
		`SELECT id, worker_id, accepting, radius_km, accept_types, updated_at FROM worker_settings WHERE worker_id = $1`, workerID).
		Scan(&st.ID, &st.WorkerID, &st.Accepting, &st.RadiusKm, &st.AcceptTypes, &up)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("worker: get settings: %w", err)
	}
	if up.Valid {
		t := up.Time
		st.UpdatedAt = &t
	}
	return &st, nil
}

// UpsertSettings 写入/更新接单设置(师傅1:1),返回 id。
func (s *PGStore) UpsertSettings(ctx context.Context, st Settings) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO worker_settings(worker_id, accepting, radius_km, accept_types, updated_at)
		VALUES($1,$2,$3,$4,$5)
		ON CONFLICT (worker_id) DO UPDATE SET accepting = $2, radius_km = $3, accept_types = $4, updated_at = $5
		RETURNING id`,
		st.WorkerID, st.Accepting, st.RadiusKm, st.AcceptTypes, st.UpdatedAt).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("worker: upsert settings: %w", err)
	}
	return id, nil
}

// ListMessages 列出师傅站内消息;workerID=0 返回全部。
func (s *PGStore) ListMessages(ctx context.Context, workerID int64) ([]Message, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, worker_id, level, title, content, sent_at, read FROM worker_messages WHERE ($1 = 0 OR worker_id = $1) ORDER BY sent_at, id`, workerID)
	if err != nil {
		return nil, fmt.Errorf("worker: list messages: %w", err)
	}
	defer rows.Close()
	out := make([]Message, 0)
	for rows.Next() {
		var m Message
		if err := rows.Scan(&m.ID, &m.WorkerID, &m.Level, &m.Title, &m.Content, &m.SentAt, &m.Read); err != nil {
			return nil, fmt.Errorf("worker: scan message: %w", err)
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// SendMessage 下发站内消息,返回自增 id。
func (s *PGStore) SendMessage(ctx context.Context, m Message) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO worker_messages(worker_id, level, title, content, sent_at, read)
		VALUES($1,$2,$3,$4,$5,$6) RETURNING id`,
		m.WorkerID, m.Level, m.Title, m.Content, m.SentAt, m.Read).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("worker: send message: %w", err)
	}
	return id, nil
}

// AppendClock 追加考勤打卡流水,返回自增 id。
func (s *PGStore) AppendClock(ctx context.Context, a Attendance) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO worker_attendance(worker_id, clock_type, clocked_at)
		VALUES($1,$2,$3) RETURNING id`, a.WorkerID, a.ClockType, a.ClockedAt).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("worker: append clock: %w", err)
	}
	return id, nil
}

// ListClocks 取师傅当日打卡流水(自然日按本地时区,day 取其零点后区间)。
func (s *PGStore) ListClocks(ctx context.Context, workerID int64, day time.Time) ([]Attendance, error) {
	start := time.Date(day.Year(), day.Month(), day.Day(), 0, 0, 0, 0, day.Location())
	end := start.Add(24 * time.Hour)
	rows, err := s.db.Query(ctx, `
		SELECT id, worker_id, clock_type, clocked_at
		FROM worker_attendance
		WHERE worker_id = $1 AND clocked_at >= $2 AND clocked_at < $3
		ORDER BY clocked_at, id`, workerID, start, end)
	if err != nil {
		return nil, fmt.Errorf("worker: list clocks: %w", err)
	}
	defer rows.Close()
	out := make([]Attendance, 0)
	for rows.Next() {
		var a Attendance
		if err := rows.Scan(&a.ID, &a.WorkerID, &a.ClockType, &a.ClockedAt); err != nil {
			return nil, fmt.Errorf("worker: scan clock: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}
