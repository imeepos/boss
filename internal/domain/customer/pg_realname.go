package customer

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ListVerifications 列出实名核验记录;customerID=0 返回全部。
// 读回 real_name/id_card_no 给前端脱敏回显(合成客户没 customers 主档,nameMasked 兜底从核验单拿)。
func (s *PGStore) ListVerifications(ctx context.Context, customerID int64) ([]RealNameVerification, error) {
	rows, err := s.db.Query(ctx, `
		SELECT id, subject_id, method, COALESCE(real_name, ''), COALESCE(id_card_no, ''),
		       verified_at, result, reject_reason,
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
		if err := rows.Scan(&v.ID, &v.CustomerID, &v.Method, &v.RealName, &v.IDCardNo,
			&v.VerifiedAt, &v.Result, &v.RejectReason, &v.OperatorAccountID, &v.OperatorName); err != nil {
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

// guardRealNameIdentity PASS 一致性门禁:待核验单 id_card_no 对照 customers 主档。
// 无待核验单(直建客户,POST /customers 镜像插入,未经用户端核验单提交)→ 无对照物,跳过门禁放行;
// 主档 id_no 为空(新客补登)→ 回填 id_no/real_name_status 前置数据;非空不一致 → ErrRealNameMismatch。
// 查无主档(合成客户,负数段隔离空间)→ 无门禁可施,直接放行:PASS 只落 verifications,
// 后续 customers 状态同步为 0 行 no-op,用户端以最新核验单回显结论(profile_handlers 合成客户回退)。
func (s *PGStore) guardRealNameIdentity(ctx context.Context, customerID int64) error {
	var pendingIDNo, masterIDNo sql.NullString
	err := s.db.QueryRow(ctx, `
SELECT (SELECT v.id_card_no FROM verifications v
         WHERE v.subject_type='customer' AND v.subject_id=c.id AND v.result=$2
         ORDER BY v.verified_at DESC, v.id DESC LIMIT 1),
       COALESCE(c.id_no, '')
  FROM customers c
 WHERE c.id = $1`, customerID, RealNamePending).Scan(&pendingIDNo, &masterIDNo)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil
		}
		return fmt.Errorf("customer: verify identity guard: %w", err)
	}
	if !pendingIDNo.Valid || pendingIDNo.String == "" {
		// 无 PENDING 核验单可对照(直建客户此前 50000:NULL 直扫 *string),PASS 不做一致性拦截。
		return nil
	}
	if !masterIDNo.Valid || masterIDNo.String == "" {
		// 主档无证件号:以核验单回填,保证 PASS 后两侧一致。
		if _, err := s.db.Exec(ctx,
			`UPDATE customers SET id_no=$2 WHERE id=$1 AND COALESCE(id_no,'')=''`, customerID, pendingIDNo.String); err != nil {
			return fmt.Errorf("customer: verify backfill id_no: %w", err)
		}
		return nil
	}
	if pendingIDNo.String != masterIDNo.String {
		return ErrRealNameMismatch
	}
	return nil
}
