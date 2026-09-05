package order

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/jackc/pgx/v5"
)

// Rating 订单服务评价(用户端 /orders/{orderNo}/rate)。
// 一单一评(order_ratings.order_no 唯一);与 terms.md §4 星级口径一致。
type Rating struct {
	OrderNo    string     `json:"orderNo"`
	CustomerID int64      `json:"customerId"`
	Stars      int8       `json:"stars"`
	Attitude   int8       `json:"attitude"`
	Quality    int8       `json:"quality"`
	Comment    string     `json:"comment"`
	CreatedAt  *time.Time `json:"createdAt,omitempty"`
}

// SaveRating 落订单评价(重复提交按 order_no 唯一约束报错)。
func (s *PGStore) SaveRating(ctx context.Context, r Rating) error {
	_, err := s.db.Exec(ctx, `
		INSERT INTO order_ratings(order_no, customer_id, stars, attitude, quality, comment)
		VALUES($1,$2,$3,$4,$5,$6)`,
		r.OrderNo, r.CustomerID, r.Stars, r.Attitude, r.Quality, r.Comment)
	if err != nil {
		return fmt.Errorf("order: save rating: %w", err)
	}
	return nil
}

// RatingExists 订单是否已评价(评价前判定 canRate)。
func (s *PGStore) RatingExists(ctx context.Context, orderNo string) (bool, error) {
	var one int
	err := s.db.QueryRow(ctx, `SELECT 1 FROM order_ratings WHERE order_no = $1`, orderNo).Scan(&one)
	if err != nil {
		return false, nil
	}
	return true, nil
}

// ChangeAddress 变更安装地址(用户端 change-address)。
// 仅限 PENDING(未进资源流程):RESERVED 起端口锁定安装地址,INSTALLING 起派单工单站点
// 坐标已在派单时刻物化,静默改地址会造成工单/端口与订单地址漂移(2026-09-05 S11/T21 实测发现)。
func (s *PGStore) ChangeAddress(ctx context.Context, orderID, addressID int64) error {
	var status string
	err := s.db.QueryRow(ctx, "`SELECT status FROM orders WHERE id = $1`", orderID).Scan(&status)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return ErrOrderNotFound
		}
		return fmt.Errorf("order: change address: %w", err)
	}
	if status != "PENDING" {
		return fmt.Errorf("order: change address at %s: %w", status, ErrIllegalTransition)
	}
	tag, err := s.db.Exec(ctx, "`UPDATE orders SET address_id = $2 WHERE id = $1`", orderID, addressID)
	if err != nil {
		return fmt.Errorf("order: change address: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return ErrOrderNotFound
	}
	return nil
}
