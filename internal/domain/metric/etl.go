package metric

import (
	"context"
	"errors"
	"strings"
	"time"
)

const (
	ETLSuccess = "SUCCESS"
	ETLFailed  = "FAILED"
	ETLRunning = "RUNNING"
	ETLOverdue = "OVERDUE"
	Fresh      = "fresh"
	Stale      = "stale"
	Overdue    = "overdue"
)

type ETLJob struct {
	ID                int64      `json:"id"`
	JobKey            string     `json:"jobKey"`
	Name              string     `json:"name"`
	SourceDomain      string     `json:"sourceDomain"`
	TargetDomain      string     `json:"targetDomain"`
	CronExpr          string     `json:"cronExpr"`
	RefreshCadence    string     `json:"refreshCadence"`
	Owner             string     `json:"owner"`
	Enabled           bool       `json:"enabled"`
	LastRunAt         *time.Time `json:"lastRunAt"`
	LastStatus        string     `json:"lastStatus"`
	LastDurationMS    int64      `json:"lastDurationMs"`
	LastRowsAffected  int64      `json:"lastRowsAffected"`
	LatenessThreshold int        `json:"latenessThreshold"`
	CreatedAt         time.Time  `json:"createdAt"`
	UpdatedAt         time.Time  `json:"updatedAt"`
}

type ETLJobRun struct {
	ID           int64      `json:"id"`
	JobKey       string     `json:"jobKey"`
	StartedAt    time.Time  `json:"startedAt"`
	FinishedAt   *time.Time `json:"finishedAt"`
	Status       string     `json:"status"`
	RowsAffected int64      `json:"rowsAffected"`
	Error        string     `json:"error"`
}

type Freshness struct {
	JobKey            string     `json:"jobKey"`
	Name              string     `json:"name"`
	Status            string     `json:"status"`
	LastRunAt         *time.Time `json:"lastRunAt"`
	LatenessThreshold int        `json:"latenessThreshold"`
}

type ETLService interface {
	ListJobs(context.Context) ([]ETLJob, error)
	GetJob(context.Context, string) (*ETLJob, error)
	UpsertJob(context.Context, ETLJob) (*ETLJob, error)
	DisableJob(context.Context, string) error
	RecordRun(context.Context, ETLJobRun) error
	ListFreshness(context.Context) ([]Freshness, error)
	ListRuns(context.Context, string) ([]ETLJobRun, error)
}

var ErrETLNotFound = errors.New("metric: etl job not found")

func freshness(now time.Time, j ETLJob) string {
	if j.LastRunAt == nil {
		return Overdue
	}
	age := now.Sub(*j.LastRunAt)
	threshold := time.Duration(j.LatenessThreshold) * time.Minute
	if age > threshold*2 {
		return Overdue
	}
	if age > threshold {
		return Stale
	}
	return Fresh
}

type MemoryETLStore struct {
	Jobs   map[string]ETLJob
	Runs   map[string][]ETLJobRun
	NextID int64
}

func NewMemoryETLStore() *MemoryETLStore {
	return &MemoryETLStore{Jobs: map[string]ETLJob{}, Runs: map[string][]ETLJobRun{}, NextID: 1}
}
func (s *MemoryETLStore) ListJobs(_ context.Context) ([]ETLJob, error) {
	out := make([]ETLJob, 0, len(s.Jobs))
	for _, j := range s.Jobs {
		out = append(out, j)
	}
	return out, nil
}
func (s *MemoryETLStore) GetJob(_ context.Context, k string) (*ETLJob, error) {
	j, ok := s.Jobs[k]
	if !ok {
		return nil, ErrETLNotFound
	}
	return &j, nil
}
func (s *MemoryETLStore) UpsertJob(_ context.Context, j ETLJob) (*ETLJob, error) {
	if j.JobKey == "" || j.Name == "" {
		return nil, errors.New("metric: jobKey and name required")
	}
	if j.LatenessThreshold <= 0 {
		j.LatenessThreshold = 60
	}
	old, ok := s.Jobs[j.JobKey]
	if ok {
		j.ID = old.ID
		j.CreatedAt = old.CreatedAt
	} else {
		j.Enabled = true // 新建缺省启用,对齐 000132 的 DEFAULT TRUE
	}
	if j.ID == 0 {
		j.ID = s.NextID
		s.NextID++
	}
	j.UpdatedAt = time.Now()
	if j.CreatedAt.IsZero() {
		j.CreatedAt = j.UpdatedAt
	}
	s.Jobs[j.JobKey] = j
	return &j, nil
}
func (s *MemoryETLStore) DisableJob(_ context.Context, k string) error {
	j, ok := s.Jobs[k]
	if !ok {
		return ErrETLNotFound
	}
	j.Enabled = false
	s.Jobs[k] = j
	return nil
}
func normalizeRun(r ETLJobRun) (ETLJobRun, error) {
	r.JobKey = strings.TrimSpace(r.JobKey)
	if r.JobKey == "" {
		return r, errors.New("metric: jobKey required")
	}
	if r.RowsAffected < 0 {
		return r, errors.New("metric: rowsAffected cannot be negative")
	}
	if r.Status != ETLRunning && r.Status != ETLSuccess && r.Status != ETLFailed {
		return r, errors.New("metric: invalid run status")
	}
	if r.StartedAt.IsZero() {
		r.StartedAt = time.Now()
	}
	if r.Status == ETLRunning && r.FinishedAt != nil {
		return r, errors.New("metric: running run cannot have finishedAt")
	}
	if r.FinishedAt != nil && r.FinishedAt.Before(r.StartedAt) {
		return r, errors.New("metric: finishedAt before startedAt")
	}
	if (r.Status == ETLSuccess || r.Status == ETLFailed) && r.FinishedAt == nil {
		now := time.Now()
		r.FinishedAt = &now
	}
	return r, nil
}

func (s *MemoryETLStore) RecordRun(_ context.Context, r ETLJobRun) error {
	var err error
	r, err = normalizeRun(r)
	if err != nil {
		return err
	}
	j, ok := s.Jobs[r.JobKey]
	if !ok {
		return ErrETLNotFound
	}
	r.ID = s.NextID
	s.NextID++
	s.Runs[r.JobKey] = append(s.Runs[r.JobKey], r)
	j.LastRunAt = &r.StartedAt
	j.LastStatus = r.Status
	j.LastRowsAffected = r.RowsAffected
	if r.FinishedAt != nil {
		j.LastDurationMS = r.FinishedAt.Sub(r.StartedAt).Milliseconds()
	}
	s.Jobs[r.JobKey] = j
	return nil
}
func (s *MemoryETLStore) ListFreshness(_ context.Context) ([]Freshness, error) {
	now := time.Now()
	out := []Freshness{}
	for _, j := range s.Jobs {
		if !j.Enabled {
			continue // 禁用任务不监控:尚未部署真实执行器的台账保持静默,不产逾期派单
		}
		out = append(out, Freshness{JobKey: j.JobKey, Name: j.Name, Status: freshness(now, j), LastRunAt: j.LastRunAt, LatenessThreshold: j.LatenessThreshold})
	}
	return out, nil
}
func (s *MemoryETLStore) ListRuns(_ context.Context, k string) ([]ETLJobRun, error) {
	if _, ok := s.Jobs[k]; !ok {
		return nil, ErrETLNotFound
	}
	return s.Runs[k], nil
}
