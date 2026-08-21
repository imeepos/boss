package portal

import (
	"context"
	"strings"
	"testing"
)

// 内存实现全方法覆盖:正常路径 + 未命中/错误分支,与 PG 实现同一契约。
func TestMemorySms(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()
	if err := s.IssueSms(ctx, "+8613800000001", "login"); err != nil {
		t.Fatalf("IssueSms: %v", err)
	}
	for _, tc := range []struct {
		name string
		code string
		want bool
	}{
		{"wrong code", "000000", false},
		{"empty code", "", false},
		{"unknown phone", "x", false},
	} {
		phone := "+8613800000001"
		if tc.code == "x" {
			phone = "+8613800000099"
		}
		got, err := s.ConsumeSms(ctx, phone, "login", tc.code)
		if err != nil || got != tc.want {
			t.Fatalf("%s: ConsumeSms=%v,%v", tc.name, got, err)
		}
	}
	got, err := s.ConsumeSms(ctx, "+8613800000001", "login", "123456")
	if err != nil || !got {
		t.Fatalf("correct code: %v,%v", got, err)
	}
	// 一次性消费:第二次同码失败。
	if got, _ = s.ConsumeSms(ctx, "+8613800000001", "login", "123456"); got {
		t.Fatal("code reusable after consume")
	}
}

// TestMemoryLatestSmsCode 开发模式回显:签发后能查到,消费后变 nil,未签发也 nil。
func TestMemoryLatestSmsCode(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()
	if got, err := s.LatestSmsCode(ctx, "+8613800000001", "login"); err != nil || got != nil {
		t.Fatalf("未签发应为 nil: %+v, %v", got, err)
	}
	if err := s.IssueSms(ctx, "+8613800000001", "login"); err != nil {
		t.Fatalf("IssueSms: %v", err)
	}
	got, err := s.LatestSmsCode(ctx, "+8613800000001", "login")
	if err != nil || got == nil || got.Code != "123456" {
		t.Fatalf("签发后应查到 123456: %+v, %v", got, err)
	}
	if _, err := s.ConsumeSms(ctx, "+8613800000001", "login", "123456"); err != nil {
		t.Fatalf("ConsumeSms: %v", err)
	}
	if got, err := s.LatestSmsCode(ctx, "+8613800000001", "login"); err != nil || got != nil {
		t.Fatalf("消费后应为 nil: %+v, %v", got, err)
	}
}

func TestMemoryAccounts(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()
	if _, err := s.UpsertAccount(ctx, "+8613800000002", "secret", 7); err != nil {
		t.Fatalf("UpsertAccount: %v", err)
	}
	a, err := s.AccountByPhone(ctx, "+8613800000002")
	if err != nil || a.CustomerID != 7 {
		t.Fatalf("AccountByPhone: %v,%v", a, err)
	}
	if _, err = s.AccountByPhone(ctx, "+8613800000003"); err != ErrNotFound {
		t.Fatalf("AccountByPhone miss = %v, want ErrNotFound", err)
	}
	if a, err = s.AccountByCustomer(ctx, 7); err != nil || a.Phone != "+8613800000002" {
		t.Fatalf("AccountByCustomer: %v,%v", a, err)
	}
	if _, err = s.AccountByCustomer(ctx, 8); err != ErrNotFound {
		t.Fatalf("AccountByCustomer miss = %v", err)
	}
	// 超长密码触发 bcrypt 报错分支。
	if _, err = s.UpsertAccount(ctx, "+8613800000004", strings.Repeat("a", 100), 9); err == nil {
		t.Fatal("long password: want bcrypt error")
	}
	// 未知手机号 VerifyPassword 返回 false 而非错误。
	ok, err := s.VerifyPassword(ctx, "+8613800000003", "x")
	if err != nil || ok {
		t.Fatalf("VerifyPassword unknown phone: %v,%v", ok, err)
	}
	if ok, err = s.VerifyPassword(ctx, "+8613800000002", "wrong"); err != nil || ok {
		t.Fatalf("VerifyPassword wrong: %v,%v", ok, err)
	}
	if ok, err = s.VerifyPassword(ctx, "+8613800000002", "secret"); err != nil || !ok {
		t.Fatalf("VerifyPassword ok: %v,%v", ok, err)
	}
	// 改密走同一 upsert 入口。
	if _, err = s.UpsertAccount(ctx, "+8613800000002", "newpass", 7); err != nil {
		t.Fatalf("re-UpsertAccount: %v", err)
	}
	if ok, _ = s.VerifyPassword(ctx, "+8613800000002", "newpass"); !ok {
		t.Fatal("password not updated")
	}
}

