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

// ListGroups 列出全部在职班组(装维队;软删排除,附在职成员数)。
func (s *PGStore) ListGroups(ctx context.Context) ([]Group, error) {
	rows, err := s.db.Query(ctx, `
		SELECT g.id, g.legal_entity_id, g.code, g.name, COALESCE(g.leader_id, 0), COALESCE(g.leader_name, ''),
		       (SELECT COUNT(*) FROM workers w WHERE w.group_id = g.id AND w.status = 1)
		FROM worker_groups g WHERE g.deleted_at IS NULL ORDER BY g.id`)
	if err != nil {
		return nil, fmt.Errorf("worker: list groups: %w", err)
	}
	defer rows.Close()
	out := make([]Group, 0)
	for rows.Next() {
		var g Group
		if err := rows.Scan(&g.ID, &g.LegalEntityID, &g.Code, &g.Name, &g.LeaderID, &g.LeaderName, &g.MemberCount); err != nil {
			return nil, fmt.Errorf("worker: scan group: %w", err)
		}
		out = append(out, g)
	}
	return out, rows.Err()
}

// CreateGroup 新建班组,返回自增 id。
// 校验 legal_entity_id 存在性,防止孤儿班组。
func (s *PGStore) CreateGroup(ctx context.Context, g Group) (int64, error) {
	// 关联完整性校验
	if g.LegalEntityID > 0 {
		ok, err := s.exists(ctx, "legal_entities", g.LegalEntityID)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("worker: legal entity %d: %w", g.LegalEntityID, ErrForeignKeyViolation)
		}
	}

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

// workerFrom 读路径统一 FROM:LEFT JOIN 班组/区域取人读名现值(§6.2,列名随 JOIN 全限定)。
const workerFrom = `workers LEFT JOIN worker_groups wg ON wg.id = workers.group_id LEFT JOIN regions r ON r.id = workers.region_id`

const workerCols = `workers.id, workers.staff_no, workers.name, workers.group_id, COALESCE(wg.name, '') AS group_name, workers.region_id, COALESCE(r.name, '') AS region_name, workers.phone, workers.status, workers.joined_at, workers.left_at`

// ListWorkers 列出师傅;groupID=0 返回全部,否则按班组过滤;keyword 命中姓名/工号/手机号。
// 附带负责区域集合(迁移 000175;批量一查询回填,避免 N+1)。
func (s *PGStore) ListWorkers(ctx context.Context, groupID int64, keyword string) ([]Worker, error) {
	rows, err := s.db.Query(ctx,
		`SELECT `+workerCols+` FROM `+workerFrom+`
		WHERE ($1 = 0 OR workers.group_id = $1)
		  AND ($2 = '' OR workers.name ILIKE '%' || $2 || '%' OR workers.staff_no ILIKE '%' || $2 || '%' OR workers.phone ILIKE '%' || $2 || '%')
		ORDER BY workers.id`, groupID, keyword)
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
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("worker: list workers rows: %w", err)
	}
	ptrs := make([]*Worker, len(out))
	for i := range out {
		ptrs[i] = &out[i]
	}
	if err := s.attachRegionIDs(ctx, ptrs); err != nil {
		return nil, err
	}
	return out, nil
}

// CreateWorker 新建师傅(无登录密码),返回自增 id。
// 由 CreateWorkerWithPassword(ctx, w, "") 承接,详见 pg_password.go。
func (s *PGStore) CreateWorker(ctx context.Context, w Worker) (int64, error) {
	return s.CreateWorkerWithPassword(ctx, w, "")
}

// GetWorker 按 id 查师傅;未命中返回 ErrNotFound。附带负责区域集合(迁移 000175)。
func (s *PGStore) GetWorker(ctx context.Context, id int64) (*Worker, error) {
	w, err := scanWorker(s.db.QueryRow(ctx, `SELECT `+workerCols+` FROM `+workerFrom+` WHERE workers.id = $1`, id))
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("worker: get worker: %w", err)
	}
	if err := s.attachRegionIDs(ctx, []*Worker{w}); err != nil {
		return nil, err
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
	if err := r.Scan(&w.ID, &w.StaffNo, &w.Name, &w.GroupID, &w.GroupName, &w.RegionID, &w.RegionName, &w.Phone, &w.Status, &w.JoinedAt, &leftAt); err != nil {
		return nil, err
	}
	if leftAt.Valid {
		t := leftAt.Time
		w.LeftAt = &t
	}
	return &w, nil
}
