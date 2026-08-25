package worker

import (
	"context"
	"time"
)

// Location 师傅端上报的 WGS84 实时位置。
type Location struct {
	ID         int64     `json:"id"`
	WorkerID   int64     `json:"workerId"`
	Lat        float64   `json:"lat"`
	Lng        float64   `json:"lng"`
	AccuracyM  float32   `json:"accuracyM"`
	SpeedMPS   float32   `json:"speedMps"`
	Bearing    float32   `json:"bearing"`
	ReportedAt time.Time `json:"reportedAt"`
}

// LocationService 管理师傅实时位置与最近轨迹。
type LocationService interface {
	ReportLocation(ctx context.Context, location Location) error
	LatestLocation(ctx context.Context, workerID int64) (*Location, error)
	LatestLocationForOrder(ctx context.Context, orderID int64) (*Location, error)
	ListLocationHistory(ctx context.Context, workerID int64, since time.Time, limit int) ([]Location, error)
}
