package odn

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
)

// portCols 端口查询列。
const portCols = "p.id, p.device_id, p.port_no, p.status, COALESCE(p.order_id,0), " +
	"to_char(p.updated_at,'YYYY-MM-DD HH24:MI:SS')"

// ListPorts 设备端口列表(端口号序)。
func (s *PGStore) ListPorts(ctx context.Context, deviceID int64) ([]ODNPort, error) {
	rows, err := s.db.Query(ctx, `SELECT `+portCols+` FROM odn_port p WHERE p.device_id=$1 ORDER BY p.port_no`, deviceID)
	if err != nil {
		return nil, fmt.Errorf("odn: list ports: %w", err)
	}
	defer rows.Close()
	out := []ODNPort{}
	for rows.Next() {
		var p ODNPort
		if err := rows.Scan(&p.ID, &p.DeviceID, &p.PortNo, &p.Status, &p.OrderID, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("odn: scan port: %w", err)
		}
		out = append(out, p)
	}
	return out, rows.Err()
}

// AllocatePort 设备上分配空闲端口(IDLE→RESERVED,SKIP LOCKED 防并发双占)。
func (s *PGStore) AllocatePort(ctx context.Context, deviceID, orderID int64) (*ODNPort, error) {
	var ord any
	if orderID > 0 {
		ord = orderID
	}
	q := `UPDATE odn_port SET status='RESERVED', order_id=$2, updated_at=now()
		WHERE device_id=$1 AND status='IDLE' AND id=(
		SELECT id FROM odn_port WHERE device_id=$1 AND status='IDLE' ORDER BY port_no LIMIT 1 FOR UPDATE SKIP LOCKED)
		RETURNING id, port_no`
	var p ODNPort
	err := s.db.QueryRow(ctx, q, deviceID, ord).Scan(&p.ID, &p.PortNo)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNoFreePort
	}
	if err != nil {
		log.Printf("[odn-port] ALLOCATE FAILED device=%d order=%d: %v", deviceID, orderID, err)
		return nil, fmt.Errorf("odn: allocate port: %w", err)
	}
	p.DeviceID = deviceID
	p.Status = PortReserved
	p.OrderID = orderID
	return &p, nil
}

// ReleasePort 释放预占(RESERVED→IDLE,拆机 IN_SERVICE→IDLE;终态 CAS)。
func (s *PGStore) ReleasePort(ctx context.Context, portID int64) error {
	return s.casPort(ctx, portID, PortIdle)
}

// ActivatePort 端口开通(RESERVED→IN_SERVICE,保持订单绑定)。
func (s *PGStore) ActivatePort(ctx context.Context, portID int64) error {
	return s.casPort(ctx, portID, PortInSvc)
}

// casPort 端口状态 CAS 转移;0 行时区分不存在与状态漂移;失败留痕。
func (s *PGStore) casPort(ctx context.Context, portID int64, to string) error {
	var from string
	err := s.db.QueryRow(ctx, `SELECT status FROM odn_port WHERE id=$1`, portID).Scan(&from)
	if errors.Is(err, pgx.ErrNoRows) {
		return ErrNotFound
	}
	if err != nil {
		return fmt.Errorf("odn: port read: %w", err)
	}
	if err := ValidatePortTransition(from, to); err != nil {
		return err
	}
	tag, err := s.db.Exec(ctx, `UPDATE odn_port SET status=$2, updated_at=now()
		WHERE id=$1 AND status=$3`, portID, to, from)
	if err != nil {
		log.Printf("[odn-port] CAS FAILED port=%d to=%s: %v", portID, to, err)
		return fmt.Errorf("odn: port cas: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrInvalidPortState
	}
	return nil
}
