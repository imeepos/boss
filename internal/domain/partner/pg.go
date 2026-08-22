// partner PG 存储构造 + 企业工作台读路径(Profile/Staff/Orders)。
package partner

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type dbtx interface {
	Query(ctx context.Context, sql string, args ...any) (pgx.Rows, error)
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
	Exec(ctx context.Context, sql string, args ...any) (pgconn.CommandTag, error)
	Begin(ctx context.Context) (pgx.Tx, error)
}

// PGStore 入驻域 PG 实现。
type PGStore struct {
	db dbtx
}

// NewPGStore 构造 PGStore;db 传 *pgxpool.Pool。
func NewPGStore(db dbtx) *PGStore {
	return &PGStore{db: db}
}

// entityOfAccount 账号归属的 legal_entity;0=未绑定(非入驻企业成员)。
func (s *PGStore) entityOfAccount(ctx context.Context, accountID int64) (int64, error) {
	var entityID int64
	err := s.db.QueryRow(ctx,
		`SELECT COALESCE(legal_entity_id,0) FROM accounts WHERE id=$1`, accountID).Scan(&entityID)
	if err != nil {
		return 0, fmt.Errorf("partner: account entity lookup: %w", err)
	}
	return entityID, nil
}

// Profile 入驻企业档案:legal_entities × 申请回填信息。
func (s *PGStore) Profile(ctx context.Context, accountID int64) (PartnerProfile, error) {
	var p PartnerProfile
	err := s.db.QueryRow(ctx, `
SELECT le.id, le.name, pa.credit_code, pa.contact_name, pa.contact_phone,
       COALESCE(pa.email,''), pa.submitted_at, pa.reviewed_at
FROM accounts a
JOIN legal_entities le ON le.id = a.legal_entity_id
JOIN partner_applications pa ON pa.legal_entity_id = le.id
WHERE a.id = $1 AND a.legal_entity_id IS NOT NULL`, accountID).
		Scan(&p.LegalEntityID, &p.CompanyName, &p.CreditCode, &p.ContactName, &p.ContactPhone,
			&p.Email, &p.AppliedAt, &p.ApprovedAt)
	if err != nil {
		return p, fmt.Errorf("partner: profile: %w", err)
	}
	return p, nil
}

// ListStaff 本企业员工账号(partner_* 角色,id 升序)。
func (s *PGStore) ListStaff(ctx context.Context, accountID int64) ([]StaffRow, error) {
	entityID, err := s.entityOfAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if entityID == 0 {
		return nil, ErrNotPartner
	}
	rows, err := s.db.Query(ctx, `
SELECT a.id, a.username, a.real_name, COALESCE(a.phone,''), r.code, a.status, a.created_at
FROM accounts a JOIN roles r ON r.id = a.role_id
WHERE a.legal_entity_id = $1 AND r.code IN ('partner_admin','partner_staff')
ORDER BY a.id`, entityID)
	if err != nil {
		return nil, fmt.Errorf("partner: list staff: %w", err)
	}
	defer rows.Close()
	var out []StaffRow
	for rows.Next() {
		var r StaffRow
		if err := rows.Scan(&r.ID, &r.Username, &r.RealName, &r.Phone, &r.RoleCode, &r.Status, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("partner: scan staff: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}

// CreateStaff 企业管理员新建员工账号(角色固定 partner_staff,归属本企业)。
func (s *PGStore) CreateStaff(ctx context.Context, adminAccountID int64, username, password, realName, phone string) (int64, error) {
	entityID, err := s.entityOfAccount(ctx, adminAccountID)
	if err != nil {
		return 0, err
	}
	if entityID == 0 {
		return 0, ErrNotPartner
	}
	hash, err := bcryptHash(password)
	if err != nil {
		return 0, err
	}
	var id int64
	err = s.db.QueryRow(ctx, `
INSERT INTO accounts(username, password_hash, real_name, phone, role_id, legal_entity_id, status)
SELECT $1,$2,$3,$4,r.id,$5,1 FROM roles r WHERE r.code='partner_staff' RETURNING id`,
		username, hash, realName, phone, entityID).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("partner: create staff: %w", err)
	}
	return id, nil
}

// SetStaffStatus 启用/停用本企业员工(越权拒;partner_admin 自身不可停,防自锁)。
func (s *PGStore) SetStaffStatus(ctx context.Context, adminAccountID, staffID int64, status int16) error {
	entityID, err := s.entityOfAccount(ctx, adminAccountID)
	if err != nil {
		return err
	}
	if entityID == 0 {
		return ErrNotPartner
	}
	if status != 0 && status != 1 {
		return fmt.Errorf("partner: invalid status %d", status)
	}
	tag, err := s.db.Exec(ctx, `
UPDATE accounts a SET status=$1, updated_at=now()
FROM roles r
WHERE a.id=$2 AND a.role_id=r.id AND a.legal_entity_id=$3
  AND r.code IN ('partner_admin','partner_staff') AND a.id <> $4`,
		status, staffID, entityID, adminAccountID)
	if err != nil {
		return fmt.Errorf("partner: set staff status: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrStaffScope
	}
	return nil
}

// ListOrders 本企业订单(创建时间倒序,带客户名快照)。
func (s *PGStore) ListOrders(ctx context.Context, accountID int64) ([]OrderRow, error) {
	entityID, err := s.entityOfAccount(ctx, accountID)
	if err != nil {
		return nil, err
	}
	if entityID == 0 {
		return nil, ErrNotPartner
	}
	rows, err := s.db.Query(ctx, `
SELECT o.id, o.order_no, COALESCE(c.name,''), o.stage, o.status, o.created_at
FROM orders o LEFT JOIN customers c ON c.id = o.customer_id
WHERE o.legal_entity_id = $1
ORDER BY o.created_at DESC, o.id DESC LIMIT 200`, entityID)
	if err != nil {
		return nil, fmt.Errorf("partner: list orders: %w", err)
	}
	defer rows.Close()
	var out []OrderRow
	for rows.Next() {
		var r OrderRow
		if err := rows.Scan(&r.ID, &r.OrderNo, &r.CustomerName, &r.Stage, &r.Status, &r.CreatedAt); err != nil {
			return nil, fmt.Errorf("partner: scan order: %w", err)
		}
		out = append(out, r)
	}
	return out, rows.Err()
}
