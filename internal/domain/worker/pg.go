package worker

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// ErrNotFound 记录不存在。
var ErrNotFound = errors.New("worker: not found")

// ErrForeignKeyViolation 关联实体不存在(孤儿数据防护)。
var ErrForeignKeyViolation = errors.New("worker: foreign key violation")

// dbtx 是 PGStore 依赖的最小数据库接口;*pgxpool.Pool 天然满足,单测用 pgxmock 注入。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// PGStore 是 WorkerService 接口的 PostgreSQL 实现(阶段2)。
type PGStore struct {
	db dbtx
}

// NewPGStore 构造 PGStore;db 传 *pgxpool.Pool 或测试 mock。
func NewPGStore(db dbtx) *PGStore {
	return &PGStore{db: db}
}

// exists 校验单表存在性(workers 无外键约束,关联完整性由本域应用层保证)。
func (s *PGStore) exists(ctx context.Context, table string, id int64) (bool, error) {
	var ok bool
	err := s.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM `+table+` WHERE id = $1)`, id).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("worker: check %s %d: %w", table, id, err)
	}
	return ok, nil
}

// idOrNil 把 0 归一为 NULL(可空约定:0=空)。
func idOrNil(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

// ListGroups 列出全部班组。
func (s *PGStore) ListGroups(ctx context.Context) ([]Group, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, legal_entity_id, code, name, COALESCE(leader_id, 0), COALESCE(leader_name, '') FROM worker_groups ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("worker: list groups: %w", err)
	}
	defer rows.Close()
	out := make([]Group, 0)
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.ID, &g.LegalEntityID, &g.Code, &g.Name, &g.LeaderID, &g.LeaderName); err != nil {
			return nil, fmt.Errorf("worker: scan group: %w", err)
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// CreateGroup 新建班组,返回自增 id。
func (s *PGStore) CreateGroup(ctx context.Context, g Group) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO worker_groups(legal_entity_id, code, name, leader_id, leader_name)
		VALUES($1,$2,$3,$4,$5) RETURNING id`,
		g.LegalEntityID, g.Code, g.Name, idOrNil(g.LeaderID), g.LeaderName).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("worker: create group: %w", err)
	}
	return id, nil
}

const workerCols = `id, staff_no, name, group_id, region_id, phone, status, joined_at, left_at`

// ListWorkers 列出师傅;groupID=0 返回全部,否则按班组过滤。
func (s *PGStore) ListWorkers(ctx context.Context, groupID int64) ([]Worker, error) {
	rows, err := s.db.Query(ctx,
		`SELECT `+workerCols+` FROM workers WHERE ($1 = 0 OR group_id = $1) ORDER BY id`, groupID)
	if err != nil {
		return nil, fmt.Errorf("worker: list workers: %w", err)
	}
	defer rows.Close()
	out := make([]Worker, 0)
	for rows.Next() {
		w, err := scanWorker(rows)
		if err != nil {
			return nil, err
		}
		out = append(out, *w)
	}
	return out, rows.Err()
}

// CreateWorker 新建师傅,返回自增 id。
func (s *PGStore) CreateWorker(ctx context.Context, w Worker) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO workers(staff_no, name, group_id, region_id, phone, status, joined_at, left_at)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8) RETURNING id`,
		w.StaffNo, w.Name, w.GroupID, w.RegionID, w.Phone, w.Status, w.JoinedAt, w.LeftAt).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("worker: create worker: %w", err)
	}
	return id, nil
}

// GetWorker 按 id 查师傅;未命中返回 ErrNotFound。
func (s *PGStore) GetWorker(ctx context.Context, id int64) (*Worker, error) {
	w, err := scanWorker(s.db.QueryRow(ctx, `SELECT `+workerCols+` FROM workers WHERE id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("worker: get worker: %w", err)
	}
	return w, nil
}

// rowScanner 抽象 pgx.Row 与 pgx.Rows 的 Scan。
type rowScanner interface {
	Scan(dest ...any) error
}

func scanWorker(r rowScanner) (*Worker, error) {
	var w Worker
	var leftAt pgtype.Timestamptz
	if err := r.Scan(&w.ID, &w.StaffNo, &w.Name, &w.GroupID, &w.RegionID, &w.Phone, &w.Status, &w.JoinedAt, &leftAt); err != nil {
		return nil, err
	}
	if leftAt.Valid {
		t := leftAt.Time
		w.LeftAt = &t
	}
	return &w, nil
}
