package portal

import (
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/ymm-001/boss/internal/pkg/database"
)

// openPGTest 打开测试 PG 并迁移(未设 DSN 时跳过)。
//
//	运行: BOSS_PG_TEST_DSN="host=192.168.0.102 port=25432 user=boss password=boss dbname=boss sslmode=disable" \
//		go test ./internal/domain/portal/ -v -count=1
func openPGTest(t *testing.T) Service {
	t.Helper()
	dsn := os.Getenv("BOSS_PG_TEST_DSN")
	if dsn == "" {
		t.Skip("BOSS_PG_TEST_DSN 未设置,跳过集成测试")
	}
	ctx := context.Background()
	pool, err := database.Open(ctx, dsn)
	if err != nil {
		t.Fatalf("database.Open: %v", err)
	}
	t.Cleanup(pool.Close)
	if err := database.Migrate(ctx, pool, "../../../migrations"); err != nil {
		t.Fatalf("database.Migrate: %v", err)
	}
	return NewPGStore(pool)
}

// stubSender 记录调用、可注入错误,覆盖外发通道两个分支。
type stubSender struct {
	err   error
	calls int
	last  string
}

func (s *stubSender) Send(_ context.Context, _, code, _ string) error {
	s.calls++
	s.last = code
	return s.err
}

func cancelledCtx() (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	return ctx, cancel
}

// TestPGIssueConsumeSms 验证码签发/冷却/一次性消费/通道错误闭环。
func TestPGIssueConsumeSms(t *testing.T) {
	s := openPGTest(t)
	pool := s.(*pgStore).pool
	ctx := context.Background()
	const phone = "+99900000001"
	pool.Exec(ctx, `DELETE FROM portal_sms_codes WHERE phone=$1`, phone)

	if err := s.IssueSms(ctx, phone, "login"); err != nil {
		t.Fatalf("IssueSms: %v", err)
	}
	if err := s.IssueSms(ctx, phone, "login"); err != ErrSmsCooldown {
		t.Fatalf("cooldown = %v, want ErrSmsCooldown", err)
	}
	if ok, err := s.ConsumeSms(ctx, phone, "login", ""); err != nil || ok {
		t.Fatalf("empty code: %v,%v", ok, err)
	}
	if ok, err := s.ConsumeSms(ctx, phone, "login", "000000"); err != nil || ok {
		t.Fatalf("wrong code: %v,%v", ok, err)
	}
	var code string
	if err := pool.QueryRow(ctx, `SELECT code FROM portal_sms_codes WHERE phone=$1 AND scene='login'`, phone).Scan(&code); err != nil {
		t.Fatal(err)
	}
	if ok, err := s.ConsumeSms(ctx, phone, "login", code); err != nil || !ok {
		t.Fatalf("correct code: %v,%v", ok, err)
	}
	if ok, _ := s.ConsumeSms(ctx, phone, "login", code); ok {
		t.Fatal("code reusable after consume")
	}

	// 注入故障随机源:IssueSms 签发失败分支。
	oldRand := randReader
	randReader = errReader{}
	rerr := s.IssueSms(ctx, phone, "reset2")
	randReader = oldRand
	if rerr == nil {
		t.Fatal("failing rand: want error")
	}
	// 超长手机号:SELECT 无命中继续,INSERT 违反 VARCHAR(32) 触发 Exec 错误分支。
	if err := s.IssueSms(ctx, strings.Repeat("9", 33), "login"); err == nil {
		t.Fatal("long phone: want insert error")
	}
	// 已取消 ctx 触发首个 QueryRow 错误分支。
	cctx, cancel := cancelledCtx()
	defer cancel()
	if err := s.IssueSms(cctx, phone, "reset"); err == nil {
		t.Fatal("cancelled ctx: want error")
	}

	// 带通道:外发成功与外发失败两个分支。
	st := &stubSender{}
	ps := NewPGStoreWithSender(pool, st)
	if err := ps.IssueSms(ctx, phone, "register"); err != nil || st.calls != 1 {
		t.Fatalf("send ok: %v, calls=%d", err, st.calls)
	}
	st.err = fmt.Errorf("boom")
	if err := ps.IssueSms(ctx, phone, "reset"); err == nil || st.calls != 2 {
		t.Fatalf("send fail: %v, calls=%d", err, st.calls)
	}
	// 已取消 ctx:ConsumeSms Exec 错误分支。
	if _, err := s.ConsumeSms(cctx, phone, "login", "123456"); err == nil {
		t.Fatal("ConsumeSms cancelled: want error")
	}
	pool.Exec(ctx, `DELETE FROM portal_sms_codes WHERE phone=$1`, phone)
}

