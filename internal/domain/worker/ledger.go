package worker

import (
	"context"
	"time"
)

// Membership 师傅班组归属台账(每次归属班组的时间段,历史不随当前 group 漂移)。
type Membership struct {
	ID                int64
	WorkerID          int64
	GroupID           int64
	GroupName         string
	LegalEntityID     int64
	LegalEntityName   string
	RegionID          int64
	RegionName        string
	Reason            string
	OperatorAccountID int64       // 0=空
	EffectiveFrom     time.Time
	EffectiveTo       *time.Time  // nil=至今
}

// Settings 师傅接单设置(师傅1:1)。
type Settings struct {
	ID          int64
	WorkerID    int64
	Accepting   bool
	RadiusKm    int16
	AcceptTypes string
	UpdatedAt   *time.Time // nil=未更新
}

// Message 师傅站内消息(系统/调度下发)。
type Message struct {
	ID       int64
	WorkerID int64
	Level    string // INFO/WARN/URGENT
	Title    string
	Content  string
	SentAt   time.Time
	Read     bool
}

// WorkerLedgerService 师傅台账域服务口(阶段2):归属台账/接单设置/站内消息。
type WorkerLedgerService interface {
	ListMemberships(ctx context.Context, workerID int64) ([]Membership, error)
	AppendMembership(ctx context.Context, m Membership) (int64, error)
	GetSettings(ctx context.Context, workerID int64) (*Settings, error)
	UpsertSettings(ctx context.Context, s Settings) (int64, error)
	ListMessages(ctx context.Context, workerID int64) ([]Message, error)
	SendMessage(ctx context.Context, m Message) (int64, error)
}
