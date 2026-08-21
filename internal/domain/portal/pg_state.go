package portal

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/jackc/pgx/v5"
)

// errNoRows 行不存在哨兵(pg.go 复用,避免重复 import pgx)。
var errNoRows = pgx.ErrNoRows

// GetPrefs 读取偏好;无记录返回空 Prefs(语言空串由端解释为默认语言)。
func (s *pgStore) GetPrefs(ctx context.Context, customerID int64) (*Prefs, error) {
	var raw []byte
	p := &Prefs{Notify: map[string]any{}}
	err := s.pool.QueryRow(ctx, `SELECT notify, language FROM portal_prefs WHERE customer_id=$1`, customerID).
		Scan(&raw, &p.Language)
	if errors.Is(err, errNoRows) {
		return p, nil
	}
	if err != nil {
		return nil, err
	}
	if err := json.Unmarshal(raw, &p.Notify); err != nil {
		return nil, err
	}
	return p, nil
}

// SavePrefs 保存偏好;notify 传 nil 表示仅更新语言,反之亦然(零值不清空另一侧)。
func (s *pgStore) SavePrefs(ctx context.Context, customerID int64, notify map[string]any, language string) error {
	cur, err := s.GetPrefs(ctx, customerID)
	if err != nil {
		return err
	}
	if notify != nil {
		cur.Notify = notify
	}
	if language != "" {
		cur.Language = language
	}
	raw, err := json.Marshal(cur.Notify)
	if err != nil {
		return err
	}
	// 简单协议下 []byte 会按文本格式化成 "[123 34 ...]" 导致 22P02,必须传 string。
	_, err = s.pool.Exec(ctx, `INSERT INTO portal_prefs(customer_id, notify, language) VALUES ($1,$2,$3)
		ON CONFLICT (customer_id) DO UPDATE SET notify=EXCLUDED.notify, language=EXCLUDED.language`,
		customerID, string(raw), cur.Language)
	return err
}

func (s *pgStore) Messages(ctx context.Context, customerID int64) ([]Message, error) {
	rows, err := s.pool.Query(ctx,
		`SELECT id, customer_id, payload, read, created_at FROM portal_messages
		 WHERE customer_id=$1 ORDER BY id DESC LIMIT 100`, customerID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var out []Message
	for rows.Next() {
		var m Message
		var raw []byte
		if err := rows.Scan(&m.ID, &m.CustomerID, &raw, &m.Read, &m.CreatedAt); err != nil {
			return nil, err
		}
		if err := json.Unmarshal(raw, &m.Payload); err != nil {
			return nil, err
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

func (s *pgStore) PutMessage(ctx context.Context, customerID int64, payload map[string]any) error {
	raw, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	// 同 SavePrefs:jsonb 参数传 string,不传 []byte(简单协议兼容)。
	_, err = s.pool.Exec(ctx, `INSERT INTO portal_messages(customer_id, payload) VALUES ($1,$2)`, customerID, string(raw))
	return err
}

func (s *pgStore) MarkAllRead(ctx context.Context, customerID int64) error {
	_, err := s.pool.Exec(ctx, `UPDATE portal_messages SET read=TRUE WHERE customer_id=$1`, customerID)
	return err
}

func (s *pgStore) MarkRead(ctx context.Context, customerID int64, messageID string) (bool, error) {
	tag, err := s.pool.Exec(ctx,
		`UPDATE portal_messages SET read=TRUE WHERE customer_id=$1 AND message_id=$2`,
		customerID, messageID)
	if err != nil {
		return false, err
	}
	return tag.RowsAffected() > 0, nil
}

func (s *pgStore) HasUnread(ctx context.Context, customerID int64) (bool, error) {
	var n int
	err := s.pool.QueryRow(ctx, `SELECT count(*) FROM portal_messages WHERE customer_id=$1 AND read=FALSE`, customerID).Scan(&n)
	return n > 0, err
}

func (s *pgStore) Balance(ctx context.Context, customerID int64) (float64, error) {
	var bal float64
	err := s.pool.QueryRow(ctx, `SELECT balance::float8 FROM portal_wallets WHERE customer_id=$1`, customerID).Scan(&bal)
	if errors.Is(err, errNoRows) {
		return 0, nil
	}
	return bal, err
}

func (s *pgStore) AdjustBalance(ctx context.Context, customerID int64, delta float64) error {
	_, err := s.pool.Exec(ctx, `INSERT INTO portal_wallets(customer_id, balance) VALUES ($1, $2)
		ON CONFLICT (customer_id) DO UPDATE SET balance = portal_wallets.balance + EXCLUDED.balance`,
		customerID, delta)
	return err
}
