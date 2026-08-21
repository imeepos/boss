package push

// devices PostgreSQL 实现(迁移 000095 push_devices)。

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// DevicesPGStore DevicesService 的 PostgreSQL 实现。
type DevicesPGStore struct{ db devicesDB }

// devicesDB pgx 最小事务接口。
type devicesDB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// NewDevicesPGStore 构造 PG 设备注册表。
func NewDevicesPGStore(db devicesDB) *DevicesPGStore { return &DevicesPGStore{db: db} }

// RegisterDevice 幂等注册 + 改绑:同 RegistrationID 冲突时更新主体/厂商/活跃时间。
func (s *DevicesPGStore) RegisterDevice(ctx context.Context, subjectType string, subjectID int64, registrationID, vendor string) error {
	if !validDevice(subjectType, subjectID, registrationID) {
		return ErrInvalidDevice
	}
	_, err := s.db.Exec(ctx, `
		INSERT INTO push_devices (subject_type, subject_id, registration_id, vendor, last_active_at)
		VALUES ($1, $2, $3, $4, now())
		ON CONFLICT (registration_id) DO UPDATE SET
			subject_type = EXCLUDED.subject_type,
			subject_id = EXCLUDED.subject_id,
			vendor = EXCLUDED.vendor,
			last_active_at = now()`,
		subjectType, subjectID, registrationID, vendor)
	if err != nil {
		return fmt.Errorf("push: register device: %w", err)
	}
	return nil
}

// RegistrationIDs 主体绑定的全部设备。
func (s *DevicesPGStore) RegistrationIDs(ctx context.Context, subjectType string, subjectID int64) ([]string, error) {
	if subjectType != SubjectUser && subjectType != SubjectWorker || subjectID <= 0 {
		return nil, ErrInvalidDevice
	}
	rows, err := s.db.Query(ctx,
		`SELECT registration_id FROM push_devices WHERE subject_type = $1 AND subject_id = $2`,
		subjectType, subjectID)
	if err != nil {
		return nil, fmt.Errorf("push: list devices: %w", err)
	}
	defer rows.Close()
	out := []string{}
	for rows.Next() {
		var id string
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("push: scan device: %w", err)
		}
		out = append(out, id)
	}
	return out, rows.Err()
}
