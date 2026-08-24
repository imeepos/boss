package metric

import (
	"context"
	"fmt"
	"github.com/jackc/pgx/v5"
	"time"
)

const etlJobCols = `id,job_key,name,source_domain,target_domain,cron_expr,refresh_cadence,owner,enabled,last_run_at,last_status,last_duration_ms,last_rows_affected,lateness_threshold,created_at,updated_at`

func (s *PGStore) ListJobs(ctx context.Context) ([]ETLJob, error) {
	rows, e := s.db.Query(ctx, "SELECT "+etlJobCols+" FROM etl_job ORDER BY job_key")
	if e != nil {
		return nil, e
	}
	return scanETLJobs(rows)
}
func (s *PGStore) GetJob(ctx context.Context, k string) (*ETLJob, error) {
	rows, e := s.db.Query(ctx, "SELECT "+etlJobCols+" FROM etl_job WHERE job_key=$1", k)
	if e != nil {
		return nil, e
	}
	a, e := scanETLJobs(rows)
	if e != nil || len(a) == 0 {
		return nil, ErrETLNotFound
	}
	return &a[0], e
}
func (s *PGStore) UpsertJob(ctx context.Context, j ETLJob) (*ETLJob, error) {
	if j.JobKey == "" || j.Name == "" {
		return nil, fmt.Errorf("metric: jobKey and name required")
	}
	if j.LatenessThreshold <= 0 {
		j.LatenessThreshold = 60
	}
	_, e := s.db.Exec(ctx, `INSERT INTO etl_job (job_key,name,source_domain,target_domain,cron_expr,refresh_cadence,owner,enabled,lateness_threshold) VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9) ON CONFLICT (job_key) DO UPDATE SET name=EXCLUDED.name,source_domain=EXCLUDED.source_domain,target_domain=EXCLUDED.target_domain,cron_expr=EXCLUDED.cron_expr,refresh_cadence=EXCLUDED.refresh_cadence,owner=EXCLUDED.owner,enabled=EXCLUDED.enabled,lateness_threshold=EXCLUDED.lateness_threshold,updated_at=now()`, j.JobKey, j.Name, j.SourceDomain, j.TargetDomain, j.CronExpr, j.RefreshCadence, j.Owner, j.Enabled, j.LatenessThreshold)
	if e != nil {
		return nil, e
	}
	return s.GetJob(ctx, j.JobKey)
}
func (s *PGStore) DisableJob(ctx context.Context, k string) error {
	t, e := s.db.Exec(ctx, "UPDATE etl_job SET enabled=FALSE,updated_at=now() WHERE job_key=$1", k)
	if e != nil {
		return e
	}
	if t.RowsAffected() == 0 {
		return ErrETLNotFound
	}
	return nil
}
func (s *PGStore) RecordRun(ctx context.Context, r ETLJobRun) error {
	var err error
	r, err = normalizeRun(r)
	if err != nil {
		return err
	}
	_, e := s.db.Exec(ctx, `WITH run AS (INSERT INTO etl_job_run (job_key,started_at,finished_at,status,rows_affected,error) VALUES ($1,$2,$3,$4,$5,$6)) UPDATE etl_job SET last_run_at=$2,last_status=$4,last_rows_affected=$5,last_duration_ms=COALESCE(EXTRACT(EPOCH FROM ($3-$2))*1000,0)::bigint,updated_at=now() WHERE job_key=$1`, r.JobKey, r.StartedAt, r.FinishedAt, r.Status, r.RowsAffected, r.Error)
	return e
}
func (s *PGStore) ListFreshness(ctx context.Context) ([]Freshness, error) {
	rows, e := s.db.Query(ctx, `SELECT job_key,name,last_run_at,lateness_threshold,CASE WHEN last_run_at IS NULL OR now()-last_run_at > (lateness_threshold*2)*interval '1 minute' THEN 'overdue' WHEN now()-last_run_at > lateness_threshold*interval '1 minute' THEN 'stale' ELSE 'fresh' END FROM etl_job ORDER BY job_key`)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []Freshness{}
	for rows.Next() {
		var f Freshness
		if e := rows.Scan(&f.JobKey, &f.Name, &f.LastRunAt, &f.LatenessThreshold, &f.Status); e != nil {
			return nil, e
		}
		out = append(out, f)
	}
	return out, rows.Err()
}
func (s *PGStore) ListRuns(ctx context.Context, k string) ([]ETLJobRun, error) {
	rows, e := s.db.Query(ctx, "SELECT id,job_key,started_at,finished_at,status,rows_affected,error FROM etl_job_run WHERE job_key=$1 ORDER BY started_at DESC", k)
	if e != nil {
		return nil, e
	}
	defer rows.Close()
	out := []ETLJobRun{}
	for rows.Next() {
		var r ETLJobRun
		if e := rows.Scan(&r.ID, &r.JobKey, &r.StartedAt, &r.FinishedAt, &r.Status, &r.RowsAffected, &r.Error); e != nil {
			return nil, e
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
func scanETLJobs(rows pgx.Rows) ([]ETLJob, error) {
	defer rows.Close()
	out := []ETLJob{}
	for rows.Next() {
		var j ETLJob
		if e := rows.Scan(&j.ID, &j.JobKey, &j.Name, &j.SourceDomain, &j.TargetDomain, &j.CronExpr, &j.RefreshCadence, &j.Owner, &j.Enabled, &j.LastRunAt, &j.LastStatus, &j.LastDurationMS, &j.LastRowsAffected, &j.LatenessThreshold, &j.CreatedAt, &j.UpdatedAt); e != nil {
			return nil, e
		}
		out = append(out, j)
	}
	return out, rows.Err()
}

var _ = time.Time{}
