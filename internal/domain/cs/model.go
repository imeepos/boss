package cs

import "time"

const (
	Open       = "OPEN"
	Processing = "PROCESSING"
	Closed     = "CLOSED"
)

// TicketExtension is the CS-owned extension of the legacy complaints record.
type TicketExtension struct {
	TicketID         int64
	Priority         string
	EscalationLevel  int16
	FirstResponseAt  *time.Time
	ResolvedAt       *time.Time
	SLADueAt         *time.Time
	ResolutionCode   string
}

// Metric names are stable dashboard contract keys.
const (
	FirstResponseRate           = "first_response_rate"
	FirstContactResolutionRate  = "first_contact_resolution_rate"
	SLAOverdueRate              = "sla_overdue_rate"
	RepeatTicketRate            = "repeat_ticket_rate"
	ArrearsRecoveryRate         = "arrears_recovery_rate"
)
