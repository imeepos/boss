package order

import (
	"context"
	"errors"
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

	// 环节2 资源核查通过后,环节3 才可预占。
	if err := s.CheckResource(context.Background(), o.ID); err != nil {
		t.Fatalf("CheckResource err = %v", err)
	}
	if err := s.Reserve(context.Background(), o.ID); err != nil {
		t.Fatalf("Reserve err = %v", err)
	}
	got, _, _ := s.Track(context.Background(), o.ID)
	if got.Status != "RESERVED" || got.Stage != 3 {
		t.Fatalf("status=%q stage=%d, want RESERVED/3", got.Status, got.Stage)
	}

	// 第二次 Reserve:stage 已=3,顺序守卫拒绝(环节不可重复)。
	if err := s.Reserve(context.Background(), o.ID); err != ErrIllegalTransition {
		t.Fatalf("err = %v, want ErrIllegalTransition", err)
	}
	_ = checker
}

// TestRollbackStage 回归(ISSUE.md worker rollback 仅审计不落库):
// 回退删最新日志、stage 前移、DONE→INSTALLING 逆向迁移,且可重新推进。
func TestRollbackStage(t *testing.T) {
	s, _ := newSvc(1)
	ctx := context.Background()
	o, _ := s.Submit(ctx, SubmitReq{CustomerID: 1, OfferID: 10, AddressID: 100, ChannelID: 5})
	for _, fn := range []func(context.Context, int64) error{
		s.CheckResource, s.Reserve, s.ChargeContract, s.ApplyTag, s.CreateUserProfile,
		s.PreConfigOLT, s.DispatchOrder, s.ScanBind, s.ActivateUser, s.NotifyActivation, s.UpdateMap,
	} {
		if err := fn(ctx, o.ID); err != nil {
			t.Fatalf("advance: %v", err)
		}
	}
	if err := s.RollbackStage(ctx, o.ID); err != nil {
		t.Fatalf("RollbackStage: %v", err)
	}
	got, logs, _ := s.Track(ctx, o.ID)
	// 环节12(updateMap)不产生 status,回退到 11 时 DONE 保持(done 由环节11产生)。
	if got.Stage != 11 || got.Status != "DONE" {
		t.Fatalf("stage=%d status=%s, want 11/DONE", got.Stage, got.Status)
	}
	if n := len(logs); n != 11 { // 环节1~11(环节12日志已删)
		t.Fatalf("logs=%d, want 11", n)
	}
	// 再回退一次:离开环节11 → DONE→INSTALLING。
	if err := s.RollbackStage(ctx, o.ID); err != nil {
		t.Fatalf("RollbackStage(2): %v", err)
	}
	got, _, _ = s.Track(ctx, o.ID)
	if got.Stage != 10 || got.Status != "INSTALLING" {
		t.Fatalf("stage=%d status=%s, want 10/INSTALLING", got.Stage, got.Status)
	}
	// 重新推进 11/12 环节可走通(顺序守卫以回退后的 stage 为准)。
	for _, fn := range []func(context.Context, int64) error{s.NotifyActivation, s.UpdateMap} {
		if err := fn(ctx, o.ID); err != nil {
			t.Fatalf("re-advance: %v", err)
		}
	}
	got, _, _ = s.Track(ctx, o.ID)
	if got.Stage != 12 || got.Status != "DONE" {
		t.Fatalf("re-advanced stage=%d status=%s, want 12/DONE", got.Stage, got.Status)
	}
	// stage<2 拒绝回退。
	o2, _ := s.Submit(ctx, SubmitReq{CustomerID: 1, OfferID: 10, AddressID: 100, ChannelID: 5})
	if err := s.RollbackStage(ctx, o2.ID); err != ErrIllegalTransition {
		t.Fatalf("stage1 rollback err=%v, want ErrIllegalTransition", err)
	}
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

// TestMemoryService_SubmitOwnership 契约:注入 OwnershipResolver 时归属由地址推导,
// 请求直传值仅做冲突校验(adopted note 2026-08-20-order-legal-entity-by-address)。
func TestMemoryService_SubmitOwnership(t *testing.T) {
	own := OwnershipMap{100: {LegalEntityID: 2, RegionPath: "root.luzon.ncr"}}
	cust := stubExists{ok: true}
	s := NewMemoryService(cust, &stubChecker{}, own)

	o, err := s.Submit(context.Background(), SubmitReq{CustomerID: 1, AddressID: 100, ChannelID: 5})
	if err != nil {
		t.Fatalf("Submit: %v", err)
	}
	if o.LegalEntityID != 2 || o.RegionPath != "root.luzon.ncr" {
		t.Fatalf("ownership not derived: %+v", o)
	}

	if _, err := s.Submit(context.Background(), SubmitReq{
		CustomerID: 1, AddressID: 100, ChannelID: 5, LegalEntityID: 1,
	}); !errors.Is(err, ErrOwnershipMismatch) {
		t.Fatalf("err=%v, want ErrOwnershipMismatch", err)
	}

	if _, err := s.Submit(context.Background(), SubmitReq{
		CustomerID: 1, AddressID: 999, ChannelID: 5,
	}); !errors.Is(err, ErrAddressNotCovered) {
		t.Fatalf("err=%v, want ErrAddressNotCovered", err)
	}
}
