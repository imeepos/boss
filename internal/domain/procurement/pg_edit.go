package procurement

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// SupplierUpdate 供应商编辑入参(PUT):指针 nil = 保持原值(部分更新语义);
// 编码不在入参中,结构性不可改;禁用态同样可改资料(WHERE 不带 status 条件),
// 状态流转归 disable/enable 端点专管。
type SupplierUpdate struct {
	Name           *string `json:"name"`
	ContactName    *string `json:"contactName"`
	ContactPhone   *string `json:"contactPhone"`
	Remark         *string `json:"remark"`
	ContractorType *string `json:"contractorType"`
	Qualification  *string `json:"qualification"`
}

// OrderDraftUpdate 采购单草稿编辑入参(PUT):Remark/ExpectedDate nil = 保持原值;
// Items nil = 保持原明细,非 nil(含空数组)= 整体替换并重算总金额。
type OrderDraftUpdate struct {
	Remark       *string     `json:"remark"`
	ExpectedDate *time.Time  `json:"expectedDate"`
	Items        []OrderItem `json:"items"`
}

// UpdateSupplier 编辑供应商(A):名称/联系人/电话/备注可改,编码不可改,禁用态可改资料。
func (s *PGStore) UpdateSupplier(ctx context.Context, id int64, in SupplierUpdate) error {
	if in.Name != nil && *in.Name == "" {
		return fmt.Errorf("procurement: supplier %d: %w", id, ErrInvalidInput)
	}
	if in.ContractorType != nil {
		ct, err := normalizeContractorType(*in.ContractorType)
		if err != nil {
			return fmt.Errorf("procurement: supplier %d: %w", id, err)
		}
		in.ContractorType = &ct
	}
	if in.Qualification != nil {
		if err := validateQualification(*in.Qualification); err != nil {
			return fmt.Errorf("procurement: supplier %d: %w", id, err)
		}
	}
	tag, err := s.db.Exec(ctx,
		`UPDATE procurement_suppliers
		 SET name = COALESCE($2, name), contact_name = COALESCE($3, contact_name),
		     contact_phone = COALESCE($4, contact_phone), remark = COALESCE($5, remark),
		     contractor_type = COALESCE($6, contractor_type), qualification = COALESCE($7, qualification),
		     updated_at = now()
		 WHERE id = $1`,
		id, in.Name, in.ContactName, in.ContactPhone, in.Remark, in.ContractorType, in.Qualification)
	if err != nil {
		return fmt.Errorf("procurement: update supplier: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrNotFound
	}
	return nil
}

// EnableSupplier 启用供应商(B):DISABLED to ENABLED;已 ENABLED 幂等成功;
// 不存在返回 ErrNotFound(40400),与其他状态撞车返回 ErrStateConflict(40900)。
func (s *PGStore) EnableSupplier(ctx context.Context, id int64) error {
	var status string
	err := s.db.QueryRow(ctx,
		`SELECT status FROM procurement_suppliers WHERE id=$1`, id).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("procurement: lookup supplier: %w", err)
	}
	if status == "ENABLED" {
		return nil // 幂等:重复启用原样成功
	}
	tag, err := s.db.Exec(ctx,
		`UPDATE procurement_suppliers SET status='ENABLED', updated_at=now()
		 WHERE id=$1 AND status='DISABLED'`, id)
	if err != nil {
		return fmt.Errorf("procurement: enable supplier: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("procurement: supplier %d status=%s: %w", id, status, ErrStateConflict)
	}
	return nil
}

// UpdateOrderDraft 编辑采购单草稿(C):仅 DRAFT 可改(否则 ErrStateConflict,HTTP 40900);
// 备注/期望日期可改,Items 非 nil 整体替换(行 quantity 必须>0,违者 ErrInvalidInput)
// 并重算总金额;同事务 FOR UPDATE 锁单头,防编辑与提交/取消并发竞态。
func (s *PGStore) UpdateOrderDraft(ctx context.Context, id int64, in OrderDraftUpdate) error {
	for _, it := range in.Items {
		if it.Quantity <= 0 {
			return fmt.Errorf("procurement: item %s quantity=%d: %w", it.MaterialCode, it.Quantity, ErrInvalidInput)
		}
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("procurement: begin tx: %w", err)
	}
	defer tx.Rollback(ctx)

	var status string
	err = tx.QueryRow(ctx,
		`SELECT status FROM procurement_orders WHERE id=$1 FOR UPDATE`, id).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("procurement: lock order: %w", err)
	}
	if status != "DRAFT" {
		return fmt.Errorf("procurement: order %d status=%s: %w", id, status, ErrStateConflict)
	}

	// 总金额口径:明细替换按新明细 quantity*unit_amount 求和;未替换按库内明细重算(自愈)。
	total := 0.0
	if in.Items != nil {
		for _, it := range in.Items {
			total += float64(it.Quantity) * it.UnitAmount
		}
		if _, err = tx.Exec(ctx,
			`DELETE FROM procurement_order_items WHERE order_id=$1`, id); err != nil {
			return fmt.Errorf("procurement: clear items: %w", err)
		}
		for _, it := range in.Items {
			if _, err = tx.Exec(ctx,
				`INSERT INTO procurement_order_items(order_id, material_code, spec, quantity, unit_amount, remark)
				 VALUES($1,$2,$3,$4,$5,$6)`,
				id, it.MaterialCode, it.Spec, it.Quantity, it.UnitAmount, it.Remark); err != nil {
				return fmt.Errorf("procurement: insert item: %w", err)
			}
		}
	} else {
		err = tx.QueryRow(ctx,
			`SELECT COALESCE(SUM(quantity * unit_amount),0) FROM procurement_order_items WHERE order_id=$1`, id).Scan(&total)
		if err != nil {
			return fmt.Errorf("procurement: sum items: %w", err)
		}
	}

	if _, err = tx.Exec(ctx,
		`UPDATE procurement_orders
		 SET remark = COALESCE($2, remark), expected_date = COALESCE($3, expected_date),
		     total_amount = $4, updated_at = now()
		 WHERE id=$1`,
		id, in.Remark, in.ExpectedDate, total); err != nil {
		return fmt.Errorf("procurement: update order draft: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("procurement: commit: %w", err)
	}
	return nil
}

// GetSupplier 查供应商(含承建类型维度;未命中 nil,不报错)。
func (s *PGStore) GetSupplier(ctx context.Context, id int64) (*Supplier, error) {
	var sup Supplier
	err := s.db.QueryRow(ctx,
		`SELECT id, code, name, COALESCE(contact_name,''), COALESCE(contact_phone,''),
		       legal_entity_id, status, COALESCE(remark,''), created_at, updated_at,
		       contractor_type, COALESCE(qualification,'')
		 FROM procurement_suppliers WHERE id=$1`, id).Scan(
		&sup.ID, &sup.Code, &sup.Name, &sup.ContactName, &sup.ContactPhone,
		&sup.LegalEntityID, &sup.Status, &sup.Remark, &sup.CreatedAt, &sup.UpdatedAt,
		&sup.ContractorType, &sup.Qualification)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("procurement: get supplier: %w", err)
	}
	return &sup, nil
}

// normalizeContractorType 承建类型归一:空值=材料类(存量语义),其余必须为枚举值。
func normalizeContractorType(s string) (string, error) {
	switch s {
	case "":
		return ContractorMaterial, nil
	case ContractorMaterial, ContractorConstruction:
		return s, nil
	default:
		return "", fmt.Errorf("contractor_type %q: %w", s, ErrInvalidInput)
	}
}

// validateQualification 资质信息长度守卫(与列 VARCHAR(255) 对齐,超长走 42200 而非库错误)。
func validateQualification(q string) error {
	if len(q) > 255 {
		return fmt.Errorf("qualification %d bytes: %w", len(q), ErrInvalidInput)
	}
	return nil
}
