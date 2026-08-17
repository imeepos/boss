package worker

import (
	"context"
	"time"
)

// Membership 师傅班组归属台账(每次归属班组的时间段,历史不随当前 group 漂移)。
type Membership struct {
	ID                int64      `json:"id"`
	WorkerID          int64      `json:"workerId"`
	GroupID           int64      `json:"groupId"`
	GroupName         string     `json:"groupName"`
	LegalEntityID     int64      `json:"legalEntityId"`
	LegalEntityName   string     `json:"legalEntityName"`
	RegionID          int64      `json:"regionId"`
	RegionName        string     `json:"regionName"`
	Reason            string     `json:"reason"`
	OperatorAccountID int64      `json:"operatorAccountId"` // 0=空
	EffectiveFrom     time.Time  `json:"effectiveFrom"`
	EffectiveTo       *time.Time `json:"effectiveTo,omitempty"` // nil=至今
}

// Settings 师傅接单设置(师傅1:1)。
type Settings struct {
	ID          int64      `json:"id"`
	WorkerID    int64      `json:"workerId"`
	Accepting   bool       `json:"accepting"`
	RadiusKm    int16      `json:"radiusKm"`
	AcceptTypes string     `json:"acceptTypes"`
	UpdatedAt   *time.Time `json:"updatedAt,omitempty"` // nil=未更新
}

// Message 师傅站内消息(系统/调度下发)。
type Message struct {
	ID       int64     `json:"id"`
	WorkerID int64     `json:"workerId"`
	Level    string    `json:"level"` // INFO/WARN/URGENT
	Title    string    `json:"title"`
	Content  string    `json:"content"`
	SentAt   time.Time `json:"sentAt"`
	Read     bool      `json:"read"`
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
