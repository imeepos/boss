package device

import (
	"time"

	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// dbtx 是 PGStore 依赖的最小数据库接口;*pgxpool.Pool 天然满足,单测用 pgxmock 注入。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// PGStore 是 DeviceService 接口的 PostgreSQL 实现(阶段7)。
type PGStore struct {
	db dbtx
}

// NewPGStore 构造 PGStore;db 传 *pgxpool.Pool 或测试 mock。
func NewPGStore(db dbtx) *PGStore {
	return &PGStore{db: db}
}

// ListMetrics 列出设备指标;resourceID=0 返回全部,否则按设备过滤。
func (s *PGStore) ListMetrics(ctx context.Context, resourceID int64) ([]DeviceMetric, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, resource_id, optical_power, packet_loss, status, collected_at
		FROM device_metrics WHERE ($1 = 0 OR resource_id = $1) ORDER BY collected_at, id`, resourceID)
	if err != nil {
		return nil, fmt.Errorf("device: list metrics: %w", err)
	}
	defer rows.Close()
	out := make([]DeviceMetric, 0)
	for rows.Next() {
		var m DeviceMetric
		var opt, pkt pgtype.Float8
		if err := rows.Scan(&m.ID, &m.ResourceID, &opt, &pkt, &m.Status, &m.CollectedAt); err != nil {
			return nil, fmt.Errorf("device: scan metric: %w", err)
		}
		if opt.Valid {
			v := opt.Float64
			m.OpticalPower = &v
		}
		if pkt.Valid {
			v := pkt.Float64
			m.PacketLoss = &v
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// AppendMetric 追加设备指标,返回自增 id。
func (s *PGStore) AppendMetric(ctx context.Context, m DeviceMetric) (int64, error) {
	var id int64
	ts := m.CollectedAt
	if ts.IsZero() {
		ts = time.Now()
	}
	err := s.db.QueryRow(ctx, `
		INSERT INTO device_metrics(resource_id, optical_power, packet_loss, status, collected_at)
		VALUES($1,$2,$3,$4,$5) RETURNING id`,
		m.ResourceID, m.OpticalPower, m.PacketLoss, m.Status, ts).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("device: append metric: %w", err)
	}
	return id, nil
}

// ListMaintenances 列出设备健康观察。
func (s *PGStore) ListMaintenances(ctx context.Context) ([]DeviceMaintenance, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, device_no, COALESCE(device_type, ''), health_score, fault_count, age_years, COALESCE(reason, ''), priority
		FROM device_maintenances ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("device: list maintenances: %w", err)
	}
	defer rows.Close()
	out := make([]DeviceMaintenance, 0)
	for rows.Next() {
		var m DeviceMaintenance
		var age pgtype.Float8
		if err := rows.Scan(&m.ID, &m.DeviceNo, &m.DeviceType, &m.HealthScore, &m.FaultCount, &age, &m.Reason, &m.Priority); err != nil {
			return nil, fmt.Errorf("device: scan maintenance: %w", err)
		}
		if age.Valid {
			v := age.Float64
			m.AgeYears = &v
		}
		out = append(out, m)
	}
	return out, rows.Err()
}

// CreateMaintenance 新建设备健康观察,返回自增 id。
func (s *PGStore) CreateMaintenance(ctx context.Context, m DeviceMaintenance) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO device_maintenances(device_no, device_type, health_score, fault_count, age_years, reason, priority)
		VALUES($1,$2,$3,$4,$5,$6,$7) RETURNING id`,
		m.DeviceNo, m.DeviceType, m.HealthScore, m.FaultCount, m.AgeYears, m.Reason, m.Priority).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("device: create maintenance: %w", err)
	}
	return id, nil
}
