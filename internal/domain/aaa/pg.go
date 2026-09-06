package aaa

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// ErrForeignKeyViolation 关联实体不存在(孤儿数据防护:lo_accounts 无外键约束)。
var ErrForeignKeyViolation = errors.New("aaa: referenced entity not found")

// dbtx 是 PGStore 依赖的最小数据库接口;*pgxpool.Pool 天然满足,单测用 pgxmock 注入。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// exists 校验单表存在性(lo_accounts 无外键,关联完整性由本域应用层保证)。
func (s *PGStore) exists(ctx context.Context, table string, id int64) (bool, error) {
	var ok bool
	err := s.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM `+table+` WHERE id = $1)`, id).Scan(&ok)
	if err != nil {
		return false, fmt.Errorf("aaa: check %s %d: %w", table, id, err)
	}
	return ok, nil
}

// PGStore 是 AaaService 接口的 PostgreSQL 实现(阶段7:LO账号/话单/认证日志)。
type PGStore struct {
	db dbtx
}

// NewPGStore 构造 PGStore;db 传 *pgxpool.Pool 或测试 mock。
func NewPGStore(db dbtx) *PGStore {
	return &PGStore{db: db}
}

const loAccountCols = `id, loid, customer_id, legal_entity_id, legal_entity_name, region_id, region_name, COALESCE(region_path,''), offer_id, qos_template_id, status, billing_mode`

// ListLoAccounts 列出全部 LO 账号。
func (s *PGStore) ListLoAccounts(ctx context.Context) ([]LoAccount, error) {
	rows, err := s.db.Query(ctx, `SELECT `+loAccountCols+` FROM lo_accounts ORDER BY id`)
	if err != nil {
		return nil, fmt.Errorf("aaa: list lo_accounts: %w", err)
	}
	defer rows.Close()
	out := make([]LoAccount, 0)
	for rows.Next() {
		var a LoAccount
		if err := rows.Scan(&a.ID, &a.Loid, &a.CustomerID, &a.LegalEntityID, &a.LegalEntityName, &a.RegionID, &a.RegionName, &a.RegionPath, &a.OfferID, &a.QosTemplateID, &a.Status, &a.BillingMode); err != nil {
			return nil, fmt.Errorf("aaa: scan lo_account: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// CreateLoAccount 新建 LO 账号,返回自增 id;BillingMode 空时继承该客户最近订单的
// 付费模式(fields.md §3.1,只读跨表与 GenerateBills 同惯例),仍空回退 POSTPAID。
// 关联完整性:customer_id/offer_id/legal_entity_id 为 NOT NULL 软引用,缺失直接拒,
// 防止孤儿 LO 账号(customer_id 曾 18 条孤儿,audit 2026-08-25)。
func (s *PGStore) CreateLoAccount(ctx context.Context, a LoAccount) (int64, error) {
	if a.CustomerID <= 0 {
		return 0, fmt.Errorf("aaa: customer_id required: %w", ErrForeignKeyViolation)
	}
	for _, ref := range []struct {
		table string
		id    int64
		label string
	}{
		{"customers", a.CustomerID, "customer"},
		{"product_offers", a.OfferID, "offer"},
		{"legal_entities", a.LegalEntityID, "legal entity"},
	} {
		if ref.id <= 0 {
			return 0, fmt.Errorf("aaa: %s_id required: %w", ref.label, ErrForeignKeyViolation)
		}
		ok, err := s.exists(ctx, ref.table, ref.id)
		if err != nil {
			return 0, err
		}
		if !ok {
			return 0, fmt.Errorf("aaa: %s %d: %w", ref.label, ref.id, ErrForeignKeyViolation)
		}
	}
	if a.BillingMode == "" {
		a.BillingMode = s.inheritBillingMode(ctx, a.CustomerID)
	}
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO lo_accounts(loid, customer_id, legal_entity_id, legal_entity_name, region_id, region_name, region_path, offer_id, qos_template_id, status, billing_mode)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`,
		a.Loid, a.CustomerID, a.LegalEntityID, a.LegalEntityName, a.RegionID, a.RegionName, nilIfEmpty(a.RegionPath), a.OfferID, a.QosTemplateID, a.Status, a.BillingMode).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("aaa: create lo_account: %w", err)
	}
	return id, nil
}

// inheritBillingMode 读客户最近订单的付费模式;无订单/查询失败回退 POSTPAID。
func (s *PGStore) inheritBillingMode(ctx context.Context, customerID int64) string {
	var mode string
	err := s.db.QueryRow(ctx,
		`SELECT billing_mode FROM orders WHERE customer_id = $1 ORDER BY id DESC LIMIT 1`,
		customerID).Scan(&mode)
	if err != nil || (mode != BillingModePrepaid && mode != BillingModePostpaid) {
		return BillingModePostpaid
	}
	return mode
}

// nilIfEmpty 空串归 NULL(region_path 可空)。
func nilIfEmpty(s string) any {
	if s == "" {
		return nil
	}
	return s
}

