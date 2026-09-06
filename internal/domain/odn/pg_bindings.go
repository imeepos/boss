package odn

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

// bindCols 绑定查询列。
const bindCols = "b.id, b.port_id, b.order_id, COALESCE(b.resource_port_id,0), " +
	"COALESCE(b.note,''), to_char(b.bound_at,'YYYY-MM-DD HH24:MI:SS')"

// BindPort 写绑定事实:端口须 IN_SERVICE(激活是绑定的前置,状态机保证);一口一绑定。
func (s *PGStore) BindPort(ctx context.Context, b ODNBinding) error {
	if err := ValidateBinding(b.OrderID, b.PortID); err != nil {
		return err
	}
	var lc string
	if err := s.db.QueryRow(ctx, "SELECT status FROM odn_port WHERE id=$1", b.PortID).Scan(&lc); err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrNotFound
		}
		return fmt.Errorf("odn: binding port read: %w", err)
	}
	if lc != PortInSvc {
		return ErrPortNotInService
	}
	var rp any
	if b.ResourcePortID > 0 {
		rp = b.ResourcePortID
	}
	var by any
	if b.BoundBy > 0 {
		by = b.BoundBy
	}
	_, err := s.db.Exec(ctx, `INSERT INTO odn_bindings (port_id, order_id, resource_port_id, note, bound_by)
		VALUES ($1,$2,$3,$4,$5)`, b.PortID, b.OrderID, rp, b.Note, by)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) {
			if pgErr.Code == "23505" {
				return ErrDuplicate
			}
			if pgErr.Code == "23503" {
				return ErrInvalidPortState // 端口不在 IN_SERVICE 时的兜底(并发下状态先变)
			}
		}
		log.Printf("[odn-binding] BIND FAILED port=%d order=%d: %v", b.PortID, b.OrderID, err)
		return fmt.Errorf("odn: bind port: %w", err)
	}
	return nil
}

// UnbindPort 解绑(按端口:一口一绑定)。
func (s *PGStore) UnbindPort(ctx context.Context, portID int64) error {
	tag, err := s.db.Exec(ctx, `DELETE FROM odn_bindings WHERE port_id=$1`, portID)
	if err != nil {
		log.Printf("[odn-binding] UNBIND FAILED port=%d: %v", portID, err)
		return fmt.Errorf("odn: unbind port: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrBindingNotFound
	}
	return nil
}

// ListBindingsByPort 按物理口反查绑定(GIS 点位反查在用订单的数据源)。
func (s *PGStore) ListBindingsByPort(ctx context.Context, portID int64) ([]ODNBinding, error) {
	return s.queryBindings(ctx, `SELECT `+bindCols+` FROM odn_bindings b WHERE b.port_id=$1 ORDER BY b.bound_at DESC`, portID)
}

// ListBindingsByOrder 按订单反查绑定(售后侧:这单用的哪个物理口)。
func (s *PGStore) ListBindingsByOrder(ctx context.Context, orderID int64) ([]ODNBinding, error) {
	return s.queryBindings(ctx, `SELECT `+bindCols+` FROM odn_bindings b WHERE b.order_id=$1 ORDER BY b.bound_at DESC`, orderID)
}

func (s *PGStore) queryBindings(ctx context.Context, q string, args ...any) ([]ODNBinding, error) {
	rows, err := s.db.Query(ctx, q, args...)
	if err != nil {
		return nil, fmt.Errorf("odn: query bindings: %w", err)
	}
	defer rows.Close()
	out := []ODNBinding{}
	for rows.Next() {
		var b ODNBinding
		if err := rows.Scan(&b.ID, &b.PortID, &b.OrderID, &b.ResourcePortID, &b.Note, &b.BoundAt); err != nil {
			return nil, fmt.Errorf("odn: scan binding: %w", err)
		}
		out = append(out, b)
	}
	return out, rows.Err()
}
