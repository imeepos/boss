package quadlink

// W5 写侧:扫码绑定校验(强制扫码)、拆机必扫码、四码对账任务。
// 契约:terms.md §2 扫码 result = MATCH/MISMATCH/OFFLINE_CACHED;预绑定与现场扫码必须一致。

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// ErrScanMismatch 扫码与预绑定不一致(拒绝推进,换机/重绑流程)。
var ErrScanMismatch = errors.New("quadlink: scan mismatch")

// ErrScanRequired 拆机必须扫码(不扫码拦截)。
var ErrScanRequired = errors.New("quadlink: scan required")

// ErrNotPrebound 订单尚未预绑定四码(applyTag 未执行)。
var ErrNotPrebound = errors.New("quadlink: not prebound")

// ScanReq 扫码绑定请求(装维现场)。
type ScanReq struct {
	OrderID     int64
	WorkerID    int64
	WorkerName  string
	ScannedEPC  string // 实物电子标签 EPC
	OfflineCalc bool   // 离线缓存补传
}

// ReconcileReport 四码对账结果。
type ReconcileReport struct {
	Total    int `json:"total"`
	Linked   int `json:"linked"`
	Conflict int `json:"conflict"`
	Unlinked int `json:"unlinked"`
	Purged   int `json:"purged"` // 本轮清理的孤儿行数
}

// VerifyScan 扫码绑定(环节9 强制):实物 EPC ↔ 预绑定资产核对。
// MATCH → 四码置 LINKED + 写 scan_logs;MISMATCH → 写日志并返回 ErrScanMismatch(调用方拒绝推进)。
func (s *PGStore) VerifyScan(ctx context.Context, req ScanReq) (string, error) {
	link, err := s.linkByOrder(ctx, req.OrderID)
	if err != nil {
		return "", err
	}
	var tagID, assetID int64
	err = s.db.QueryRow(ctx, `SELECT id, COALESCE(bound_asset_id, 0) FROM tags WHERE epc_code = $1`, req.ScannedEPC).
		Scan(&tagID, &assetID)
	if errors.Is(err, pgx.ErrNoRows) || (err == nil && assetID == 0) {
		return "", fmt.Errorf("%w: epc %q 未绑定资产", ErrScanMismatch, req.ScannedEPC)
	}
	if err != nil {
		return "", fmt.Errorf("quadlink: scan tag: %w", err)
	}
	result := "MATCH"
	if assetID != link.AssetID {
		result = "MISMATCH"
	}
	if result == "MATCH" {
		if _, err := s.db.Exec(ctx,
			`UPDATE quad_links SET status = 'LINKED' WHERE id = $1`, link.ID); err != nil {
			return "", fmt.Errorf("quadlink: scan link: %w", err)
		}
	}
	if req.OfflineCalc {
		result = "OFFLINE_CACHED"
	}
	var scanID int64
	if err := s.db.QueryRow(ctx,
		`INSERT INTO scan_logs(order_id, worker_id, worker_name, tag_id, result)
		 VALUES($1,$2,$3,$4,$5) RETURNING id`,
		req.OrderID, req.WorkerID, req.WorkerName, tagID, result).Scan(&scanID); err != nil {
		return "", fmt.Errorf("quadlink: scan log: %w", err)
	}
	if result == "MISMATCH" {
		return result, ErrScanMismatch
	}
	return result, nil
}

// UnbindRequireScan 拆机必扫码:不扫码直接拒;扫码与已关联资产 EPC 不一致拒;一致则四码解绑(UNLINKED)。
func (s *PGStore) UnbindRequireScan(ctx context.Context, orderID int64, scannedEPC string) error {
	if scannedEPC == "" {
		return ErrScanRequired
	}
	link, err := s.linkByOrderCustomer(ctx, orderID)
	if err != nil {
		return err
	}
	var epc string
	err = s.db.QueryRow(ctx, `SELECT epc_code FROM tags WHERE bound_asset_id = $1`, link.AssetID).Scan(&epc)
	if errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("%w: asset %d 无标签", ErrScanMismatch, link.AssetID)
	}
	if err != nil {
		return fmt.Errorf("quadlink: unbind tag: %w", err)
	}
	if epc != scannedEPC {
		return ErrScanMismatch
	}
	if _, err := s.db.Exec(ctx,
		`UPDATE quad_links SET status = 'UNLINKED' WHERE id = $1`, link.ID); err != nil {
		return fmt.Errorf("quadlink: unbind: %w", err)
	}
	return nil
}

