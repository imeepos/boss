package aaa

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// dbtx 是 PGStore 依赖的最小数据库接口;*pgxpool.Pool 天然满足,单测用 pgxmock 注入。
type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
}

// PGStore 是 AaaService 接口的 PostgreSQL 实现(阶段7:LO账号/话单/认证日志)。
type PGStore struct {
	db dbtx
}

// NewPGStore 构造 PGStore;db 传 *pgxpool.Pool 或测试 mock。
func NewPGStore(db dbtx) *PGStore {
	return &PGStore{db: db}
}

const loAccountCols = `id, loid, customer_id, legal_entity_id, legal_entity_name, region_id, region_name, offer_id, qos_template_id, status`

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
		if err := rows.Scan(&a.ID, &a.Loid, &a.CustomerID, &a.LegalEntityID, &a.LegalEntityName, &a.RegionID, &a.RegionName, &a.OfferID, &a.QosTemplateID, &a.Status); err != nil {
			return nil, fmt.Errorf("aaa: scan lo_account: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// CreateLoAccount 新建 LO 账号,返回自增 id。
func (s *PGStore) CreateLoAccount(ctx context.Context, a LoAccount) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO lo_accounts(loid, customer_id, legal_entity_id, legal_entity_name, region_id, region_name, offer_id, qos_template_id, status)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9) RETURNING id`,
		a.Loid, a.CustomerID, a.LegalEntityID, a.LegalEntityName, a.RegionID, a.RegionName, a.OfferID, a.QosTemplateID, a.Status).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("aaa: create lo_account: %w", err)
	}
	return id, nil
}

// GetLoAccountByLoid 按 LOID 查账号;未命中返回 ErrNotFound。
func (s *PGStore) GetLoAccountByLoid(ctx context.Context, loid string) (*LoAccount, error) {
	var a LoAccount
	err := s.db.QueryRow(ctx, `SELECT `+loAccountCols+` FROM lo_accounts WHERE loid = $1`, loid).
		Scan(&a.ID, &a.Loid, &a.CustomerID, &a.LegalEntityID, &a.LegalEntityName, &a.RegionID, &a.RegionName, &a.OfferID, &a.QosTemplateID, &a.Status)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("aaa: get lo_account: %w", err)
	}
	return &a, nil
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
		`INSERT INTO auth_logs(loid, result) VALUES($1,$2) RETURNING id`, l.Loid, l.Result).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("aaa: append auth log: %w", err)
	}
	return id, nil
}

// ListAuthLogs 列出认证日志;loid 为空返回全部,否则按账号过滤。
func (s *PGStore) ListAuthLogs(ctx context.Context, loid string) ([]AuthLog, error) {
	rows, err := s.db.Query(ctx,
		`SELECT id, loid, result, created_at FROM auth_logs WHERE ($1 = '' OR loid = $1) ORDER BY created_at, id`, loid)
	if err != nil {
		return nil, fmt.Errorf("aaa: list auth logs: %w", err)
	}
	defer rows.Close()
	out := make([]AuthLog, 0)
	for rows.Next() {
		var l AuthLog
		if err := rows.Scan(&l.ID, &l.Loid, &l.Result, &l.CreatedAt); err != nil {
			return nil, fmt.Errorf("aaa: scan auth log: %w", err)
		}
		out = append(out, l)
	}
	return out, rows.Err()
}
