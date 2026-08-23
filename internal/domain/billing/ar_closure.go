package billing

import (
	"context"
	"time"
)

// AgingSnapshot is the customer daily AR distribution.
type AgingSnapshot struct {
	ID           int64     `json:"id"`
	CustomerID   int64     `json:"customerId"`
	SnapshotDate time.Time `json:"snapshotDate"`
	Current      float64   `json:"current"`
	Days1To30    float64   `json:"days1To30"`
	Days31To60   float64   `json:"days31To60"`
	Days61To90   float64   `json:"days61To90"`
	Days90Plus   float64   `json:"days90Plus"`
	Total        float64   `json:"total"`
}

type PaymentPromise struct {
	ID          int64      `json:"id"`
	CustomerID  int64      `json:"customerId"`
	Amount      float64    `json:"amount"`
	PromisedAt  time.Time  `json:"promisedAt"`
	DueAt       time.Time  `json:"dueAt"`
	Status      string     `json:"status"`
	FulfilledAt *time.Time `json:"fulfilledAt,omitempty"`
}

// Writeoff extended with approval lifecycle.
type Writeoff struct {
	ID           int64      `json:"id"`
	CustomerID   int64      `json:"customerId"`
	Amount       float64    `json:"amount"`
	Reason       string     `json:"reason"`
	ApprovedBy   *int64     `json:"approvedBy,omitempty"`
	ApprovedAt   *time.Time `json:"approvedAt,omitempty"`
	Status       string     `json:"status"`
	RequestedBy  *int64     `json:"requestedBy,omitempty"`
	ApprovedNote string     `json:"approvedNote,omitempty"`
}

// CreditProfile evaluation rules produce explainable credit levels.
type CreditProfile struct {
	CustomerID    int64     `json:"customerId"`
	CreditLevel   string    `json:"creditLevel"`
	RiskScore     float64   `json:"riskScore"`
	StopThreshold float64   `json:"stopThreshold"`
	EvaluatedAt   time.Time `json:"evaluatedAt"`
}

// ReplayEvent tracks a source event for AR consistency replay.
type ReplayEvent struct {
	ID         int64     `json:"id"`
	CustomerID int64     `json:"customerId"`
	SourceType string    `json:"sourceType"`
	SourceID   int64     `json:"sourceId"`
	EventType  string    `json:"eventType"`
	Payload    string    `json:"payload"`
	CreatedAt  time.Time `json:"createdAt"`
}

// Writeoff status values.
const (
	WriteoffPending  = "PENDING"
	WriteoffApproved = "APPROVED"
	WriteoffRejected = "REJECTED"
)

// PaymentPromise status values.
const (
	PromiseOpen      = "OPEN"
	PromiseFulfilled = "FULFILLED"
	PromiseBroken    = "BROKEN"
	PromiseCanceled  = "CANCELED"
)

// ARClosureService provides AR aging, promises, and writeoffs.
type ARClosureService interface {
	GenerateAgingSnapshots(context.Context, time.Time) (int, error)
	ListAgingSnapshots(context.Context, time.Time, int64) ([]AgingSnapshot, error)
	ListPaymentPromises(context.Context, int64) ([]PaymentPromise, error)
	CreatePaymentPromise(context.Context, PaymentPromise) (int64, error)
	UpdatePaymentPromiseStatus(context.Context, int64, string) error
	ListWriteoffs(context.Context, int64) ([]Writeoff, error)
	CreateWriteoff(context.Context, Writeoff) (int64, error)
	ApproveWriteoff(context.Context, int64, int64) error
	RejectWriteoff(context.Context, int64, string) error
	EvaluateCreditProfile(context.Context, int64) (*CreditProfile, error)
	GenerateCollectionTasks(context.Context, time.Time) (int, error)
	RecordReplayEvent(context.Context, int64, string, int64, string, string) error
	ReplayEvents(context.Context, string) (int, error)
	ListReplayEvents(context.Context, string) ([]ReplayEvent, error)
}
