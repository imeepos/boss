package worker

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

const locationCols = `id, worker_id, lat, lng, accuracy_m, speed_mps, bearing, reported_at`

func (s *PGStore) ReportLocation(ctx context.Context, location Location) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO worker_locations(worker_id, lat, lng, accuracy_m, speed_mps, bearing, reported_at)
		VALUES($1,$2,$3,$4,$5,$6,$7)`, location.WorkerID, location.Lat, location.Lng,
		location.AccuracyM, location.SpeedMPS, location.Bearing, location.ReportedAt)
	if err != nil {
		return fmt.Errorf("worker: report location: %w", err)
	}
	return nil
}

func (s *PGStore) LatestLocation(ctx context.Context, workerID int64) (*Location, error) {
	var location Location
	err := s.db.QueryRow(ctx, `SELECT `+locationCols+` FROM worker_locations WHERE worker_id=$1 ORDER BY reported_at DESC, id DESC LIMIT 1`, workerID).
		Scan(&location.ID, &location.WorkerID, &location.Lat, &location.Lng, &location.AccuracyM, &location.SpeedMPS, &location.Bearing, &location.ReportedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("worker: latest location: %w", err)
	}
	return &location, nil
}

func (s *PGStore) LatestLocationForOrder(ctx context.Context, orderID int64) (*Location, error) {
	var location Location
	err := s.db.QueryRow(ctx, `
		SELECT `+locationCols+` FROM worker_locations l
		JOIN dispatch_tickets t ON t.worker_id=l.worker_id AND t.order_id=$1
		ORDER BY l.reported_at DESC, l.id DESC LIMIT 1`, orderID).
		Scan(&location.ID, &location.WorkerID, &location.Lat, &location.Lng, &location.AccuracyM, &location.SpeedMPS, &location.Bearing, &location.ReportedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, nil
		}
		return nil, fmt.Errorf("worker: order location: %w", err)
	}
	return &location, nil
}

func (s *PGStore) ListLocationHistory(ctx context.Context, workerID int64, since time.Time, limit int) ([]Location, error) {
	if limit <= 0 || limit > 200 {
		limit = 100
	}
	rows, err := s.db.Query(ctx, `SELECT `+locationCols+` FROM worker_locations WHERE worker_id=$1 AND reported_at >= $2 ORDER BY reported_at DESC, id DESC LIMIT $3`, workerID, since, limit)
	if err != nil {
		return nil, fmt.Errorf("worker: location history: %w", err)
	}
	defer rows.Close()
	locations := make([]Location, 0)
	for rows.Next() {
		var location Location
		if err := rows.Scan(&location.ID, &location.WorkerID, &location.Lat, &location.Lng, &location.AccuracyM, &location.SpeedMPS, &location.Bearing, &location.ReportedAt); err != nil {
			return nil, fmt.Errorf("worker: scan location: %w", err)
		}
		locations = append(locations, location)
	}
	return locations, rows.Err()
}