func TestMemoryRebindPhone(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()
	if err := s.RebindPhone(ctx, 7, "+8613800000005"); err != ErrNotFound {
		t.Fatalf("RebindPhone miss = %v", err)
	}
	if _, err := s.UpsertAccount(ctx, "+8613800000002", "p", 7); err != nil {
		t.Fatal(err)
	}
	if err := s.RebindPhone(ctx, 7, "+8613800000005"); err != nil {
		t.Fatalf("RebindPhone: %v", err)
	}
	if _, err := s.AccountByPhone(ctx, "+8613800000002"); err != ErrNotFound {
		t.Fatalf("old phone still exists: %v", err)
	}
	if a, err := s.AccountByPhone(ctx, "+8613800000005"); err != nil || a.CustomerID != 7 {
		t.Fatalf("new phone: %v,%v", a, err)
	}
}

func TestMemoryPrefs(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()
	p, err := s.GetPrefs(ctx, 42)
	if err != nil || p.Language != "" || len(p.Notify) != 0 {
		t.Fatalf("GetPrefs default: %+v,%v", p, err)
	}
	if err = s.SavePrefs(ctx, 42, map[string]any{"email": true}, "zh"); err != nil {
		t.Fatalf("SavePrefs: %v", err)
	}
	// notify 为 nil 只改语言;语言空串只改 notify。
	if err = s.SavePrefs(ctx, 42, nil, "en"); err != nil {
		t.Fatal(err)
	}
	if err = s.SavePrefs(ctx, 42, map[string]any{"sms": false}, ""); err != nil {
		t.Fatal(err)
	}
	p, _ = s.GetPrefs(ctx, 42)
	if p.Language != "en" || p.Notify["sms"] != false {
		t.Fatalf("SavePrefs partial update lost: %+v", p)
	}
}

func TestMemoryMessages(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()
	unread, err := s.HasUnread(ctx, 42)
	if err != nil || unread {
		t.Fatalf("HasUnread empty: %v,%v", unread, err)
	}
	if err = s.PutMessage(ctx, 42, map[string]any{"t": "a"}); err != nil {
		t.Fatal(err)
	}
	if err = s.PutMessage(ctx, 42, map[string]any{"t": "b"}); err != nil {
		t.Fatal(err)
	}
	ms, err := s.Messages(ctx, 42)
	if err != nil || len(ms) != 2 || ms[0].Payload["t"] != "b" || ms[0].Read {
		t.Fatalf("Messages: %+v,%v", ms, err)
	}
	if unread, _ = s.HasUnread(ctx, 42); !unread {
		t.Fatal("HasUnread after put = false")
	}
	if err = s.MarkAllRead(ctx, 42); err != nil {
		t.Fatal(err)
	}
	if unread, _ = s.HasUnread(ctx, 42); unread {
		t.Fatal("HasUnread after mark = true")
	}
	ms, _ = s.Messages(ctx, 43)
	if len(ms) != 0 {
		t.Fatalf("Messages other customer = %d", len(ms))
	}
}

func TestMemoryWalletSeqAutopay(t *testing.T) {
	s := NewMemory()
	ctx := context.Background()
	if bal, err := s.Balance(ctx, 42); err != nil || bal != 0 {
		t.Fatalf("Balance default: %v,%v", bal, err)
	}
	if err := s.AdjustBalance(ctx, 42, 10.5); err != nil {
		t.Fatal(err)
	}
	if err := s.AdjustBalance(ctx, 42, -0.5); err != nil {
		t.Fatal(err)
	}
	if bal, _ := s.Balance(ctx, 42); bal != 10 {
		t.Fatalf("Balance = %v", bal)
	}
	no, err := s.NextNo(ctx, "PAY")
	if err != nil || no != "PAY-1" {
		t.Fatalf("NextNo: %v,%v", no, err)
	}
	if no, _ = s.NextNo(ctx, "PAY"); no != "PAY-2" {
		t.Fatalf("NextNo seq = %v", no)
	}
	if on, err := s.AutoPay(ctx, 42); err != nil || on {
		t.Fatalf("AutoPay default: %v,%v", on, err)
	}
	if err = s.SetAutoPay(ctx, 42, true); err != nil {
		t.Fatal(err)
	}
	if on, _ := s.AutoPay(ctx, 42); !on {
		t.Fatal("AutoPay after enable = false")
	}
}
