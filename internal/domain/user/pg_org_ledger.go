package user

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgtype"
)

// idOrNil 把 0 归一为 NULL(可空外键约定:0=空)。
func idOrNil(id int64) any {
	if id == 0 {
		return nil
	}
	return id
}

// ListAccountOrgHistories 列出账号组织归属台账;accountID=0 返回全部。
func (s *PGStore) ListAccountOrgHistories(ctx context.Context, accountID int64) ([]AccountOrgHistory, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, account_id,
		       COALESCE(legal_entity_id, 0), COALESCE(legal_entity_name, ''),
		       COALESCE(dept_id, 0), COALESCE(dept_name, ''),
		       COALESCE(post_id, 0), COALESCE(post_name, ''),
		       COALESCE(reason, ''), COALESCE(operator_account_id, 0),
		       effective_from, effective_to
		FROM account_org_histories WHERE ($1 = 0 OR account_id = $1) ORDER BY effective_from, id`, accountID)
	if err != nil {
		return nil, fmt.Errorf("user: list org histories: %w", err)
	}
	defer rows.Close()
	out := make([]AccountOrgHistory, 0)
	for rows.Next() {
		var h AccountOrgHistory
		var effTo pgtype.Timestamptz
		if err := rows.Scan(&h.ID, &h.AccountID,
			&h.LegalEntityID, &h.LegalEntityName, &h.DeptID, &h.DeptName, &h.PostID, &h.PostName,
			&h.Reason, &h.OperatorAccountID, &h.EffectiveFrom, &effTo); err != nil {
			return nil, fmt.Errorf("user: scan org history: %w", err)
		}
		if effTo.Valid {
			t := effTo.Time
			h.EffectiveTo = &t
		}
		out = append(out, h)
	}
	return out, rows.Err()
}

// AppendAccountOrgHistory 追加账号组织归属台账,返回自增 id。
func (s *PGStore) AppendAccountOrgHistory(ctx context.Context, h AccountOrgHistory) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO account_org_histories(account_id, legal_entity_id, legal_entity_name, dept_id, dept_name,
			post_id, post_name, reason, operator_account_id, effective_from, effective_to)
		VALUES($1,$2,$3,$4,$5,$6,$7,$8,$9,$10,$11) RETURNING id`,
		h.AccountID, idOrNil(h.LegalEntityID), h.LegalEntityName, idOrNil(h.DeptID), h.DeptName,
		idOrNil(h.PostID), h.PostName, h.Reason, idOrNil(h.OperatorAccountID), h.EffectiveFrom, h.EffectiveTo).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("user: append org history: %w", err)
	}
	return id, nil
}
