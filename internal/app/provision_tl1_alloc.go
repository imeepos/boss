// ONUNO 分配器(设计 §4):每 (OLT,PON) 幂等自增分配,同事务回写 ports.onu_no。
// 首次分配 next_no=0,其后 1,2,...;0~127 由网元/运维保证,此处只做行级幂等。
package app

import (
	"context"
	"fmt"
)

// onuNo 取端口 ONUNO:已分配(onu_no 非空)直接复用,否则 pon_onu_alloc 自增分配并回写。
// 分配与回写同事务,防并发重复占号;失败返回带原因的 error(任务 FAILED 可诊断)。
func (r *TL1ParamResolver) onuNo(ctx context.Context, p portRow) (int, error) {
	if p.onuNo.Valid {
		return int(p.onuNo.Int16), nil
	}
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return 0, fmt.Errorf("begin alloc: %w", err)
	}
	defer tx.Rollback(ctx)

	var next int
	err = tx.QueryRow(ctx, `
		INSERT INTO pon_onu_alloc(olt_resource_id, pon_frame, pon_slot, pon_port, next_no)
		VALUES ($1, $2, $3, $4, 0)
		ON CONFLICT (olt_resource_id, pon_frame, pon_slot, pon_port)
		DO UPDATE SET next_no = pon_onu_alloc.next_no + 1
		RETURNING next_no`,
		p.resourceID, p.ponFrame, p.ponSlot, p.ponPort).Scan(&next)
	if err != nil {
		return 0, fmt.Errorf("alloc onu_no: %w", err)
	}

	if _, err := tx.Exec(ctx, `UPDATE ports SET onu_no = $1 WHERE id = $2`, next, p.id); err != nil {
		return 0, fmt.Errorf("write onu_no: %w", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return 0, fmt.Errorf("commit alloc: %w", err)
	}
	return next, nil
}
