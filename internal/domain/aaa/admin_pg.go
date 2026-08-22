package aaa

import (
	"context"
	"fmt"
	"strings"
)

// ListLoAccountsPage 管理端按公司/区域分页查询 LO 账号。
func (s *PGStore) ListLoAccountsPage(ctx context.Context, q AdminPage, scope AdminScope) (AdminPageResult[LoAccount], error) {
	q = normalizePage(q)
	where, args := loAccountWhere(q, scope)
	count, err := s.count(ctx, "lo_accounts", where, args)
	if err != nil {
		return AdminPageResult[LoAccount]{}, err
	}
	rows, err := s.db.Query(ctx, `SELECT `+loAccountCols+` FROM lo_accounts WHERE `+where+` ORDER BY id LIMIT $`+fmt.Sprint(len(args)+1)+` OFFSET $`+fmt.Sprint(len(args)+2), append(args, q.PageSize, (q.Page-1)*q.PageSize)...)
	if err != nil {
		return AdminPageResult[LoAccount]{}, fmt.Errorf("aaa: list lo accounts page: %w", err)
	}
	defer rows.Close()
	items := make([]LoAccount, 0, q.PageSize)
	for rows.Next() {
		var a LoAccount
		if err := rows.Scan(&a.ID, &a.Loid, &a.CustomerID, &a.LegalEntityID, &a.LegalEntityName, &a.RegionID, &a.RegionName, &a.RegionPath, &a.OfferID, &a.QosTemplateID, &a.Status, &a.BillingMode); err != nil {
			return AdminPageResult[LoAccount]{}, err
		}
		items = append(items, a)
	}
	return AdminPageResult[LoAccount]{Items: items, Total: count, Page: q.Page, PageSize: q.PageSize}, rows.Err()
}

// ListCdrsPage 管理端按 LOID/公司/区域分页查询 CDR。
func (s *PGStore) ListCdrsPage(ctx context.Context, q AdminPage, scope AdminScope) (AdminPageResult[CdrRecord], error) {
	q = normalizePage(q)
	where, args := cdrWhere(q, scope)
	return s.pageCdrs(ctx, q, where, args)
}

// ListAuthLogsPage 管理端按 LOID/公司/区域分页查询认证日志。
func (s *PGStore) ListAuthLogsPage(ctx context.Context, q AdminPage, scope AdminScope) (AdminPageResult[AuthLog], error) {
	q = normalizePage(q)
	where, args := authLogWhere(q, scope)
	count, err := s.count(ctx, "auth_logs al JOIN lo_accounts la ON la.loid = al.loid", where, args)
	if err != nil {
		return AdminPageResult[AuthLog]{}, err
	}
	rows, err := s.db.Query(ctx, `SELECT al.id, al.loid, al.result, al.created_at FROM auth_logs al JOIN lo_accounts la ON la.loid = al.loid WHERE `+where+` ORDER BY al.created_at DESC, al.id DESC LIMIT $`+fmt.Sprint(len(args)+1)+` OFFSET $`+fmt.Sprint(len(args)+2), append(args, q.PageSize, (q.Page-1)*q.PageSize)...)
	if err != nil {
		return AdminPageResult[AuthLog]{}, fmt.Errorf("aaa: list auth logs page: %w", err)
	}
	defer rows.Close()
	items := make([]AuthLog, 0, q.PageSize)
	for rows.Next() {
		var l AuthLog
		if err := rows.Scan(&l.ID, &l.Loid, &l.Result, &l.CreatedAt); err != nil {
			return AdminPageResult[AuthLog]{}, err
		}
		items = append(items, l)
	}
	return AdminPageResult[AuthLog]{Items: items, Total: count, Page: q.Page, PageSize: q.PageSize}, rows.Err()
}