// GetLoAccountByLoid 按 LOID 查账号;未命中返回 ErrNotFound。
func (s *PGStore) GetLoAccountByLoid(ctx context.Context, loid string) (*LoAccount, error) {
	var a LoAccount
	err := s.db.QueryRow(ctx, `SELECT `+loAccountCols+` FROM lo_accounts WHERE loid = $1`, loid).
		Scan(&a.ID, &a.Loid, &a.CustomerID, &a.LegalEntityID, &a.LegalEntityName, &a.RegionID, &a.RegionName, &a.RegionPath, &a.OfferID, &a.QosTemplateID, &a.Status, &a.BillingMode)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("aaa: get lo_account: %w", err)
	}
	return &a, nil
}

// GetLoAccountByCustomer 按客户查 LO 账号(客户 1:1);未命中返回 ErrNotFound。
func (s *PGStore) GetLoAccountByCustomer(ctx context.Context, customerID int64) (*LoAccount, error) {
	var a LoAccount
	err := s.db.QueryRow(ctx, `SELECT `+loAccountCols+` FROM lo_accounts WHERE customer_id = $1`, customerID).
		Scan(&a.ID, &a.Loid, &a.CustomerID, &a.LegalEntityID, &a.LegalEntityName, &a.RegionID, &a.RegionName, &a.RegionPath, &a.OfferID, &a.QosTemplateID, &a.Status, &a.BillingMode)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("aaa: get lo_account by customer: %w", err)
	}
	return &a, nil
}

// AlignLoAccountOffer 把已有 LO 账号的生效套餐/计费模式对齐到变更单套餐(改套餐,TMF change order 语义)。
// 值未变化时 0 行 no-op 返回 false;RADIUS 授权实时 JOIN lo_accounts,对齐即下次认证生效新档。
func (s *PGStore) AlignLoAccountOffer(ctx context.Context, customerID, offerID int64, billingMode string) (bool, error) {
	tag, err := s.db.Exec(ctx, `
		UPDATE lo_accounts SET offer_id = $2, billing_mode = $3
		WHERE customer_id = $1
		  AND (offer_id IS DISTINCT FROM $2 OR COALESCE(billing_mode,'POSTPAID') IS DISTINCT FROM COALESCE(NULLIF($3,''),'POSTPAID'))`,
		customerID, offerID, billingMode)
	if err != nil {
		return false, fmt.Errorf("aaa: align lo_account offer: %w", err)
	}
	return tag.RowsAffected() > 0, nil
}

const cdrCols = `id, loid, COALESCE(username, ''), acct_status, COALESCE(session_id, ''), session_time, input_octets, output_octets, COALESCE(nas_ip, ''), billing_status, started_at`

// AppendCdr 追加话单,返回自增 id。
func (s *PGStore) AppendCdr(ctx context.Context, c CdrRecord) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO cdrs(loid, username, acct_status, session_id, session_time, input_octets, output_octets, nas_ip, billing_status)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`,
		c.Loid, c.Username, c.AcctStatus, c.SessionID, c.SessionTime, c.InputOctets, c.OutputOctets, c.NasIP, c.BillingStatus).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("aaa: append cdr: %w", err)
	}
	return id, nil
}

// ListCdrs 列出话单;loid 为空返回全部,否则按账号过滤。
func (s *PGStore) ListCdrs(ctx context.Context, loid string) ([]CdrRecord, error) {
	rows, err := s.db.Query(ctx, `
		SELECT `+cdrCols+` FROM cdrs WHERE ($1 = '' OR loid = $1) ORDER BY started_at, id`, loid)
	if err != nil {
		return nil, fmt.Errorf("aaa: list cdrs: %w", err)
	}
	defer rows.Close()
	out := make([]CdrRecord, 0)
	for rows.Next() {
		var c CdrRecord
		if err := rows.Scan(&c.ID, &c.Loid, &c.Username, &c.AcctStatus, &c.SessionID, &c.SessionTime, &c.InputOctets, &c.OutputOctets, &c.NasIP, &c.BillingStatus, &c.StartedAt); err != nil {
			return nil, fmt.Errorf("aaa: scan cdr: %w", err)
		}
		out = append(out, c)
	}
	return out, rows.Err()
}

// AppendAuthLog 追加认证日志,返回自增 id。
func (s *PGStore) AppendAuthLog(ctx context.Context, l AuthLog) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx,
		`INSERT INTO auth_logs(loid, result, fail_reason) VALUES($1,$2,$3) RETURNING id`, l.Loid, l.Result, l.FailReason).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("aaa: append auth log: %w", err)
	}
	return id, nil
}

// ListAuthLogs 列出认证日志;loid 为空返回全部,否则按账号过滤。
func (s *PGStore) ListAuthLogs(ctx context.Context, loid string) ([]AuthLog, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, loid, result, fail_reason, created_at FROM auth_logs WHERE ($1 = '' OR loid = $1) ORDER BY created_at, id`, loid)
	if err != nil {
		return nil, fmt.Errorf("aaa: list auth logs: %w", err)
	}
	defer rows.Close()
	out := make([]AuthLog, 0)
	for rows.Next() {
		var l AuthLog
		if err := rows.Scan(&l.ID, &l.Loid, &l.Result, &l.FailReason, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("aaa: scan auth log: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
