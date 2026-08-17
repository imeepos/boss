package worker

import "context"

// Performance 师傅绩效(一行=师傅该月在该班组该区域的一段贡献)。
type Performance struct {
	ID              int64
	WorkerID        int64
	GroupID         int64
	GroupName       string
	LegalEntityID   int64
	LegalEntityName string
	RegionID        int64
	RegionName      string
	Period          string // 2026-08
	Finished        int32
	OnTimeRate      int16
	Score           float64
}

// Commission 师傅佣金(按月×班组×区域计提)。
type Commission struct {
	ID              int64
	WorkerID        int64
	GroupID         int64
	GroupName       string
	LegalEntityID   int64
	LegalEntityName string
	RegionID        int64
	RegionName      string
	Period          string
	Formula         string
	Amount          float64
}

// Schedule 师傅考勤(按月×班组×区域统计)。
type Schedule struct {
	ID              int64
	WorkerID        int64
	GroupID         int64
	GroupName       string
	LegalEntityID   int64
	LegalEntityName string
	RegionID        int64
	RegionName      string
	Month           string
	BusyDays        int16
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
