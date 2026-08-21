// 数据备份迁移域(SYS 域运维工具,迁移 000095):按表导出 gzip JSONL 归档,
// 恢复 = 逐行 INSERT ... ON CONFLICT DO NOTHING 追加导入(不删不改既有数据)。
// 设计裁定见 docs/notes/adopted/2026-08-21-backup-local-jsonl.md;字段口径 docs/contract/fields.md 1.5.6。
package backup

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"sync"

	"github.com/jackc/pgx/v5/pgxpool"
)

// ErrInvalidInput 入参不合法(表名不在 public 候选集内等)。
var ErrInvalidInput = errors.New("backup: invalid input")

// 任务类型/状态枚举。
const (
	KindBackup       = "backup"
	KindRestore      = "restore"
	ScopeAll         = "all"
	ScopeTables      = "tables"
	StatusRunning    = "running"
	StatusSucceeded  = "succeeded"
	StatusFailed     = "failed"
	defaultBackupDir = "data/backups"
)

// Job 一次备份/恢复任务的记录行(页面列名对齐 fields.md 1.5.6)。
type Job struct {
	ID         int64    `json:"id"`
	Kind       string   `json:"kind"`
	Scope      string   `json:"scope"`
	Tables     []string `json:"tables"`
	Status     string   `json:"status"`
	FileName   string   `json:"fileName"`
	SizeBytes  int64    `json:"sizeBytes"`
	TableCount int      `json:"tableCount"`
	RowCount   int64    `json:"rowCount"`
	Error      string   `json:"error,omitempty"`
	Operator   string   `json:"operator"`
	CreatedAt  string   `json:"createdAt"`
	FinishedAt string   `json:"finishedAt,omitempty"`
}

// ListFilter 任务清单过滤(kind/status 可空 = 不过滤)。
type ListFilter struct {
	Kind   string
	Status string
	Limit  int
	Offset int
}

// Service 备份迁移服务:任务簿记 + 进程内串行异步执行(同一时刻至多一条任务在跑)。
type Service struct {
	store   *PGStore
	dir     string
	mu      sync.Mutex
	running bool
}

// NewService 构造服务;dir 空 = 默认 data/backups(相对服务进程工作目录)。
func NewService(pool *pgxpool.Pool, dir string) (*Service, error) {
	if dir == "" {
		dir = defaultBackupDir
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, err
	}
	return &Service{store: NewPGStore(pool), dir: dir}, nil
}

// Dir 归档目录(绝对化,便于日志排查)。
func (s *Service) Dir() string {
	abs, err := filepath.Abs(s.dir)
	if err != nil {
		return s.dir
	}
	return abs
}

// tryLock 抢占执行槽;已有任务在跑返回 false。
func (s *Service) tryLock() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.running {
		return false
	}
	s.running = true
	return true
}

func (s *Service) unlock() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.running = false
}

// CreateBackup 创建并异步执行备份任务;tables 空 = public 全表(scope=all)。
func (s *Service) CreateBackup(ctx context.Context, operatorID int64, tables []string) (int64, error) {
	candidates, err := s.store.ListTables(ctx)
	if err != nil {
		return 0, err
	}
	valid := make(map[string]bool, len(candidates))
	for _, t := range candidates {
		valid[t] = true
	}
	scope := ScopeTables
	if len(tables) == 0 {
		scope = ScopeAll
		tables = candidates
	} else {
		for _, t := range tables {
			if !valid[t] {
				return 0, ErrInvalidInput
			}
		}
	}
	if !s.tryLock() {
		return 0, ErrBusy
	}
	id, err := s.store.CreateJob(ctx, KindBackup, scope, tables, operatorID)
	if err != nil {
		s.unlock()
		return 0, err
	}
	go s.runBackup(id, tables)
	return id, nil
}

// CreateRestore 创建并异步执行恢复任务;srcPath 为已落盘的临时上传文件(移入归档目录)。
func (s *Service) CreateRestore(ctx context.Context, operatorID int64, fileName, srcPath string) (int64, error) {
	if !s.tryLock() {
		return 0, ErrBusy
	}
	dest := s.jobFilePath(fileName)
	if err := os.Rename(srcPath, dest); err != nil {
		s.unlock()
		return 0, err
	}
	id, err := s.store.CreateJob(ctx, KindRestore, ScopeTables, nil, operatorID)
	if err != nil {
		s.unlock()
		os.Remove(dest)
		return 0, err
	}
	s.store.SetFile(ctx, id, fileName)
	go s.runRestore(id, dest)
	return id, nil
}

// jobFilePath 归档文件绝对路径(文件名由调用方保证不含路径分隔符)。
func (s *Service) jobFilePath(fileName string) string {
	return filepath.Join(s.dir, fileName)
}

// ListTables 候选表清单(public 普通表,排除系统表)。
func (s *Service) ListTables(ctx context.Context) ([]string, error) {
	return s.store.ListTables(ctx)
}

// List 任务清单。
func (s *Service) List(ctx context.Context, f ListFilter) ([]Job, int, error) {
	return s.store.ListJobs(ctx, f)
}

// Get 任务详情。
func (s *Service) Get(ctx context.Context, id int64) (*Job, error) {
	return s.store.GetJob(ctx, id)
}

// Delete 删除任务行;有归档文件时一并清理磁盘。
func (s *Service) Delete(ctx context.Context, id int64) error {
	fileName, err := s.store.DeleteJob(ctx, id)
	if err != nil {
		return err
	}
	if fileName != "" {
		if rmErr := os.Remove(s.jobFilePath(fileName)); rmErr != nil && !os.IsNotExist(rmErr) {
			return rmErr
		}
	}
	return nil
}