// TestPGAccounts 账号 upsert/双查/验密/换绑闭环。
func TestPGAccounts(t *testing.T) {
	s := openPGTest(t)
	pool := s.(*pgStore).pool
	ctx := context.Background()
	const phone, phone2 = "+99900000002", "+99900000003"
	const cid int64 = -888002
	pool.Exec(ctx, `DELETE FROM portal_accounts WHERE customer_id=$1 OR phone IN ($2,$3)`, cid, phone, phone2)

	// 超长密码触发 bcrypt 错误分支。
	if _, err := s.UpsertAccount(ctx, phone, strings.Repeat("a", 100), cid); err == nil {
		t.Fatal("long password: want bcrypt error")
	}
	a, err := s.UpsertAccount(ctx, phone, "secret", cid)
	if err != nil || a.Phone != phone || a.CustomerID != cid || a.PasswordHash == "" {
		t.Fatalf("UpsertAccount: %+v,%v", a, err)
	}
	// 超长手机号触发 Exec 错误分支。
	if _, err := s.UpsertAccount(ctx, strings.Repeat("9", 33), "p", cid); err == nil {
		t.Fatal("long phone: want exec error")
	}
	if a, err = s.AccountByPhone(ctx, phone); err != nil || a.CustomerID != cid {
		t.Fatalf("AccountByPhone: %+v,%v", a, err)
	}
	if _, err = s.AccountByPhone(ctx, "+99900000009"); err != ErrNotFound {
		t.Fatalf("AccountByPhone miss = %v", err)
	}
	if a, err = s.AccountByCustomer(ctx, cid); err != nil || a.Phone != phone {
		t.Fatalf("AccountByCustomer: %+v,%v", a, err)
	}
	if _, err = s.AccountByCustomer(ctx, -888099); err != ErrNotFound {
		t.Fatalf("AccountByCustomer miss = %v", err)
	}
	for _, tc := range []struct {
		phone, pass string
		want        bool
	}{
		{phone, "secret", true},
		{phone, "wrong", false},
		{"+99900000009", "x", false},
	} {
		if ok, err := s.VerifyPassword(ctx, tc.phone, tc.pass); err != nil || ok != tc.want {
			t.Fatalf("VerifyPassword(%s)=%v,%v", tc.phone, ok, err)
		}
	}
	if err := s.RebindPhone(ctx, cid, phone2); err != nil {
		t.Fatalf("RebindPhone: %v", err)
	}
	if err := s.RebindPhone(ctx, -888099, "+99900000008"); err != ErrNotFound {
		t.Fatalf("RebindPhone miss = %v", err)
	}
	// 已取消 ctx 触发 AccountByPhone/AccountByCustomer/RebindPhone 错误分支。
	cctx, cancel := cancelledCtx()
	defer cancel()
	if _, err := s.AccountByPhone(cctx, phone); err == nil || err == ErrNotFound {
		t.Fatal("AccountByPhone cancelled: want error")
	}
	if _, err := s.AccountByCustomer(cctx, cid); err == nil || err == ErrNotFound {
		t.Fatal("AccountByCustomer cancelled: want error")
	}
	if err := s.RebindPhone(cctx, cid, phone); err == nil {
		t.Fatal("RebindPhone cancelled: want error")
	}
	if _, err := s.VerifyPassword(cctx, phone, "secret"); err == nil {
		t.Fatal("VerifyPassword cancelled: want error")
	}
	pool.Exec(ctx, `DELETE FROM portal_accounts WHERE customer_id=$1`, cid)
}

