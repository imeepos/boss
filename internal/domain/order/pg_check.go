package order

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// CheckResource 资源核查(环节2):调 ResourceChecker,推进 stage=2;无资源停在 stage=2 等待。
// 自愈:环节2 PENDING(此前无资源)时可重查,资源已可用则补正为 DONE,
// 修复"环节2 永久 PENDING 卡死后续环节"的遗留形态(2026-08-25 审计 §2.3.2)。
func (s *PGStore) CheckResource(ctx context.Context, orderID int64) error {
	var stage int8
	var addressID int64
	err := s.db.QueryRow(ctx, `SELECT stage, address_id FROM orders WHERE id = $1`, orderID).
		Scan(&stage, &addressID)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrOrderNotFound
	}
	if err != nil {
		return fmt.Errorf("order: check resource select: %w", err)
	}
	var prevResult string
	err = s.db.QueryRow(ctx,
		`SELECT result FROM order_stages WHERE order_id=$1 AND stage=2`, orderID).Scan(&prevResult)
	switch {
	case err == nil && prevResult == "DONE":
		return nil // 幂等:已核查通过不再重复推进
	case err != nil && !errors.Is(err, pgx.ErrNoRows):
		return fmt.Errorf("order: check resource prev: %w", err)
	}
	if s.checker == nil {
		return errors.New("order: resource checker not wired")
	}
	avail, _, err := s.checker.Check(ctx, addressID)
	if err != nil {
		return err
	}
	if stage < 2 {
		if _, err := s.db.Exec(ctx, `UPDATE orders SET stage = 2 WHERE id = $1`, orderID); err != nil {
			return fmt.Errorf("order: check resource update: %w", err)
		}
	}
	if prevResult == "PENDING" {
		if !avail {
			return nil // 仍无资源,维持等待
		}
		// 自愈补正:原 PENDING 行改 DONE,不重复插入。
		_, err := s.db.Exec(ctx,
			`UPDATE order_stages SET result='DONE', finished_at=now()
			 WHERE order_id=$1 AND stage=2 AND result='PENDING'`, orderID)
		return err
	}
	result := "DONE"
	if !avail {
		result = "PENDING"
	}
	return s.appendStage(ctx, orderID, 2, result)
}
