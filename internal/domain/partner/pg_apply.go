// 入驻申请写路径:Submit/List/Approve/Reject(PG 实现)。
package partner

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strings"

	"github.com/jackc/pgx/v5"
	"golang.org/x/crypto/bcrypt"
)

const appCols = `id, company_name, credit_code, contact_name, contact_phone,
 COALESCE(email,''), business_desc, status, COALESCE(review_note,''),
 COALESCE(reviewer_account_id,0), COALESCE(legal_entity_id,0), COALESCE(admin_account_id,0),
 submitted_at, reviewed_at`

// Validate 校验申请必填项(公开端点入参,宽松但非空)。
func (a Application) Validate() error {
	for k, v := range map[string]string{
		"companyName":  a.CompanyName,
		"creditCode":   a.CreditCode,
		"contactName":  a.ContactName,
		"contactPhone": a.ContactPhone,
		"businessDesc": a.BusinessDesc,
	} {
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("partner: %s required", k)
		}
	}
	return nil
}

func scanApp(row pgx.Row) (Application, error) {
	var a Application
	err := row.Scan(&a.ID, &a.CompanyName, &a.CreditCode, &a.ContactName, &a.ContactPhone,
		&a.Email, &a.BusinessDesc, &a.Status, &a.ReviewNote,
		&a.ReviewerAccountID, &a.LegalEntityID, &a.AdminAccountID,
		&a.SubmittedAt, &a.ReviewedAt)
	return a, err
}

// Submit 新建入驻申请(落 PENDING);同信用码已有未审申请则拒,防重复排队。
func (s *PGStore) Submit(ctx context.Context, in Application) (int64, error) {
	if err := in.Validate(); err != nil {
		return 0, err
	}
	var dup bool
	err := s.db.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM partner_applications WHERE credit_code=$1 AND status='PENDING')`,
		in.CreditCode).Scan(&dup)
	if err != nil {
		return 0, fmt.Errorf("partner: submit dup check: %w", err)
	}
	if dup {
		return 0, ErrApplicationDuplicate
	}
	var id int64
	err = s.db.QueryRow(ctx, `
INSERT INTO partner_applications(company_name, credit_code, contact_name, contact_phone,
                                 email, business_desc)
VALUES($1,$2,$3,$4,NULLIF($5,''),$6) RETURNING id`,
		strings.TrimSpace(in.CompanyName), strings.TrimSpace(in.CreditCode),
		strings.TrimSpace(in.ContactName), strings.TrimSpace(in.ContactPhone),
		strings.TrimSpace(in.Email), strings.TrimSpace(in.BusinessDesc)).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("partner: submit insert: %w", err)
	}
	return id, nil
}

// ListApplications 按状态(空=全部)列出申请,提交时间倒序。
func (s *PGStore) ListApplications(ctx context.Context, status string) ([]Application, error) {
	q := `SELECT ` + appCols + ` FROM partner_applications`
	args := []any{}
	if status != "" {
		q += ` WHERE status=$1`
		args = append(args, status)
	}
	q += ` ORDER BY submitted_at DESC, id DESC`
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("partner: list applications: %w", err)
	}
	defer rows.Close()
	var out []Application
	for rows.Next() {
		a, err := scanApp(rows)
		if err != nil {
			return nil, fmt.Errorf("partner: scan application: %w", err)
		}
		out = append(out, a)
	}
	return out, rows.Err()
}

// bcryptHash 口令哈希(集中一处,成本与 user 域一致)。
func bcryptHash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("partner: bcrypt: %w", err)
	}
	return string(hash), nil
}

// genPassword 随机初始口令:12 位,字母+数字,去掉易混字符。
const pwdAlphabet = "23456789ABCDEFGHJKMNPQRSTUVWXYZabcdefghjkmnpqrstuvwxyz"

func genPassword() (string, error) {
	n := big.NewInt(int64(len(pwdAlphabet)))
	buf := make([]byte, 12)
	for i := range buf {
		idx, err := rand.Int(rand.Reader, n)
		if err != nil {
			return "", err
		}
		buf[i] = pwdAlphabet[idx.Int64()]
	}
	return string(buf), nil
}

// Approve 审核通过:PENDING→APPROVED,建子公司 + partner_admin 账号并回填。
// 子公司 code 取 P-<信用码后8位>;账号 username 取 pt_<手机号>。
func (s *PGStore) Approve(ctx context.Context, id, reviewerAccountID int64) (ApproveResult, error) {
	var in Application
	var err error
	if in, err = scanApp(s.db.QueryRow(ctx,
		`SELECT `+appCols+` FROM partner_applications WHERE id=$1 AND status=$2`, id, StatusPending)); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ApproveResult{}, ErrApplicationConflict
		}
		return ApproveResult{}, fmt.Errorf("partner: approve read: %w", err)
	}
	password, err := genPassword()
	if err != nil {
		return ApproveResult{}, fmt.Errorf("partner: gen password: %w", err)
	}
	hash, err := bcryptHash(password)
	if err != nil {
		return ApproveResult{}, err
	}
	username := "pt_" + in.ContactPhone
	if len(in.CreditCode) > 8 {
		username = "pt_" + in.CreditCode[len(in.CreditCode)-8:]
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return ApproveResult{}, fmt.Errorf("partner: begin: %w", err)
	}
	defer tx.Rollback(ctx)

	var entityID int64
	err = tx.QueryRow(ctx, `
INSERT INTO legal_entities(code, name) VALUES($1,$2) RETURNING id`,
		"P-"+in.CreditCode, in.CompanyName).Scan(&entityID)
	if err != nil {
		return ApproveResult{}, fmt.Errorf("partner: create legal entity: %w", err)
	}
	var adminID int64
	err = tx.QueryRow(ctx, `
INSERT INTO accounts(username, password_hash, real_name, phone, role_id, legal_entity_id, status)
SELECT $1,$2,$3,$4,r.id,$5,1 FROM roles r WHERE r.code='partner_admin' RETURNING id`,
		username, string(hash), in.ContactName, in.ContactPhone, entityID).Scan(&adminID)
	if err != nil {
		return ApproveResult{}, fmt.Errorf("partner: create admin account: %w", err)
	}
	_, err = tx.Exec(ctx, `
UPDATE partner_applications SET status=$1, reviewer_account_id=$2,
 legal_entity_id=$3, admin_account_id=$4, reviewed_at=now()
 WHERE id=$5 AND status=$6`,
		StatusApproved, reviewerAccountID, entityID, adminID, id, StatusPending)
	if err != nil {
		return ApproveResult{}, fmt.Errorf("partner: approve update: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return ApproveResult{}, fmt.Errorf("partner: commit: %w", err)
	}
	return ApproveResult{
		ApplicationID: id, LegalEntityID: entityID, AdminAccountID: adminID,
		Username: username, InitialPassword: password,
	}, nil
}

// Reject 审核驳回:记审核意见(仅作用于 PENDING)。
func (s *PGStore) Reject(ctx context.Context, id, reviewerAccountID int64, note string) error {
	if strings.TrimSpace(note) == "" {
		return fmt.Errorf("partner: reject note required")
	}
	tag, err := s.db.Exec(ctx, `
UPDATE partner_applications SET status=$1, reviewer_account_id=$2, review_note=$3, reviewed_at=now()
 WHERE id=$4 AND status=$5`,
		StatusRejected, reviewerAccountID, note, id, StatusPending)
	if err != nil {
		return fmt.Errorf("partner: reject: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrApplicationConflict
	}
	return nil
}
