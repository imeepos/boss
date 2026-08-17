package order

import (
	"context"
	"testing"
)

// stubCustomer 桩:按 map 判定客户是否存在。
type stubCustomer struct {
	m map[int64]bool
}

func (s *stubCustomer) Exists(ctx context.Context, id int64) (bool, error) {
	return s.m[id], nil
}

// stubChecker 桩:固定返回核查结果。
type stubChecker struct {
	available bool
	options   []string
}

func (s *stubChecker) Check(ctx context.Context, addressID int64) (bool, []string, error) {
	return s.available, s.options, nil
}

func newSvc(custIDs ...int64) (*MemoryService, *stubChecker) {
	cust := &stubCustomer{m: map[int64]bool{}}
	for _, id := range custIDs {
		cust.m[id] = true
	}
	checker := &stubChecker{available: true}
	return NewMemoryService(cust, checker), checker
}

func TestSubmitOk(t *testing.T) {
	s, _ := newSvc(1)
	o, err := s.Submit(context.Background(), SubmitReq{CustomerID: 1, OfferID: 10, AddressID: 100, ChannelID: 5})
	if err != nil {
		t.Fatalf("Submit err = %v", err)
	}
	if o.Status != "PENDING" || o.Stage != 1 {
		t.Fatalf("status=%q stage=%d, want PENDING/1", o.Status, o.Stage)
	}
	if o.ChannelID != 5 {
		t.Fatalf("ChannelID = %d, want 5", o.ChannelID)
	}
}

func TestSubmitCustomerNotFound(t *testing.T) {
	s, _ := newSvc(1)
	if _, err := s.Submit(context.Background(), SubmitReq{CustomerID: 99, ChannelID: 5}); err == nil {
		t.Fatal("want error for missing customer")
	}
}

func TestSubmitChannelRequired(t *testing.T) {
	s, _ := newSvc(1)
	if _, err := s.Submit(context.Background(), SubmitReq{CustomerID: 1}); err == nil {
		t.Fatal("want error for missing channel")
	}
}

func TestReserveTransitions(t *testing.T) {
	s, checker := newSvc(1)
	o, _ := s.Submit(context.Background(), SubmitReq{CustomerID: 1, OfferID: 10, AddressID: 100, ChannelID: 5})

	if err := s.Reserve(context.Background(), o.ID); err != nil {
		t.Fatalf("Reserve err = %v", err)
	}
	got, _, _ := s.Track(context.Background(), o.ID)
	if got.Status != "RESERVED" {
		t.Fatalf("status = %q, want RESERVED", got.Status)
	}

	// 第二次 Reserve:RESERVED→reserve 无定义,应拒非法流转。
	if err := s.Reserve(context.Background(), o.ID); err != ErrIllegalTransition {
		t.Fatalf("err = %v, want ErrIllegalTransition", err)
	}
	_ = checker
}

func TestCheckResourceNoResource(t *testing.T) {
	cust := &stubCustomer{m: map[int64]bool{1: true}}
	checker := &stubChecker{available: false, options: []string{"扩容", "跨区调配"}}
	s := NewMemoryService(cust, checker)

	o, _ := s.Submit(context.Background(), SubmitReq{CustomerID: 1, OfferID: 10, AddressID: 100, ChannelID: 5})
	if err := s.CheckResource(context.Background(), o.ID); err != nil {
		t.Fatalf("CheckResource err = %v", err)
	}
	got, logs, _ := s.Track(context.Background(), o.ID)
	if got.Stage != 2 {
		t.Fatalf("stage = %d, want 2", got.Stage)
	}
	if len(logs) != 2 || logs[1].Result != "PENDING" {
		t.Fatalf("want 环节2 日志为 PENDING, got %+v", logs)
	}
}

func TestTrackNotFound(t *testing.T) {
	s, _ := newSvc()
	if _, _, err := s.Track(context.Background(), 99); err != ErrOrderNotFound {
		t.Fatalf("err = %v, want ErrOrderNotFound", err)
	}
}
