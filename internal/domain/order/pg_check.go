package order

import (
	"context"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// CheckResource 资源核查(环节2):调 ResourceChecker,推进 stage=2;无资源停在 stage=2 等待。
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
	if stage >= 2 {
		return nil // 幂等:已核查过不再重复推进
	}
	if s.checker == nil {
		return errors.New("order: resource checker not wired")
	}
	avail, _, err := s.checker.Check(ctx, addressID)
	if err != nil {
		return err
	}
	result := "DONE"
	if !avail {
		result = "PENDING"
	}
	if _, err := s.db.Exec(ctx, `UPDATE orders SET stage = 2 WHERE id = $1`, orderID); err != nil {
		return fmt.Errorf("order: check resource update: %w", err)
	}
	return s.appendStage(ctx, orderID, 2, result)
}
