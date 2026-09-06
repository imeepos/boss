package aaa

// AAA-A5 per-NAS 注册表 PG 实现回归:未注册/停用/解密失败 fail-closed,CRUD 与唯一约束。

import (
	"context"
	"errors"
	"log"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/pashagolub/pgxmock/v4"

	"github.com/ymm-001/boss/internal/domain/aaa/credential"
)

func newNasMock(t *testing.T) (pgxmock.PgxPoolIface, *PGStore, *credential.Codec) {
	t.Helper()
	mock, err := pgxmock.NewPool()
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { mock.Close() })
	codec, err := credential.New("unit-test-material-32")
	if err != nil {
		t.Fatal(err)
	}
	return mock, NewPGStore(mock).WithCredentialCodec(codec), codec
}

const nasColsTest = "id, name, nas_ip, vendor, coa_port, enabled, created_at, updated_at, secret_enc"

// nasRow 命中行(enabled 可控,enc 为密文)。
func nasRow(mock pgxmock.PgxPoolIface, enabled bool, enc string) *pgxmock.Rows {
	return mock.NewRows(strings.Split(nasColsTest, ", ")).
		AddRow(int64(7), "OLT-01", "10.0.0.9", "HUAWEI", 3799, enabled, ts, ts, enc)
}

func TestLookupNasAcceptsRegistered(t *testing.T) {
	mock, s, codec := newNasMock(t)
	enc, err := codec.Encode("nas-secret-plain")
	if err != nil {
		t.Fatal(err)
	}
	mock.ExpectQuery("SELECT id, name, nas_ip").WithArgs("10.0.0.9").WillReturnRows(nasRow(mock, true, enc))
	nas, err := s.LookupNas(context.Background(), "10.0.0.9")
	if err != nil {
		t.Fatalf("lookup: %v", err)
	}
	if string(nas.Secret) != "nas-secret-plain" || nas.Client.ID != 7 || nas.Client.Vendor != NasVendorHuawei || nas.Client.CoAPort != 3799 {
		t.Fatalf("nas=%+v secret=%q", nas.Client, nas.Secret)
	}
	if err := mock.ExpectationsWereMet(); err != nil {
		t.Fatal(err)
	}
}

func TestLookupNasRejects(t *testing.T) {
	mock, s, codec := newNasMock(t)
	enc, _ := codec.Encode("nas-secret-plain")
	ctx := context.Background()

	t.Run("未注册 ErrNasNotFound", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, name, nas_ip").WithArgs("10.9.9.9").WillReturnError(pgx.ErrNoRows)
		if _, err := s.LookupNas(ctx, "10.9.9.9"); !errors.Is(err, ErrNasNotFound) {
			t.Fatalf("err=%v want ErrNasNotFound", err)
		}
	})
	t.Run("停用 ErrNasDisabled", func(t *testing.T) {
		mock.ExpectQuery("SELECT id, name, nas_ip").WithArgs("10.0.0.9").WillReturnRows(nasRow(mock, false, enc))
		if _, err := s.LookupNas(ctx, "10.0.0.9"); !errors.Is(err, ErrNasDisabled) {
			t.Fatalf("err=%v want ErrNasDisabled", err)
		}
	})
	t.Run("密文损坏 fail-closed 并留痕", func(t *testing.T) {
		buf := captureStdLog(t)
		mock.ExpectQuery("SELECT id, name, nas_ip").WithArgs("10.0.0.9").WillReturnRows(nasRow(mock, true, "v1$gcm$broken$broken"))
		if _, err := s.LookupNas(ctx, "10.0.0.9"); err == nil || errors.Is(err, ErrNasNotFound) || errors.Is(err, ErrNasDisabled) {
			t.Fatalf("err=%v want decrypt fail", err)
		}
		if !strings.Contains(buf.String(), "[aaa] NAS SECRET DECRYPT FAILED") {
			t.Fatalf("log=%s", buf.String())
		}
	})
}

func TestCreateNasEncodesAndDedups(t *testing.T) {
	mock, s, _ := newNasMock(t)
	enabled := true
	u := NasUpsert{Name: "OLT-02", NasIP: "10.0.1.1", SecretPlain: "pw", Vendor: NasVendorZTE, Enabled: &enabled}
	mock.ExpectQuery("INSERT INTO aaa_nas_clients").
		WithArgs("OLT-02", "10.0.1.1", pgxmock.AnyArg(), "ZTE", 3799, true).
		WillReturnRows(mock.NewRows([]string{"id"}).AddRow(int64(9)))
	id, err := s.CreateNas(context.Background(), u)
	if err != nil || id != 9 {
		t.Fatalf("id=%d err=%v", id, err)
	}
	mock.ExpectQuery("INSERT INTO aaa_nas_clients").
		WithArgs(pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg(), pgxmock.AnyArg()).
		WillReturnError(&pgconn.PgError{Code: "23505", Message: "duplicate key"})
	if _, err := s.CreateNas(context.Background(), u); !errors.Is(err, ErrNasDuplicate) {
		t.Fatalf("err=%v want ErrNasDuplicate", err)
	}
	if _, err := s.CreateNas(context.Background(), NasUpsert{Name: "x", NasIP: "10.0.1.2"}); err == nil || !strings.Contains(err.Error(), "secret required") {
		t.Fatalf("create 缺密钥应拒绝: %v", err)
	}
}