// TestPGSeq 合成 ID 首次补行/后续自增与未知单号错误。
func TestPGSeq(t *testing.T) {
	s := openPGTest(t)
	pool := s.(*pgStore).pool
	ctx := context.Background()

	// NextNo:正常自增 + 未知 kind + 已取消 ctx 错误。
	no, err := s.NextNo(ctx, "PAY")
	if err != nil || !strings.HasPrefix(no, "PAY-") {
		t.Fatalf("NextNo: %v,%v", no, err)
	}
	if _, err := s.NextNo(ctx, "NOPE"); err == nil {
		t.Fatal("unknown kind: want error")
	}
	cctx, cancel := cancelledCtx()
	defer cancel()
	if _, err := s.NextNo(cctx, "PAY"); err == nil {
		t.Fatal("NextNo cancelled: want error")
	}

	// 删掉 CUST 行触发 errNoRows 首次插入分支,随后走 UPDATE 分支。
	pool.Exec(ctx, `DELETE FROM portal_seq WHERE kind='CUST'`)
	id, err := s.NextSyntheticCustomerID(ctx)
	if err != nil || id != -1 {
		t.Fatalf("first synthetic id: %v,%v", id, err)
	}
	if id, err = s.NextSyntheticCustomerID(ctx); err != nil || id != -2 {
		t.Fatalf("second synthetic id: %v,%v", id, err)
	}
	if _, err := s.NextSyntheticCustomerID(cctx); err == nil {
		t.Fatal("synthetic id cancelled: want error")
	}
	// BEFORE INSERT 触发器让首次补行 INSERT 失败,覆盖 errNoRows 后 Exec 错误分支。
	if _, err := pool.Exec(ctx, `CREATE OR REPLACE FUNCTION portal_seq_block_cust() RETURNS trigger AS $$
		BEGIN IF NEW.kind='CUST' THEN RAISE EXCEPTION 'blocked'; END IF; RETURN NEW; END $$ LANGUAGE plpgsql`); err != nil {
		t.Fatalf("create fn: %v", err)
	}
	pool.Exec(ctx, `DROP TRIGGER IF EXISTS portal_seq_block ON portal_seq`)
	if _, err := pool.Exec(ctx, `CREATE TRIGGER portal_seq_block BEFORE INSERT ON portal_seq
		FOR EACH ROW EXECUTE FUNCTION portal_seq_block_cust()`); err != nil {
		t.Fatalf("create trigger: %v", err)
	}
	pool.Exec(ctx, `DELETE FROM portal_seq WHERE kind='CUST'`)
	if _, err := s.NextSyntheticCustomerID(ctx); err == nil {
		t.Fatal("blocked insert: want error")
	}
	pool.Exec(ctx, `DROP TRIGGER IF EXISTS portal_seq_block ON portal_seq;
		DROP FUNCTION IF EXISTS portal_seq_block_cust()`)
	pool.Exec(ctx, `DELETE FROM portal_seq WHERE kind='CUST'`)
	pool.Exec(ctx, `INSERT INTO portal_seq(kind, last) VALUES ('CUST', 0)`)
}

// TestRandDigits 纯函数:6 位数字串;注入故障 reader 覆盖错误分支。
type errReader struct{}

func (errReader) Read([]byte) (int, error) { return 0, fmt.Errorf("boom") }

func TestRandDigits(t *testing.T) {
	got, err := randDigits(6)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 6 {
		t.Fatalf("len=%d", len(got))
	}
	for _, c := range got {
		if c < '0' || c > '9' {
			t.Fatalf("non-digit %q", c)
		}
	}
	old := randReader
	randReader = errReader{}
	defer func() { randReader = old }()
	if _, err := randDigits(6); err == nil {
		t.Fatal("failing reader: want error")
	}
}
