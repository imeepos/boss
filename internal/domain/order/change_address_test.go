package order

// ChangeAddress 状态闸门回归(2026-09-05 S11/T21 发现:施工段改地址被静默接受)。
// 契约:仅 PENDING(未进资源流程)可变更安装地址;RESERVED 起端口锁定地址,
// INSTALLING 起派单工单站点坐标已物化,继续放行会造成工单/端口与订单地址漂移。

import (
	"context"
	"errors"
	"testing"
)

func changeAddrSvc(t *testing.T) *MemoryService {
	t.Helper()
	s, _ := newSvc(1)
	if _, err := s.Submit(context.Background(), SubmitReq{CustomerID: 1, OfferID: 10, AddressID: 100, ChannelID: 5}); err != nil {
		t.Fatalf("Submit err = %v", err)
	}
	return s
}

func TestChangeAddressPendingAllowed(t *testing.T) {
	s := changeAddrSvc(t)
	o, _ := s.List(context.Background(), OrderQuery{})
	if len(o) == 0 {
		t.Fatal("no order")
	}
	if err := s.ChangeAddress(context.Background(), o[0].ID, 200); err != nil {
		t.Fatalf("ChangeAddress at PENDING err = %v", err)
	}
	if s.m[o[0].ID].AddressID != 200 {
		t.Fatalf("AddressID = %d, want 200", s.m[o[0].ID].AddressID)
	}
}

func TestChangeAddressRejectsNonPending(t *testing.T) {
	for _, status := range []string{"RESERVED", "INSTALLING", "DONE", "CANCELLED"} {
		s := changeAddrSvc(t)
		o, _ := s.List(context.Background(), OrderQuery{})
		s.m[o[0].ID].Status = status
		err := s.ChangeAddress(context.Background(), o[0].ID, 200)
		if !errors.Is(err, ErrIllegalTransition) {
			t.Fatalf("ChangeAddress at %s err = %v, want ErrIllegalTransition", status, err)
		}
	}
}
