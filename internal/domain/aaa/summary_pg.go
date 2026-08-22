package aaa

import (
	"context"
	"fmt"
)

// GetAdminSummary 在数据库侧聚合 AAA 总览，避免页面拉取三份全量数据。
func (s *PGStore) GetAdminSummary(ctx context.Context, scope AdminScope) (AdminSummary, error) {
	loWhere, loArgs := loAccountWhere(AdminPage{}, scope)
	cdrWhereSQL, cdrArgs := cdrWhere(AdminPage{}, scope)
	authWhereSQL, authArgs := authLogWhere(AdminPage{}, scope)
	var out AdminSummary
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*), COUNT(*) FILTER (WHERE status='ACTIVE'), COUNT(*) FILTER (WHERE status='SUSPENDED'), COUNT(*) FILTER (WHERE status='CLOSED') FROM lo_accounts WHERE `+loWhere, loArgs...).Scan(&out.Accounts, &out.Active, &out.Suspended, &out.Closed); err != nil {
		return out, fmt.Errorf("aaa: summary accounts: %w", err)
	}
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*), COUNT(*) FILTER (WHERE c.billing_status='UNBILLED') FROM cdrs c JOIN lo_accounts la ON la.loid=c.loid WHERE `+cdrWhereSQL, cdrArgs...).Scan(&out.Cdrs, &out.Unbilled); err != nil {
		return out, fmt.Errorf("aaa: summary cdrs: %w", err)
	}
	if err := s.db.QueryRow(ctx, `SELECT COUNT(*) FILTER (WHERE al.result='SUCCESS'), COUNT(*) FILTER (WHERE al.result='FAILED') FROM auth_logs al JOIN lo_accounts la ON la.loid=al.loid WHERE `+authWhereSQL, authArgs...).Scan(&out.AuthSuccess, &out.AuthFailed); err != nil {
		return out, fmt.Errorf("aaa: summary auth: %w", err)
	}
	return out, nil
}
