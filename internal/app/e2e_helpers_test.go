package app_test

// e2e 辅助:种子、登录、HTTP 调用与审计轮询;主流程见 e2e_pg_integration_test.go。

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"golang.org/x/crypto/bcrypt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/events"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/test/bufconn"
	"net"
)

// 唯一性:username/路径/编码带纳秒后缀,可重复运行不冲突。
type capPub struct {
	events.Publisher
	got []events.Event
}

func seedE2E(t *testing.T, ctx context.Context, pool *pgxpool.Pool, a *app.Application) *e2eSeed {
	t.Helper()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1e12)
	s := &e2eSeed{username: "e2e-" + suffix, password: "E2e-pass-123"}
	registerE2ECleanup(t, ctx, pool, s, suffix)

	hash, err := bcrypt.GenerateFromPassword([]byte(s.password), bcrypt.MinCost)
	if err != nil {
		t.Fatal(err)
	}
	// 管理员账号挂 sysadmin 角色(种子已授全量菜单权限)。
	_, err = pool.Exec(ctx,
		`INSERT INTO accounts(username, password_hash, real_name, role_id)
		 SELECT $1, $2, 'E2E管理员', id FROM roles WHERE code='sysadmin'`, s.username, string(hash))
	if err != nil {
		t.Fatalf("seed account: %v", err)
	}

	// 地址(level=1 市级,与 nlevel 一致)。
	err = pool.QueryRow(ctx,
		`INSERT INTO addresses(path, level, name) VALUES($1, 1, $2) RETURNING id`,
		"e2e"+suffix, "E2E测试市").Scan(&s.addressID)
	if err != nil {
		t.Fatalf("seed address: %v", err)
	}
	// 经营区域快照取种子省节点。
	err = pool.QueryRow(ctx,
		`SELECT id, name FROM regions WHERE path='root.luzon.ncr.manila'`).Scan(&s.regionID, &s.regionName)
	if err != nil {
		t.Fatalf("seed region: %v", err)
	}

	// 设备(分光器)+ 2 个空闲端口(一个走全流程,一个走取消释放)。
	resID, err := a.Resource.CreateResource(ctx, resSeed(s.addressID, suffix))
	if err != nil {
		t.Fatalf("seed resource: %v", err)
	}
	for i := 1; i <= 2; i++ {
		s.portID, err = a.Resource.CreatePort(ctx, portSeed(resID, s, suffix, i))
		if err != nil {
			t.Fatalf("seed port: %v", err)
		}
	}

	s.channelID, err = a.Channel.CreateChannel(ctx, chanSeed(suffix))
	if err != nil {
		t.Fatalf("seed channel: %v", err)
	}
	s.offerID, err = a.Product.CreateProduct(ctx, offerSeed(suffix))
	if err != nil {
		t.Fatalf("seed offer: %v", err)
	}
	s.customerID, err = a.Customer.Create(ctx, custSeed(s))
	if err != nil {
		t.Fatalf("seed customer: %v", err)
	}
	return s
}

// loginE2E 登录换 token(真实 JWT 签发 + bcrypt 校验)。
func loginE2E(t *testing.T, ts *httptest.Server, s *e2eSeed) string {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"username": s.username, "password": s.password})
	resp, err := http.Post(ts.URL+"/api/admin/v1/auth/login", "application/json", bytes.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out struct {
		Code int `json:"code"`
		Data struct {
			Token string `json:"token"`
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Code != 0 || out.Data.Token == "" {
		t.Fatalf("login code=%d token=%q", out.Code, out.Data.Token)
	}
	return out.Data.Token
}

// submitOrderE2E 经 HTTP 下单,返回订单 id/orderNo。
func submitOrderE2E(t *testing.T, ts *httptest.Server, token string, s *e2eSeed) (int64, string) {
	t.Helper()
	body, _ := json.Marshal(map[string]any{
		"CustomerID": s.customerID, "OfferID": s.offerID, "AddressID": s.addressID,
		"ChannelID": s.channelID, "LegalEntityID": 1, "RegionPath": "root.luzon.ncr.manila",
	})
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/admin/v1/orders", bytes.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out struct {
		Code int `json:"code"`
		Data struct {
			ID      int64  `json:"id"`
			OrderNo string `json:"orderNo"`
		}
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Code != 0 || out.Data.ID == 0 {
		t.Fatalf("submit code=%d data=%+v", out.Code, out.Data)
	}
	return out.Data.ID, out.Data.OrderNo
}

// postOK 发 POST 并断言 code=0;field 非空时从 data 中取该字段返回。
func postOK(t *testing.T, ts *httptest.Server, token, path, body, field string) string {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, ts.URL+path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var out struct {
		Code int            `json:"code"`
		Data map[string]any `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
		t.Fatal(err)
	}
	if out.Code != 0 {
		t.Fatalf("POST %s code=%d", path, out.Code)
	}
	if field == "" {
		return ""
	}
	v, _ := out.Data[field].(string)
	return v
}

// waitAudit 轮询审计落库(异步写,尽力而为):下单动作必须留痕。
func waitAudit(t *testing.T, ctx context.Context, pool *pgxpool.Pool, orderNo string) {
	t.Helper()
	for i := 0; i < 50; i++ {
		var n int
		if err := pool.QueryRow(ctx,
			`SELECT count(*) FROM audit_logs WHERE target_type='order' AND target_id=$1`, orderNo).Scan(&n); err != nil {
			t.Fatal(err)
		}
		if n > 0 {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("订单 %s 无审计留痕", orderNo)
}

// bufconn 辅助与 package app 单测同形(app_test 与 package app 不能共享未导出符号)。
const bufSize = 1024

func newBufConnServer(t *testing.T, register func(*grpc.Server)) *grpc.ClientConn {
	t.Helper()
	lis := bufconn.Listen(bufSize)
	srv := grpc.NewServer()
	register(srv)
	go func() { _ = srv.Serve(lis) }()
	t.Cleanup(srv.Stop)

	conn, err := grpc.NewClient("passthrough:///bufnet",
		grpc.WithContextDialer(func(ctx context.Context, _ string) (net.Conn, error) {
			return lis.DialContext(ctx)
		}),
		grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		t.Fatalf("grpc dial: %v", err)
	}
	t.Cleanup(func() { _ = conn.Close() })
	return conn
}