// Reconcile 四码对账任务:成员缺失置 CONFLICT → 自动清理孤儿行 → 返回统计。
func (s *PGStore) Reconcile(ctx context.Context) (*ReconcileReport, error) {
	// 1. 标 CONFLICT。
	if _, err := s.db.Exec(ctx, `
		UPDATE quad_links ql SET status = 'CONFLICT'
		WHERE NOT EXISTS (SELECT 1 FROM assets a WHERE a.id = ql.asset_id)
		   OR NOT EXISTS (SELECT 1 FROM customers c WHERE c.id = ql.customer_id)
		   OR NOT EXISTS (SELECT 1 FROM ports p WHERE p.id = ql.port_id)
		   OR NOT EXISTS (SELECT 1 FROM addresses ad WHERE ad.id = ql.address_id)`); err != nil {
		return nil, fmt.Errorf("quadlink: reconcile conflict: %w", err)
	}
	// 2. 清理孤儿行(与标 CONFLICT 相同判定条件)。
	purged, err := s.PurgeOrphans(ctx)
	if err != nil {
		return nil, fmt.Errorf("quadlink: reconcile purge: %w", err)
	}
	// 3. 统计。
	rows, err := s.db.Query(ctx, `SELECT status, count(*) FROM quad_links GROUP BY status`)
	if err != nil {
		return nil, fmt.Errorf("quadlink: reconcile stats: %w", err)
	}
	defer rows.Close()
	rep := &ReconcileReport{Purged: int(purged)}
	for rows.Next() {
		var st string
		var n int
		if err := rows.Scan(&st, &n); err != nil {
			return nil, fmt.Errorf("quadlink: reconcile scan: %w", err)
		}
		rep.Total += n
		switch st {
		case "LINKED":
			rep.Linked = n
		case "CONFLICT":
			rep.Conflict = n
		case "UNLINKED":
			rep.Unlinked = n
		}
	}
	return rep, rows.Err()
}

// ErrIllegalTransition 非冲突态不可执行冲突处理。
var ErrIllegalTransition = errors.New("quadlink: illegal transition")

// ResolveConflict 四码冲突人工处理:仅 CONFLICT 可置回 UNLINKED(修复后重新预绑定/扫码)。
func (s *PGStore) ResolveConflict(ctx context.Context, linkID int64) error {
	tag, err := s.db.Exec(ctx,
		`UPDATE quad_links SET status = 'UNLINKED' WHERE id = $1 AND status = 'CONFLICT'`, linkID)
	if err != nil {
		return fmt.Errorf("quadlink: resolve conflict: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrIllegalTransition
	}
	return nil
}

// PurgeOrphans 删除所有孤儿 quad_link 行(成员不存在则删),返回删除条数。
// 孤儿判定与 Reconcile 相同:资产/客户/端口/地址任一实体缺失。
// 建议在 Reconcile 之后调用,先标 CONFLICT 再清理。
func (s *PGStore) PurgeOrphans(ctx context.Context) (int64, error) {
	tag, err := s.db.Exec(ctx, `
		DELETE FROM quad_links ql
		WHERE NOT EXISTS (SELECT 1 FROM assets a WHERE a.id = ql.asset_id)
		   OR NOT EXISTS (SELECT 1 FROM customers c WHERE c.id = ql.customer_id)
		   OR NOT EXISTS (SELECT 1 FROM ports p WHERE p.id = ql.port_id)
		   OR NOT EXISTS (SELECT 1 FROM addresses ad WHERE ad.id = ql.address_id)`)
	if err != nil {
		return 0, fmt.Errorf("quadlink: purge orphans: %w", err)
	}
	return tag.RowsAffected(), nil
}

// linkByOrder 经订单→端口预占定位预绑定四码(扫码绑定场景)。
func (s *PGStore) linkByOrder(ctx context.Context, orderID int64) (*QuadLink, error) {
	var customerID int64
	err := s.db.QueryRow(ctx, `SELECT customer_id FROM orders WHERE id = $1`, orderID).Scan(&customerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("quadlink: order customer: %w", err)
	}
	var portID int64
	err = s.db.QueryRow(ctx,
		`SELECT id FROM ports WHERE order_id = $1 AND status = 'RESERVED'`, orderID).Scan(&portID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotPrebound
	}
	if err != nil {
		return nil, fmt.Errorf("quadlink: order port: %w", err)
	}
	link, err := s.getBy(ctx, "port_id", portID)
	if errors.Is(err, ErrNotFound) {
		return nil, ErrNotPrebound // 端口已预占但四码未预绑定(applyTag 未执行)
	}
	return link, err
}

// linkByOrderCustomer 经订单→客户定位已关联四码(拆机场景,端口可能已回收)。
func (s *PGStore) linkByOrderCustomer(ctx context.Context, orderID int64) (*QuadLink, error) {
	var customerID int64
	err := s.db.QueryRow(ctx, `SELECT customer_id FROM orders WHERE id = $1`, orderID).Scan(&customerID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("quadlink: order customer: %w", err)
	}
	return s.getBy(ctx, "customer_id", customerID)
}
