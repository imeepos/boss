package cs

import (
	"context"
	"fmt"
	"time"
)

// TicketEvent is a single audit record on a CS ticket.
type TicketEvent struct {
	ID         int64     `json:"id"`
	TicketID   int64     `json:"ticketId"`
	EventType  string    `json:"eventType"`
	FromStatus string    `json:"fromStatus"`
	ToStatus   string    `json:"toStatus"`
	ActorID    int64     `json:"actorId"`
	Note       string    `json:"note"`
	CreatedAt  time.Time `json:"createdAt"`
}

// TicketService provides lifecycle operations for CS tickets.
type TicketService interface {
	TransitionTicket(ctx context.Context, ticketNo, status string, actor int64, note string) error
	EscalateTicket(ctx context.Context, ticketNo string, actor int64, note string) error
	ListTicketEvents(ctx context.Context, ticketID int64) ([]TicketEvent, error)
	ListTicketEventsByNo(ctx context.Context, ticketNo string) ([]TicketEvent, error)
}

// TransitionTicket transitions a complaint ticket to a new status,
// recording the event in cs_ticket_events and updating extension timestamps.
func (s *PGKnowledgeStore) TransitionTicket(ctx context.Context, ticketNo, status string, actor int64, note string) error {
	if status != Open && status != Processing && status != Closed {
		return fmt.Errorf("cs: invalid status %q", status)
	}
	tag, err := s.db.Exec(ctx, `
		WITH old AS (
			SELECT id, status FROM complaints WHERE ticket_no=$1
		), changed AS (
			UPDATE complaints c SET status=$2,
				closed_at = CASE WHEN $2='CLOSED' THEN COALESCE(c.closed_at, now()) ELSE c.closed_at END
			FROM old WHERE c.id=old.id AND old.status<>$2
			RETURNING c.id, old.status
		), ext AS (
			INSERT INTO cs_ticket_extensions(ticket_id, first_response_at, resolved_at, updated_by)
			SELECT id,
				CASE WHEN $2='PROCESSING' THEN now() ELSE NULL END,
				CASE WHEN $2='CLOSED' THEN now() ELSE NULL END,
				NULLIF($3, 0)
			FROM changed
			ON CONFLICT (ticket_id) DO UPDATE SET
				first_response_at = COALESCE(cs_ticket_extensions.first_response_at, EXCLUDED.first_response_at),
				resolved_at      = COALESCE(cs_ticket_extensions.resolved_at, EXCLUDED.resolved_at),
				updated_by       = EXCLUDED.updated_by,
				updated_at       = now()
		)
		INSERT INTO cs_ticket_events(ticket_id, event_type, from_status, to_status, actor_id, note)
		SELECT id, 'STATUS_CHANGED', status, $2, NULLIF($3,0), $4
		FROM changed`, ticketNo, status, actor, note)
	if err != nil {
		return fmt.Errorf("cs: transition ticket: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("cs: ticket %s not found or unchanged", ticketNo)
	}
	return nil
}

// EscalateTicket increments the escalation level and sets priority.
func (s *PGKnowledgeStore) EscalateTicket(ctx context.Context, ticketNo string, actor int64, note string) error {
	tag, err := s.db.Exec(ctx, `
		WITH changed AS (
			UPDATE cs_ticket_extensions e
			SET escalation_level = e.escalation_level + 1,
			    priority = CASE WHEN e.escalation_level+1 >= 2 THEN 'URGENT' ELSE 'HIGH' END,
			    updated_by = NULLIF($2,0),
			    updated_at = now()
			FROM complaints c
			WHERE c.ticket_no = $1 AND e.ticket_id = c.id AND c.status <> 'CLOSED'
			RETURNING e.ticket_id
		)
		INSERT INTO cs_ticket_events(ticket_id, event_type, actor_id, note)
		SELECT ticket_id, 'ESCALATED', NULLIF($2,0), COALESCE(NULLIF($3,''), 'SLA escalation')
		FROM changed`, ticketNo, actor, note)
	if err != nil {
		return fmt.Errorf("cs: escalate ticket: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("cs: ticket %s not found or already closed", ticketNo)
	}
	return nil
}

// ListTicketEventsByNo returns events for a complaint identified by ticketNo.
func (s *PGKnowledgeStore) ListTicketEventsByNo(ctx context.Context, ticketNo string) ([]TicketEvent, error) {
	rows, err := s.db.Query(ctx, `
		SELECT e.id, e.ticket_id, e.event_type, COALESCE(e.from_status,''),
		       COALESCE(e.to_status,''), COALESCE(e.actor_id,0),
		       COALESCE(e.note,''), e.created_at
		FROM cs_ticket_events e
		JOIN complaints c ON c.id = e.ticket_id
		WHERE c.ticket_no = $1
		ORDER BY e.created_at, e.id`, ticketNo)
	if err != nil {
		return nil, fmt.Errorf("cs: list ticket events by no: %w", err)
	}
	defer rows.Close()
	out := make([]TicketEvent, 0)
	for rows.Next() {
		var v TicketEvent
		if err := rows.Scan(&v.ID, &v.TicketID, &v.EventType,
			&v.FromStatus, &v.ToStatus, &v.ActorID, &v.Note, &v.CreatedAt); err != nil {
			return nil, fmt.Errorf("cs: scan ticket event: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
func (s *PGKnowledgeStore) ListTicketEvents(ctx context.Context, ticketID int64) ([]TicketEvent, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, ticket_id, event_type, COALESCE(from_status,''),
		       COALESCE(to_status,''), COALESCE(actor_id,0),
		       COALESCE(note,''), created_at
		FROM cs_ticket_events
		WHERE ticket_id = $1
		ORDER BY created_at, id`, ticketID)
	if err != nil {
		return nil, fmt.Errorf("cs: list ticket events: %w", err)
	}
	defer rows.Close()
	out := make([]TicketEvent, 0)
	for rows.Next() {
		var v TicketEvent
		if err := rows.Scan(&v.ID, &v.TicketID, &v.EventType,
			&v.FromStatus, &v.ToStatus, &v.ActorID, &v.Note, &v.CreatedAt); err != nil {
			return nil, fmt.Errorf("cs: scan ticket event: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}
