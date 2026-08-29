// 开单内联建址(meeting-minutes/2026-08-29 §九):单事务全链补建,零阻塞开单。
package user

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
)

// 归属推导与 order 域 resolveOwnership(internal/domain/order/pg.go)同口径:
// 地址 region_id → 沿 regions 路径取最近覆盖祖先(000076),全空兜底平台总公司(000077)。
const ownershipCoverageSQL = `
	SELECT cov.legal_entity_id, cov.path::text, a.region_id
	FROM addresses a
	LEFT JOIN LATERAL (
		SELECT r.legal_entity_id, r.path
		FROM regions r
		WHERE r.path <@ (SELECT path FROM regions WHERE id = a.region_id)
		  AND r.legal_entity_id IS NOT NULL
		ORDER BY r.path DESC
		LIMIT 1
	) cov ON TRUE
	WHERE a.id = $1`

// CreateInlineAddressChain 开单内联建址:自上而下逐级 lookup-miss-then-create,
// 缺失层级就地补建到楼栋级;新节点继承最近祖先 region_id;
// backfillCustomer=true 时同事务回填客户档案装机地址。
func (s *PGStore) CreateInlineAddressChain(ctx context.Context, in InlineAddressInput) (InlineAddressResult, error) {
	if err := validateInlineInput(in); err != nil {
		return InlineAddressResult{}, err
	}
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return InlineAddressResult{}, fmt.Errorf("user: inline chain begin: %w", err)
	}
	defer tx.Rollback(ctx)
	res, err := buildAddressChain(ctx, tx, in)
	if err != nil {
		return InlineAddressResult{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return InlineAddressResult{}, fmt.Errorf("user: inline chain commit: %w", err)
	}
	return res, nil
}

// buildAddressChain 五级走链:命中即复用,缺失即补建;返回楼栋定位与治理回执。
func buildAddressChain(ctx context.Context, tx pgx.Tx, in InlineAddressInput) (InlineAddressResult, error) {
	if err := ensureCustomerExists(ctx, tx, in.CustomerID); err != nil {
		return InlineAddressResult{}, err
	}
	names := chainNames(in)
	var (
		parentID, parentPath = int64(0), ""
		created              []InlineAddressNode
	)
	for i, name := range names {
		level := int8(i + 1)
		id, path, found, err := findChainNode(ctx, tx, name, parentID)
		if err != nil {
			return InlineAddressResult{}, err
		}
		if !found {
			if id, path, err = insertChainNode(ctx, tx, name, level, parentID, parentPath); err != nil {
				return InlineAddressResult{}, err
			}
			created = append(created, InlineAddressNode{ID: id, Level: level, Name: name})
		}
		parentID, parentPath = id, path
	}
	review := make([]InlineAddressNode, 0, len(created))
	for _, n := range created {
		if needsReviewCap(n.Level) {
			review = append(review, n)
		}
	}
	entityID, regionPath, regionID, fallback, err := resolveChainOwnership(ctx, tx, parentID)
	if err != nil {
		return InlineAddressResult{}, err
	}
	backfilled, err := backfillCustomerAddress(ctx, tx, in, parentID, entityID, regionID)
	if err != nil {
		return InlineAddressResult{}, err
	}
	return InlineAddressResult{
		AddressID: parentID, FullPath: parentPath,
		FullPathNames: strings.Join(names, " / "),
		LegalEntityID: entityID, RegionPath: regionPath, Fallback: fallback,
		NeedsReview: review, Backfilled: backfilled,
	}, nil
}

// findChainNode 同父同名命中即复用(UNIQUE(path) 权威);同名校验取最小 id 保确定序。
func findChainNode(ctx context.Context, tx pgx.Tx, name string, parentID int64) (int64, string, bool, error) {
	var id int64
	var path string
	err := tx.QueryRow(ctx,
		`SELECT id, path::text FROM addresses
		 WHERE name = $1 AND parent_id IS NOT DISTINCT FROM NULLIF($2,0)
		 ORDER BY id LIMIT 1`, name, parentID).Scan(&id, &path)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", false, nil
	}
	if err != nil {
		return 0, "", false, fmt.Errorf("user: inline lookup %q: %w", name, err)
	}
	return id, path, true, nil
}

