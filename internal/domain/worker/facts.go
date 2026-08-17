package worker

import "context"

// Performance 师傅绩效(一行=师傅该月在该班组该区域的一段贡献)。
type Performance struct {
	ID              int64   `json:"id"`
	WorkerID        int64   `json:"workerId"`
	GroupID         int64   `json:"groupId"`
	GroupName       string  `json:"groupName"`
	LegalEntityID   int64   `json:"legalEntityId"`
	LegalEntityName string  `json:"legalEntityName"`
	RegionID        int64   `json:"regionId"`
	RegionName      string  `json:"regionName"`
	Period          string  `json:"period"` // 2026-08
	Finished        int32   `json:"finished"`
	OnTimeRate      int16   `json:"onTimeRate"`
	Score           float64 `json:"score"`
}

// Commission 师傅佣金(按月×班组×区域计提)。
type Commission struct {
	ID              int64   `json:"id"`
	WorkerID        int64   `json:"workerId"`
	GroupID         int64   `json:"groupId"`
	GroupName       string  `json:"groupName"`
	LegalEntityID   int64   `json:"legalEntityId"`
	LegalEntityName string  `json:"legalEntityName"`
	RegionID        int64   `json:"regionId"`
	RegionName      string  `json:"regionName"`
	Period          string  `json:"period"`
	Formula         string  `json:"formula"`
	Amount          float64 `json:"amount"`
}

// Schedule 师傅考勤(按月×班组×区域统计)。
type Schedule struct {
	ID              int64  `json:"id"`
	WorkerID        int64  `json:"workerId"`
	GroupID         int64  `json:"groupId"`
	GroupName       string `json:"groupName"`
	LegalEntityID   int64  `json:"legalEntityId"`
	LegalEntityName string `json:"legalEntityName"`
	RegionID        int64  `json:"regionId"`
	RegionName      string `json:"regionName"`
	Month           string `json:"month"`
	BusyDays        int16  `json:"busyDays"`
}

// WorkerFactService 师傅月度事实域服务口(阶段2)。
type WorkerFactService interface {
	ListPerformances(ctx context.Context, workerID int64) ([]Performance, error)
	UpsertPerformance(ctx context.Context, p Performance) (int64, error)
	ListCommissions(ctx context.Context, workerID int64) ([]Commission, error)
	UpsertCommission(ctx context.Context, c Commission) (int64, error)
	ListSchedules(ctx context.Context, workerID int64) ([]Schedule, error)
	UpsertSchedule(ctx context.Context, s Schedule) (int64, error)
}
