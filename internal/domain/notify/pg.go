// notify 域 PostgreSQL 实现(迁移 000090 两表)。
package notify

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrNotFound 记录不存在。
var ErrNotFound = errors.New("notify: not found")

type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// scanner pgx.Rows/pgx.Row 的最小扫描接口。
type scanner interface{ Scan(dest ...any) error }

// PGStore 是 Service 的 PostgreSQL 实现。
type PGStore struct{ db dbtx }

// NewPGStore 构造 PGStore。
func NewPGStore(db dbtx) *PGStore { return &PGStore{db: db} }

const itemCols = `n.id, n.category, n.level, n.title, n.content, n.link,
	n.ref_type, n.ref_id, n.resolved, n.created_at,
	(r.notification_id IS NOT NULL) AS read`

// Emit 幂等写入:同 (ref_type, ref_id, category) 已存在时跳过(INSERT...SELECT...WHERE NOT EXISTS)。
func (s *PGStore) Emit(ctx context.Context, in Input) error {
	if !in.Valid() {
		return ErrInvalidInput
	}
	in = in.NormalizeDefaults()
	tag, err := s.db.Exec(ctx, `
		INSERT INTO admin_notifications
			(category, level, title, content, link, ref_type, ref_id, target_role)
		SELECT $1, $2, $3, $4, $5, $6, $7, $8
		WHERE NOT EXISTS (
			SELECT 1 FROM admin_notifications
			WHERE ref_type = $6 AND ref_id = $7 AND category = $1
		)`,
		in.Category, in.Level, in.Title, in.Content, in.Link, in.RefType, in.RefID, in.TargetRole)
	if err != nil {
		return fmt.Errorf("notify: emit: %w", err)
	}
	_ = tag
	return nil
}

// Resolve 把来源域指定的未办条目置 resolved(幂等)。
func (s *PGStore) Resolve(ctx context.Context, refType, refID string) error {
	if refType == "" || refID == "" {
		return ErrInvalidInput
	}
	_, err := s.db.Exec(ctx, `
		UPDATE admin_notifications SET resolved = TRUE
		WHERE ref_type = $1 AND ref_id = $2 AND NOT resolved`, refType, refID)
	if err != nil {
		return fmt.Errorf("notify: resolve: %w", err)
	}
	return nil
}

// List 角色可见清单(新→旧)+ 账号读状态;返回条目+过滤后总数。
func (s *PGStore) List(ctx context.Context, role string, accountID int64, f Filter) ([]Item, int, error) {
	where, args := listWhere(role, accountID, f)
	total, err := s.count(ctx, where, args)
	if err != nil {
		return nil, 0, err
	}
	limit, offset := clampLimit(f.Limit), max0(f.Offset)
	rows, err := s.db.Query(ctx, `
		SELECT `+itemCols+`
		FROM admin_notifications n
		LEFT JOIN admin_notification_reads r
			ON r.notification_id = n.id AND r.account_id = $2
		WHERE `+where+`
		ORDER BY n.id DESC
		LIMIT `+fmt.Sprint(limit)+` OFFSET `+fmt.Sprint(offset), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("notify: list: %w", err)
	}
	defer rows.Close()
	items := []Item{}
	for rows.Next() {
		it, err := scanItem(rows)
		if err != nil {
			return nil, 0, err
		}
		items = append(items, it)
	}
	return items, total, rows.Err()
}

// UnreadCount 可见未读数(resolved 未读不计,前端默认隐藏已办)。
func (s *PGStore) UnreadCount(ctx context.Context, role string, accountID int64) (int, error) {
	where, args := listWhere(role, accountID, Filter{HideResolved: true})
	return s.count(ctx, where+" AND r.notification_id IS NULL", args)
}

// MarkRead 批量已读;ids 为空=当前角色可见全部已读;越权 id 静默忽略。
func (s *PGStore) MarkRead(ctx context.Context, role string, accountID int64, ids []int64) error {
	where, args := listWhere(role, accountID, Filter{})
	sql := `
		INSERT INTO admin_notification_reads (notification_id, account_id)
		SELECT n.id, $2 FROM admin_notifications n
		LEFT JOIN admin_notification_reads r
			ON r.notification_id = n.id AND r.account_id = $2
		WHERE ` + where + ` AND r.notification_id IS NULL`
	if len(ids) > 0 {
		sql = `
		INSERT INTO admin_notification_reads (notification_id, account_id)
		SELECT n.id, $2 FROM admin_notifications n
		WHERE n.id = ANY($3) AND (` + where + `)`
		args = append(args, ids)
	}
	_, err := s.db.Exec(ctx, sql+" ON CONFLICT DO NOTHING", args...)
	if err != nil {
		return fmt.Errorf("notify: mark read: %w", err)
	}
	return nil
}

func (s *PGStore) count(ctx context.Context, where string, args []any) (int, error) {
	var n int
	err := s.db.QueryRow(ctx, `
		SELECT COUNT(*) FROM admin_notifications n
		LEFT JOIN admin_notification_reads r
			ON r.notification_id = n.id AND r.account_id = $2
		WHERE `+where, args...).Scan(&n)
	if err != nil {
		return 0, fmt.Errorf("notify: count: %w", err)
	}
	return n, nil
}

// listWhere 组装过滤条件;args[0]=role,args[1]=accountID,后续按需追加。
func listWhere(role string, accountID int64, f Filter) (string, []any) {
	args := []any{role, accountID}
	w := `(n.target_role = '' OR n.target_role = $1)`
	if f.Category != "" {
		args = append(args, f.Category)
		w += fmt.Sprintf(" AND n.category = $%d", len(args))
	}
	if f.Level != "" {
		args = append(args, f.Level)
		w += fmt.Sprintf(" AND n.level = $%d", len(args))
	}
	if f.Unread {
		w += " AND r.notification_id IS NULL"
	}
	if f.HideResolved {
		w += " AND NOT n.resolved"
	}
	return w, args
}

func scanItem(row scanner) (Item, error) {
	var it Item
	var created time.Time
	if err := row.Scan(&it.ID, &it.Category, &it.Level, &it.Title, &it.Content,
		&it.Link, &it.RefType, &it.RefID, &it.Resolved, &created, &it.Read); err != nil {
		return it, fmt.Errorf("notify: scan: %w", err)
	}
	it.CreatedAt = created.UTC().Format(time.RFC3339)
	return it, nil
}

func clampLimit(n int) int {
	if n <= 0 || n > 500 {
		return 50
	}
	return n
}

func max0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}