// insertChainNode 建单级节点:label 冲突(23505)加稳定后缀重试;
// region 继承最近祖先(父链全空为 NULL,归属推导层兜底);1-3 级强制 needs_review。
func insertChainNode(ctx context.Context, tx pgx.Tx, name string, level int8, parentID int64, parentPath string) (int64, string, error) {
	region, err := nearestAncestorRegion(ctx, tx, parentPath)
	if err != nil {
		return 0, "", err
	}
	base := addrLabelFromName(name)
	label := base
	for attempt := 0; attempt < 4; attempt++ {
		path := joinPath(parentPath, label)
		var id int64
		err := tx.QueryRow(ctx, `
			INSERT INTO addresses(path, level, name, parent_id, region_id, needs_review, source)
			VALUES($1::ltree, $2, $3, NULLIF($4,0), $5, $6, $7)
			RETURNING id`,
			path, level, name, parentID, region, needsReviewCap(level), inlineSource).Scan(&id)
		if isPgCode(err, "23505") {
			label = fmt.Sprintf("%s_%d", base, attempt+2)
			continue
		}
		if err != nil {
			return 0, "", fmt.Errorf("user: inline insert level %d: %w", level, err)
		}
		return id, path, nil
	}
	return 0, "", fmt.Errorf("user: inline insert %s: %w", base, ErrDuplicate)
}

// nearestAncestorRegion 沿父路径取最近挂区域祖先(含父自身);parentPath 空(根)或无覆盖返回 nil。
func nearestAncestorRegion(ctx context.Context, tx pgx.Tx, parentPath string) (*int64, error) {
	if parentPath == "" {
		return nil, nil
	}
	var rid int64
	err := tx.QueryRow(ctx,
		`SELECT region_id FROM addresses
		 WHERE path @> $1::ltree AND region_id IS NOT NULL
		 ORDER BY level DESC LIMIT 1`, parentPath).Scan(&rid)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("user: inline region inherit: %w", err)
	}
	return &rid, nil
}

// resolveChainOwnership 楼栋归属推导:区域覆盖命中即返回;未覆盖兜底平台总公司(fallback=true)。
func resolveChainOwnership(ctx context.Context, tx pgx.Tx, buildingID int64) (int64, string, *int64, bool, error) {
	var (
		entityID   sql.NullInt64
		regionPath sql.NullString
		regionID   sql.NullInt64
	)
	err := tx.QueryRow(ctx, ownershipCoverageSQL, buildingID).Scan(&entityID, &regionPath, &regionID)
	if err != nil {
		return 0, "", nil, false, fmt.Errorf("user: inline ownership: %w", err)
	}
	if entityID.Valid {
		rid := regionID.Int64
		var regionPtr *int64
		if regionID.Valid {
			regionPtr = &rid
		}
		return entityID.Int64, regionPath.String, regionPtr, false, nil
	}
	var platformID int64
	err = tx.QueryRow(ctx,
		`SELECT id FROM legal_entities WHERE is_platform LIMIT 1`).Scan(&platformID)
	if errors.Is(err, pgx.ErrNoRows) {
		return 0, "", nil, false, ErrPlatformMissing
	}
	if err != nil {
		return 0, "", nil, false, fmt.Errorf("user: inline platform fallback: %w", err)
	}
	return platformID, "root", nil, true, nil
}

// backfillCustomerAddress 显式回填客户档案:address_id 覆盖为新楼栋并同步归属快照;
// 兜底态无区域(regionID=nil),客户 region_id NOT NULL,COALESCE 保留原值。
func backfillCustomerAddress(ctx context.Context, tx pgx.Tx, in InlineAddressInput, buildingID, entityID int64, regionID *int64) (bool, error) {
	if !in.BackfillCustomer {
		return false, nil
	}
	tag, err := tx.Exec(ctx, `
		UPDATE customers
		SET address_id = $2, legal_entity_id = $3, region_id = COALESCE($4, region_id)
		WHERE id = $1`, in.CustomerID, buildingID, entityID, regionID)
	if err != nil {
		return false, fmt.Errorf("user: inline backfill customer %d: %w", in.CustomerID, err)
	}
	return tag.RowsAffected() > 0, nil
}

// ensureCustomerExists 客户必须已存在(开单环节前置),缺失即拒。
func ensureCustomerExists(ctx context.Context, tx pgx.Tx, customerID int64) error {
	var exists bool
	if err := tx.QueryRow(ctx,
		`SELECT EXISTS(SELECT 1 FROM customers WHERE id = $1)`, customerID).Scan(&exists); err != nil {
		return fmt.Errorf("user: inline customer check %d: %w", customerID, err)
	}
	if !exists {
		return fmt.Errorf("user: customer %d: %w", customerID, ErrNotFound)
	}
	return nil
}
