package aaa

import (
	"context"
	"fmt"
	"strings"
)

// ListSessionsPage 管理端在线会话分页(loid/nasIp/status 过滤;JOIN lo_accounts 承接数据范围)。
func (s *PGStore) ListSessionsPage(ctx context.Context, q SessionPage, scope AdminScope) (AdminPageResult[SessionRecord], error) {
	q = normalizeSessionPage(q)
	where, args := sessionWhere(q, scope)
	count, err := s.count(ctx, "aaa_online_sessions s JOIN lo_accounts la ON la.loid = s.loid", where, args)
	if err != nil {
		return AdminPageResult[SessionRecord]{}, err
	}
	rowsSQL := fmt.Sprintf("SELECT s.%s FROM aaa_online_sessions s JOIN lo_accounts la ON la.loid = s.loid WHERE %s ORDER BY s.started_at DESC, s.id DESC LIMIT $%d OFFSET $%d",
		strings.ReplaceAll(sessionCols, ", ", ", s."), where, len(args)+1, len(args)+2)
	rows, err := s.db.Query(ctx, rowsSQL, append(args, q.PageSize, (q.Page-1)*q.PageSize)...)
	if err != nil {
		return AdminPageResult[SessionRecord]{}, fmt.Errorf("aaa: list sessions page: %w", err)
	}
	defer rows.Close()
	items := make([]SessionRecord, 0, q.PageSize)
	for rows.Next() {
		r, err := scanSession(rows.Scan)
		if err != nil {
			return AdminPageResult[SessionRecord]{}, fmt.Errorf("aaa: scan session page: %w", err)
		}
		items = append(items, r)
	}
	return AdminPageResult[SessionRecord]{Items: items, Total: count, Page: q.Page, PageSize: q.PageSize}, rows.Err()
}

// normalizeSessionPage 分页参数兜底与裁剪。
func normalizeSessionPage(q SessionPage) SessionPage {
	if q.Page < 1 {
		q.Page = 1
	}
	if q.PageSize < 1 || q.PageSize > 100 {
		q.PageSize = 20
	}
	q.Loid = strings.TrimSpace(q.Loid)
	q.NasIP = strings.TrimSpace(q.NasIP)
	q.Status = strings.TrimSpace(q.Status)
	return q
}

// sessionWhere 过滤条件构建(占位符序与 args 对齐)。
func sessionWhere(q SessionPage, scope AdminScope) (string, []any) {
	where := []string{"1=1"}
	args := make([]any, 0, 5)
	if q.Loid != "" {
		args = append(args, q.Loid)
		where = append(where, fmt.Sprintf("s.loid = $%d", len(args)))
	}
	if q.NasIP != "" {
		args = append(args, q.NasIP)
		where = append(where, fmt.Sprintf("s.nas_ip = $%d", len(args)))
	}
	if q.Status != "" {
		args = append(args, q.Status)
		where = append(where, fmt.Sprintf("s.status = $%d", len(args)))
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
