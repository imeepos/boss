package customer

import (
	"context"
	"fmt"
)

// ListVerifications 列出实名核验记录;customerID=0 返回全部。
func (s *PGStore) ListVerifications(ctx context.Context, customerID int64) ([]RealNameVerification, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, subject_id, method, verified_at, result,
		       COALESCE(operator_account_id, 0), COALESCE(operator_name, '')
		FROM verifications
		WHERE subject_type = 'customer' AND ($1::bigint = 0 OR subject_id = $1)
		ORDER BY verified_at, id`, customerID)
	if err != nil {
		return nil, fmt.Errorf("customer: list verifications: %w", err)
	}
	defer rows.Close()
	out := make([]RealNameVerification, 0)
	for rows.Next() {
		var v RealNameVerification
		if err := rows.Scan(&v.ID, &v.CustomerID, &v.Method, &v.VerifiedAt, &v.Result,
			&v.OperatorAccountID, &v.OperatorName); err != nil {
			return nil, fmt.Errorf("customer: scan verification: %w", err)
		}
		out = append(out, v)
	}
	return out, rows.Err()
}

// AppendVerification 追加实名核验记录,返回自增 id。
func (s *PGStore) AppendVerification(ctx context.Context, v RealNameVerification) (int64, error) {
	var id int64
	err := s.db.QueryRow(ctx, `
		INSERT INTO verifications(subject_type, subject_id, method, verified_at, result, operator_account_id, operator_name)
		VALUES('customer',$1,$2,$3,$4,$5,$6) RETURNING id`,
		v.CustomerID, v.Method, v.VerifiedAt, v.Result, idOrNil(v.OperatorAccountID), v.OperatorName).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("customer: append verification: %w", err)
	}
	return id, nil
}