func TestUpdateNasWithAndWithoutSecret(t *testing.T) {
	mock, s, _ := newNasMock(t)
	disabled := false
	mock.ExpectExec("UPDATE aaa_nas_clients SET").
		WithArgs(int64(7), "OLT-01", "10.0.0.9", "GENERIC", 3799, false).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	if err := s.UpdateNas(context.Background(), 7, NasUpsert{Name: "OLT-01", NasIP: "10.0.0.9", Enabled: &disabled}); err != nil {
		t.Fatalf("update 无密钥: %v", err)
	}
	mock.ExpectExec("UPDATE aaa_nas_clients SET").
		WithArgs(int64(7), "OLT-01", "10.0.0.9", "GENERIC", 3799, false, pgxmock.AnyArg()).
		WillReturnResult(pgxmock.NewResult("UPDATE", 1))
	if err := s.UpdateNas(context.Background(), 7, NasUpsert{Name: "OLT-01", NasIP: "10.0.0.9", SecretPlain: "newpw", Enabled: &disabled}); err != nil {
		t.Fatalf("update 换密钥: %v", err)
	}
	mock.ExpectExec("UPDATE aaa_nas_clients SET").
		WithArgs(int64(99), "x", "10.0.0.1", "GENERIC", 3799, true).
		WillReturnResult(pgxmock.NewResult("UPDATE", 0))
	if err := s.UpdateNas(context.Background(), 99, NasUpsert{Name: "x", NasIP: "10.0.0.1"}); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v want ErrNotFound", err)
	}
}

func TestDeleteAndGetNas(t *testing.T) {
	mock, s, _ := newNasMock(t)
	ctx := context.Background()
	mock.ExpectExec("DELETE FROM aaa_nas_clients").WithArgs(int64(7)).WillReturnResult(pgxmock.NewResult("DELETE", 1))
	if err := s.DeleteNas(ctx, 7); err != nil {
		t.Fatal(err)
	}
	mock.ExpectExec("DELETE FROM aaa_nas_clients").WithArgs(int64(7)).WillReturnResult(pgxmock.NewResult("DELETE", 0))
	if err := s.DeleteNas(ctx, 7); !errors.Is(err, ErrNotFound) {
		t.Fatalf("err=%v", err)
	}
	cols := strings.Split("id, name, nas_ip, vendor, coa_port, enabled, created_at, updated_at", ", ")
	mock.ExpectQuery("SELECT id, name, nas_ip").WithArgs(int64(7)).
		WillReturnRows(mock.NewRows(cols).AddRow(int64(7), "OLT-01", "10.0.0.9", "HUAWEI", 3799, true, ts, ts))
	nas, err := s.GetNas(ctx, 7)
	if err != nil || nas.Name != "OLT-01" {
		t.Fatalf("nas=%+v err=%v", nas, err)
	}
}

func TestListNasPageFilters(t *testing.T) {
	mock, s, _ := newNasMock(t)
	cols := strings.Split("id, name, nas_ip, vendor, coa_port, enabled, created_at, updated_at", ", ")
	mock.ExpectQuery("SELECT COUNT").WithArgs("%HW%", "HUAWEI", true).
		WillReturnRows(mock.NewRows([]string{"count"}).AddRow(1))
	mock.ExpectQuery("SELECT id, name, nas_ip").WithArgs("%HW%", "HUAWEI", true, 20, 0).
		WillReturnRows(mock.NewRows(cols).AddRow(int64(7), "OLT-HW", "10.0.0.9", "HUAWEI", 3799, true, ts, ts))
	got, err := s.ListNasPage(context.Background(), NasPage{Page: 1, PageSize: 20, Keyword: "HW", Vendor: "HUAWEI", Enabled: "true"})
	if err != nil {
		t.Fatal(err)
	}
	if got.Total != 1 || len(got.Items) != 1 || got.Items[0].Vendor != NasVendorHuawei {
		t.Fatalf("got=%+v", got)
	}
}

// logBuf 线程安全日志缓冲(异步 goroutine 写日志时并发断言读,race 检测要求互斥)。
type logBuf struct {
	mu  sync.Mutex
	buf strings.Builder
}

func (b *logBuf) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *logBuf) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// captureStdLog 捕获标准 log 输出(留痕断言用)。
func captureStdLog(t *testing.T) *logBuf {
	t.Helper()
	buf := &logBuf{}
	log.SetOutput(buf)
	t.Cleanup(func() { log.SetOutput(os.Stderr) })
	return buf
}
