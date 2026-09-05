package order

// 订单评价/地址变更/深拷贝工具(从 memory.go 拆出,保持单文件 ≤300 行)。

import (
	"context"
	"fmt"
)

func cloneOrder(o *Order) *Order {
	if o == nil {
		return nil
	}
	cp := *o
	return &cp
}

// ChangeAddress 变更安装地址(内存实现,与 PGStore 同口径:仅 PENDING 可改)。
func (s *MemoryService) ChangeAddress(ctx context.Context, orderID, addressID int64) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	o, ok := s.m[orderID]
	if !ok {
		return ErrOrderNotFound
	}
	if o.Status != "PENDING" {
		return fmt.Errorf("order: change address at %s: %w", o.Status, ErrIllegalTransition)
	}
	o.AddressID = addressID
	return nil
}

// SaveRating 落订单评价(内存实现:覆写同单号评价)。
func (s *MemoryService) SaveRating(ctx context.Context, r Rating) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.ratings == nil {
		s.ratings = make(map[string]Rating)
	}
	s.ratings[r.OrderNo] = r
	return nil
}

// RatingExists 订单是否已评价(内存实现)。
func (s *MemoryService) RatingExists(ctx context.Context, orderNo string) (bool, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, ok := s.ratings[orderNo]
	return ok, nil
}
