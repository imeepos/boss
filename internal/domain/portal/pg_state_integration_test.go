package portal

import (
	"context"
	"math"
	"testing"
)

// pgStateCleanup 清理本次测试客户的数据(prefs/messages/wallets/billing_prefs)。
func pgStateCleanup(t *testing.T, s Service, cid int64) {
	t.Helper()
	pool := s.(*pgStore).pool
	pool.Exec(context.Background(), `DELETE FROM portal_prefs WHERE customer_id=$1;
		DELETE FROM portal_messages WHERE customer_id=$1;
		DELETE FROM portal_wallets WHERE customer_id=$1;
		DELETE FROM portal_billing_prefs WHERE customer_id=$1`, cid)
}

// TestPGPrefs 偏好读写:缺省/部分更新/非法 JSON/ctx 取消。
func TestPGPrefs(t *testing.T) {
	s := openPGTest(t)
	const cid int64 = -888101
	pgStateCleanup(t, s, cid)
	defer pgStateCleanup(t, s, cid)
	ctx := context.Background()

	p, err := s.GetPrefs(ctx, cid)
	if err != nil || p.Language != "" || len(p.Notify) != 0 {
		t.Fatalf("GetPrefs default: %+v,%v", p, err)
	}
	if err := s.SavePrefs(ctx, cid, map[string]any{"email": true}, "zh"); err != nil {
		t.Fatalf("SavePrefs: %v", err)
	}
	// notify 为 nil 只改语言;语言空串只改 notify。
	if err = s.SavePrefs(ctx, cid, nil, "en"); err != nil {
		t.Fatal(err)
	}
	if err = s.SavePrefs(ctx, cid, map[string]any{"sms": false}, ""); err != nil {
		t.Fatal(err)
	}
	if p, err = s.GetPrefs(ctx, cid); err != nil || p.Language != "en" || p.Notify["sms"] != false {
		t.Fatalf("GetPrefs after save: %+v,%v", p, err)
	}
	// notify 存为 jsonb 标量触发 json.Unmarshal 错误分支。
	s.(*pgStore).pool.Exec(ctx, `UPDATE portal_prefs SET notify='123'::jsonb WHERE customer_id=$1`, cid)
	if _, err := s.GetPrefs(ctx, cid); err == nil {
		t.Fatal("scalar notify: want unmarshal error")
	}
	// 还原合法 notify 后,通道值无法 Marshal 触发 json.Marshal 错误分支。
	s.(*pgStore).pool.Exec(ctx, `UPDATE portal_prefs SET notify='{}'::jsonb WHERE customer_id=$1`, cid)
	if err := s.SavePrefs(ctx, cid, map[string]any{"ch": make(chan int)}, ""); err == nil {
		t.Fatal("chan notify: want marshal error")
	}
	cctx, cancel := cancelledCtx()
	defer cancel()
	if _, err := s.GetPrefs(cctx, cid); err == nil {
		t.Fatal("GetPrefs cancelled: want error")
	}
	if err := s.SavePrefs(cctx, cid, nil, "zh"); err == nil {
		t.Fatal("SavePrefs cancelled: want error")
	}
}

