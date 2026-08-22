package order

import (
	"context"
	"fmt"
	"time"
)

// 预占超时释放(terms.md 预占:超时释放,阈值走配置;roadmap §2 默认 30 分钟)。
// RESERVED 订单的环节3完成时间(缺日志回退建单时间)早于 cutoff → 回收端口 +
// 预占流水 HELD→RELEASED + status 经状态机 release 回 PENDING,不动环节序号。

// ReleaseExpiredReserves 批量释放超时预占,返回本轮释放的订单 id。
// 逐单独立执行:单单失败跳过继续,整轮失败才报错(与巡检循环尽力而为语义对齐)。
func (s *PGStore) ReleaseExpiredReserves(ctx context.Context, cutoff time.Time) ([]int64, error) {
	rows, err := s.db.Query(ctx, `
		SELECT o.id FROM orders o
		WHERE o.status = 'RESERVED'
		  AND COALESCE((SELECT max(st.finished_at) FROM order_stages st
		                WHERE st.order_id = o.id AND st.stage = 3), o.created_at) < $1
		ORDER BY o.id`, cutoff)
	if err != nil {
		return nil, fmt.Errorf("order: release expired select: %w", err)
	}
	defer rows.Close()
	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return nil, fmt.Errorf("order: release expired scan: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("order: release expired rows: %w", err)
	}
	for _, id := range ids {
		if err := s.releaseExpiredReserve(ctx, id); err != nil {
			return ids, fmt.Errorf("order: release expired %d: %w", id, err)
		}
	}
	return ids, nil
}

// releaseExpiredReserve 单单释放:与 Cancel 同谓词回收端口,另置预占流水 RELEASED,
// 再走 release 迁移(RESERVED→PENDING,工单随动由 transitionStatus 内处理)。
func (s *PGStore) releaseExpiredReserve(ctx context.Context, orderID int64) error {
	if _, err := s.db.Exec(ctx,
		`UPDATE ports SET status = 'IDLE', order_id = NULL WHERE order_id = $1 AND status = 'RESERVED'`, orderID,
	); err != nil {
		return fmt.Errorf("release ports: %w", err)
	}
	if _, err := s.db.Exec(ctx,
		`UPDATE reserve_records SET status = 'RELEASED' WHERE order_id = $1 AND status = 'HELD'`, orderID,
	); err != nil {
		return fmt.Errorf("release records: %w", err)
	}
	return s.transitionStatus(ctx, orderID, "release")
}
