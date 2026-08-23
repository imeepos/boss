package cs

import (
	"context"
	"fmt"
	"time"
)

// Callback is a scheduled CS follow-up call.
type Callback struct {
	ID          int64      `json:"id"`
	TicketID    int64      `json:"ticketId"`
	CustomerID  int64      `json:"customerId"`
	ScheduledAt time.Time  `json:"scheduledAt"`
	CompletedAt *time.Time `json:"completedAt,omitempty"`
	Result      string     `json:"result,omitempty"`
	Rating      int16      `json:"rating,omitempty"`
	Comment     string     `json:"comment,omitempty"`
	OperatorID  int64      `json:"operatorId,omitempty"`
}

type CallbackService interface {
	ListCallbacks(context.Context) ([]Callback, error)
	CreateCallback(context.Context, Callback) (int64, error)
	UpdateCallback(context.Context, int64, Callback) error
	DeleteCallback(context.Context, int64) error
}

func (s *PGKnowledgeStore) ListCallbacks(ctx context.Context) ([]Callback, error) {
	rows, err := s.db.Query(ctx, `SELECT id,ticket_id,customer_id,scheduled_at,completed_at,COALESCE(result,''),COALESCE(rating,0),COALESCE(comment,''),COALESCE(operator_id,0) FROM cs_callbacks ORDER BY scheduled_at,id`)
	if err != nil {
		return nil, fmt.Errorf("cs: list callbacks: %w", err)
	}
	defer rows.Close()
	out := make([]Callback, 0)
	for rows.Next() {
		var c Callback
		if err := rows.Scan(&c.ID, &c.TicketID, &c.CustomerID, &c.ScheduledAt, &c.CompletedAt, &c.Result, &c.Rating, &c.Comment, &c.OperatorID); err != nil {
			return nil, fmt.Errorf("cs: scan callback: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

func (s *PGKnowledgeStore) CreateCallback(ctx context.Context, c Callback) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `INSERT INTO cs_callbacks(ticket_id,customer_id,scheduled_at,result,rating,comment,operator_id) VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`, c.TicketID, c.CustomerID, c.ScheduledAt, c.Result, c.Rating, c.Comment, c.OperatorID).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("cs: create callback: %w", err)
	}
	return id, nil
}

func (s *PGKnowledgeStore) UpdateCallback(ctx context.Context, id int64, c Callback) error {
	tag, err := s.db.Exec(ctx, `UPDATE cs_callbacks SET scheduled_at=$2,completed_at=$3,result=$4,rating=$5,comment=$6,operator_id=$7 WHERE id=$1`, id, c.ScheduledAt, c.CompletedAt, c.Result, c.Rating, c.Comment, c.OperatorID)
	if err != nil {
		return fmt.Errorf("cs: update callback: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrArticleNotFound
	}
	return nil
}

func (s *PGKnowledgeStore) DeleteCallback(ctx context.Context, id int64) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM cs_callbacks WHERE id=$1`, id)
	if err != nil {
		return fmt.Errorf("cs: delete callback: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrArticleNotFound
	}
	return nil
}