func (s *PGStore) pageCdrs(ctx context.Context, q AdminPage, where string, args []any) (AdminPageResult[CdrRecord], error) {
	count, err := s.count(ctx, "cdrs c JOIN lo_accounts la ON la.loid = c.loid", where, args)
	if err != nil {
		return AdminPageResult[CdrRecord]{}, err
	}
	rows, err := s.db.Query(ctx, `SELECT c.id, c.loid, COALESCE(c.username, ''), c.acct_status, COALESCE(c.session_id, ''), c.session_time, c.input_octets, c.output_octets, COALESCE(c.nas_ip, ''), c.billing_status, c.started_at FROM cdrs c JOIN lo_accounts la ON la.loid = c.loid WHERE `+where+` ORDER BY c.started_at DESC, c.id DESC LIMIT $`+fmt.Sprint(len(args)+1)+` OFFSET $`+fmt.Sprint(len(args)+2), append(args, q.PageSize, (q.Page-1)*q.PageSize)...)
	if err != nil {
		return AdminPageResult[CdrRecord]{}, fmt.Errorf("aaa: list cdrs page: %w", err)
	}
	defer rows.Close()
	items := make([]CdrRecord, 0, q.PageSize)
	for rows.Next() {
		var c CdrRecord
		if err := rows.Scan(&c.ID, &c.Loid, &c.Username, &c.AcctStatus, &c.SessionID, &c.SessionTime, &c.InputOctets, &c.OutputOctets, &c.NasIP, &c.BillingStatus, &c.StartedAt); err != nil {
			return AdminPageResult[CdrRecord]{}, err
		}
		items = append(items, c)
	}
	return AdminPageResult[CdrRecord]{Items: items, Total: count, Page: q.Page, PageSize: q.PageSize}, rows.Err()
}

func (s *PGStore) count(ctx context.Context, table, where string, args []any) (int, error) {
	var count int
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FROM `+table+` WHERE `+where, args...).Scan(&count); err != nil {
		return 0, fmt.Errorf("aaa: count: %w", err)
	}
	return count, nil
}

func normalizePage(q AdminPage) AdminPage {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}
	q.Keyword = strings.TrimSpace(q.Keyword)
	q.Status = strings.TrimSpace(q.Status)
	q.Loid = strings.TrimSpace(q.Loid)
	return q
}

func loAccountWhere(q AdminPage, scope AdminScope) (string, []any) {
	where := []string{"1=1"}
	args := make([]any, 0, 4)
	if q.Keyword != "" {
		args = append(args, "%"+q.Keyword+"%")
		where = append(where, fmt.Sprintf("(loid ILIKE $%d OR legal_entity_name ILIKE $%d)", len(args), len(args)))
	}
	if q.Status != "" {
		args = append(args, q.Status)
		where = append(where, fmt.Sprintf("status = $%d", len(args)))
	}
	if scope.LegalEntityID > 0 {
		args = append(args, scope.LegalEntityID)
		where = append(where, fmt.Sprintf("legal_entity_id = $%d", len(args)))
	}
	if scope.RegionScope != "" {
		args = append(args, scope.RegionScope)
		where = append(where, fmt.Sprintf("region_path <@ $%d::ltree", len(args)))
	}
	return strings.Join(where, " AND "), args
}

func authLogWhere(q AdminPage, scope AdminScope) (string, []any) {
	where := []string{"1=1"}
	args := make([]any, 0, 4)
	if q.Loid != "" {
		args = append(args, q.Loid)
		where = append(where, fmt.Sprintf("al.loid = $%d", len(args)))
	}
	if q.Keyword != "" {
		args = append(args, "%"+q.Keyword+"%")
		where = append(where, fmt.Sprintf("al.loid ILIKE $%d", len(args)))
	}
	if q.Status != "" {
		args = append(args, q.Status)
		where = append(where, fmt.Sprintf("al.result = $%d", len(args)))
	}
	if scope.LegalEntityID > 0 {
		args = append(args, scope.LegalEntityID)
		where = append(where, fmt.Sprintf("la.legal_entity_id = $%d", len(args)))
	}
	if scope.RegionScope != "" {
		args = append(args, scope.RegionScope)
		where = append(where, fmt.Sprintf("la.region_path <@ $%d::ltree", len(args)))
	}
	return strings.Join(where, " AND "), args
}

func cdrWhere(q AdminPage, scope AdminScope) (string, []any) {
	where := []string{"1=1"}
	args := make([]any, 0, 4)
	if q.Loid != "" {
		args = append(args, q.Loid)
		where = append(where, fmt.Sprintf("c.loid = $%d", len(args)))
	}
	if q.Keyword != "" {
		args = append(args, "%"+q.Keyword+"%")
		where = append(where, fmt.Sprintf("(c.loid ILIKE $%d OR c.session_id ILIKE $%d)", len(args), len(args)))
	}
	if q.Status != "" {
		args = append(args, q.Status)
		where = append(where, fmt.Sprintf("c.billing_status = $%d", len(args)))
	}
	if scope.LegalEntityID > 0 {
		args = append(args, scope.LegalEntityID)
		where = append(where, fmt.Sprintf("la.legal_entity_id = $%d", len(args)))
	}
	if scope.RegionScope != "" {
		args = append(args, scope.RegionScope)
		where = append(where, fmt.Sprintf("la.region_path <@ $%d::ltree", len(args)))
	}
	return strings.Join(where, " AND "), args
}
