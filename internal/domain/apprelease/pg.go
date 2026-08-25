package apprelease

import (
	"context"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type pgDB interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// PGStore PostgreSQL 实现。
type PGStore struct{ db pgDB }

// NewPGStore 构造。
func NewPGStore(db pgDB) *PGStore { return &PGStore{db: db} }

const cols = `id, app, platform, version, version_code, min_supported_code, notes, force,
	status, rollout_percent, whitelist_ids, apk_object_key, apk_size, sha256, created_at, updated_at`

func scanRelease(row pgx.Row) (*Release, error) {
	var r Release
	if err := row.Scan(&r.ID, &r.App, &r.Platform, &r.Version, &r.VersionCode,
		&r.MinSupportedCode, &r.Notes, &r.Force, &r.Status, &r.RolloutPercent,
		&r.WhitelistIDs, &r.ApkObjectKey, &r.ApkSize, &r.Sha256, &r.CreatedAt, &r.UpdatedAt); err != nil {
		return nil, err
	}
	return &r, nil
}

// Create 落库;version_code 唯一键冲突归一为 ErrDuplicateVersion。
func (s *PGStore) Create(ctx context.Context, r *Release) error {
	if err := r.validate(); err != nil {
		return err
	}
	now := time.Now().Format(time.RFC3339)
	if r.WhitelistIDs == nil {
		r.WhitelistIDs = []int64{} // pgx 对 nil 切片编码 NULL,撞 NOT NULL 约束
	}
	return s.db.QueryRow(ctx, `
		INSERT INTO client_releases (app, platform, version, version_code, min_supported_code,
			notes, force, status, rollout_percent, whitelist_ids, apk_object_key, apk_size, sha256, created_at, updated_at)
		VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11,$12,$13,$14,$14)
		RETURNING id, created_at`,
		r.App, r.Platform, r.Version, r.VersionCode, r.MinSupportedCode,
		r.Notes, r.Force, r.Status, r.RolloutPercent, r.WhitelistIDs,
		r.ApkObjectKey, r.ApkSize, r.Sha256, now).Scan(&r.ID, &r.CreatedAt)
}

// Get 按 id 查。
func (s *PGStore) Get(ctx context.Context, id int64) (*Release, error) {
	row := s.db.QueryRow(ctx, `SELECT `+cols+` FROM client_releases WHERE id = $1`, id)
	r, err := scanRelease(row)
	if err != nil {
		if err == pgx.ErrNoRows {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("apprelease: get: %w", err)
	}
	return r, nil
}

// List 按 app 过滤(id 新→旧)。
func (s *PGStore) List(ctx context.Context, app string) ([]Release, error) {
	q := `SELECT ` + cols + ` FROM client_releases`
	var args []any
	if app != "" {
		q += ` WHERE app = $1`
		args = append(args, app)
	}
	q += ` ORDER BY id DESC LIMIT 200`
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("apprelease: list: %w", err)
	}
	defer rows.Close()
	var out []Release
	for rows.Next() {
		var r Release
		if err := rows.Scan(&r.ID, &r.App, &r.Platform, &r.Version, &r.VersionCode,
			&r.MinSupportedCode, &r.Notes, &r.Force, &r.Status, &r.RolloutPercent,
			&r.WhitelistIDs, &r.ApkObjectKey, &r.ApkSize, &r.Sha256, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("apprelease: scan: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// Update 全量更新。
func (s *PGStore) Update(ctx context.Context, r *Release) error {
	if err := r.validate(); err != nil {
		return err
	}
	if r.WhitelistIDs == nil {
		r.WhitelistIDs = []int64{} // 同 Create:nil 切片编码 NULL 撞约束
	}
	tag, err := s.db.Exec(ctx, `
		UPDATE client_releases SET
			version=$2, version_code=$3, min_supported_code=$4, notes=$5, force=$6,
			status=$7, rollout_percent=$8, whitelist_ids=$9, updated_at=$10
		WHERE id=$1`,
		r.ID, r.Version, r.VersionCode, r.MinSupportedCode, r.Notes, r.Force,
		r.Status, r.RolloutPercent, r.WhitelistIDs, time.Now().Format(time.RFC3339))
	if err != nil {
		return fmt.Errorf("apprelease: update: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// Actives 取投放候选:GRAY 与 PUBLISHED 的最高 version_code 各一行。
// LATERAL 写法各取一次 top-1,无候选返回空切片(客户端回落"已是最新")。
func (s *PGStore) Actives(ctx context.Context, app, platform string) ([]Release, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+cols+` FROM client_releases c
		WHERE c.app=$1 AND c.platform=$2 AND c.status IN ('GRAY','PUBLISHED')
			AND c.version_code = (
				SELECT max(c2.version_code) FROM client_releases c2
				WHERE c2.app=c.app AND c2.platform=c.platform AND c2.status=c.status)
		ORDER BY c.version_code DESC`,
		app, platform)
	if err != nil {
		return nil, fmt.Errorf("apprelease: actives: %w", err)
	}
	defer rows.Close()
	var out []Release
	for rows.Next() {
		var r Release
		if err := rows.Scan(&r.ID, &r.App, &r.Platform, &r.Version, &r.VersionCode,
			&r.MinSupportedCode, &r.Notes, &r.Force, &r.Status, &r.RolloutPercent,
			&r.WhitelistIDs, &r.ApkObjectKey, &r.ApkSize, &r.Sha256, &r.CreatedAt, &r.UpdatedAt); err != nil {
			return nil, fmt.Errorf("apprelease: scan: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