// TestPGMessages 站内消息:写入/倒序/已读/未读/非法 JSON/ctx 取消。
func TestPGMessages(t *testing.T) {
	s := openPGTest(t)
	const cid int64 = -888102
	pgStateCleanup(t, s, cid)
	defer pgStateCleanup(t, s, cid)
	ctx := context.Background()

	unread, err := s.HasUnread(ctx, cid)
	if err != nil || unread {
		t.Fatalf("HasUnread empty: %v,%v", unread, err)
	}
	if err := s.PutMessage(ctx, cid, map[string]any{"t": "a"}); err != nil {
		t.Fatalf("PutMessage: %v", err)
	}
	if err := s.PutMessage(ctx, cid, map[string]any{"t": "b"}); err != nil {
		t.Fatal(err)
	}
	ms, err := s.Messages(ctx, cid)
	if err != nil || len(ms) != 2 || ms[0].Payload["t"] != "b" || ms[0].Read {
		t.Fatalf("Messages: %+v,%v", ms, err)
	}
	if unread, _ = s.HasUnread(ctx, cid); !unread {
		t.Fatal("HasUnread after put = false")
	}
	if err := s.MarkAllRead(ctx, cid); err != nil {
		t.Fatalf("MarkAllRead: %v", err)
	}
	if unread, _ = s.HasUnread(ctx, cid); unread {
		t.Fatal("HasUnread after mark = true")
	}
	// created_at=infinity 无法映射 time.Time,触发 rows.Scan 错误分支。
	s.(*pgStore).pool.Exec(ctx, `INSERT INTO portal_messages(customer_id, payload, created_at)
		VALUES ($1, '{}'::jsonb, 'infinity')`, cid)
	if _, err := s.Messages(ctx, cid); err == nil {
		t.Fatal("overflow id: want scan error")
	}
	// payload 存为 jsonb 标量触发 Messages 反序列化错误分支。
	s.(*pgStore).pool.Exec(ctx, `UPDATE portal_messages SET payload='123'::jsonb WHERE customer_id=$1`, cid)
	if _, err := s.Messages(ctx, cid); err == nil {
		t.Fatal("scalar payload: want unmarshal error")
	}
	// 通道值无法 Marshal,触发 PutMessage Marshal 错误分支。
	if err := s.PutMessage(ctx, cid, map[string]any{"ch": make(chan int)}); err == nil {
		t.Fatal("chan payload: want marshal error")
	}
	cctx, cancel := cancelledCtx()
	defer cancel()
	if _, err := s.Messages(cctx, cid); err == nil {
		t.Fatal("Messages cancelled: want error")
	}
	if err := s.PutMessage(cctx, cid, nil); err == nil {
		t.Fatal("PutMessage cancelled: want error")
	}
	if err := s.MarkAllRead(cctx, cid); err == nil {
		t.Fatal("MarkAllRead cancelled: want error")
	}
	if _, err := s.HasUnread(cctx, cid); err == nil {
		t.Fatal("HasUnread cancelled: want error")
	}
}

// TestPGWallet 钱包:缺省零余额、累加、精度。
func TestPGWallet(t *testing.T) {
	s := openPGTest(t)
	const cid int64 = -888103
	pgStateCleanup(t, s, cid)
	defer pgStateCleanup(t, s, cid)
	ctx := context.Background()

	bal, err := s.Balance(ctx, cid)
	if err != nil || bal != 0 {
		t.Fatalf("Balance default: %v,%v", bal, err)
	}
	if err := s.AdjustBalance(ctx, cid, 10.25); err != nil {
		t.Fatalf("AdjustBalance: %v", err)
	}
	if err := s.AdjustBalance(ctx, cid, -0.25); err != nil {
		t.Fatal(err)
	}
	if bal, _ := s.Balance(ctx, cid); math.Abs(bal-10) > 1e-9 {
		t.Fatalf("Balance = %v, want 10", bal)
	}
	cctx, cancel := cancelledCtx()
	defer cancel()
	if _, err := s.Balance(cctx, cid); err == nil {
		t.Fatal("Balance cancelled: want error")
	}
	if err := s.AdjustBalance(cctx, cid, 1); err == nil {
		t.Fatal("AdjustBalance cancelled: want error")
	}
}

// TestPGAutoPay 自动缴费:缺省关、开/关 upsert。
func TestPGAutoPay(t *testing.T) {
	s := openPGTest(t)
	const cid int64 = -888104
	pgStateCleanup(t, s, cid)
	defer pgStateCleanup(t, s, cid)
	ctx := context.Background()

	if on, err := s.AutoPay(ctx, cid); err != nil || on {
		t.Fatalf("AutoPay default: %v,%v", on, err)
	}
	if err := s.SetAutoPay(ctx, cid, true); err != nil {
		t.Fatalf("SetAutoPay: %v", err)
	}
	if on, _ := s.AutoPay(ctx, cid); !on {
		t.Fatal("AutoPay after enable = false")
	}
	if err := s.SetAutoPay(ctx, cid, false); err != nil {
		t.Fatal(err)
	}
	if on, _ := s.AutoPay(ctx, cid); on {
		t.Fatal("AutoPay after disable = true")
	}
	cctx, cancel := cancelledCtx()
	defer cancel()
	if _, err := s.AutoPay(cctx, cid); err == nil {
		t.Fatal("AutoPay cancelled: want error")
	}
	if err := s.SetAutoPay(cctx, cid, true); err == nil {
		t.Fatal("SetAutoPay cancelled: want error")
	}
}
