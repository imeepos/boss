package app

// W4 端到端集成测试(真实 PG):一期验收 = 下单→激活全流程可走通 / 可取消 / 端口释放 / 全程留痕。
// 运行: BOSS_PG_TEST_DSN="host=192.168.0.102 port=25432 user=boss password=boss dbname=boss sslmode=disable" \
//
//	go test ./internal/app/ -run TestE2E_OrderLifecycle_Integration -v -count=1

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"golang.org/x/crypto/bcrypt"

	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/config"
	"github.com/ymm-001/boss/internal/pkg/database"
)

type e2eSeed struct {
	username, password string
	addressID          int64
	regionID           int64
	regionName         string
	customerID         int64
	offerID            int64
	channelID          int64
	portID             int64
}

// seedE2E 准备一次性种子:管理员账号(sysadmin)+ 地址 + 设备/端口 + 渠道 + 产品 + 客户。
// 唯一性:username/路径/编码带纳秒后缀,可重复运行不冲突。
func seedE2E(t *testing.T, ctx context.Context, pool *pgxpool.Pool, a *Application) *e2eSeed {
	t.Helper()
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1e12)
	s := &e2eSeed{username: "e2e-" + suffix, password: "E2e-pass-123"}

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
	resp, err := http.Post(ts.URL+"/api/v1/auth/login", "application/json", bytes.NewReader(body))
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
	req, _ := http.NewRequest(http.MethodPost, ts.URL+"/api/v1/orders", bytes.NewReader(body))
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

// TestE2E_OrderLifecycle_Integration 主流程:登录→HTTP下单→12环节推进到激活→审计留痕;并行验证取消→端口释放。
func TestE2E_OrderLifecycle_Integration(t *testing.T) {
	dsn := os.Getenv("BOSS_PG_TEST_DSN")
	if dsn == "" {
		t.Skip("BOSS_PG_TEST_DSN 未设置,跳过集成测试")
	}
	ctx := context.Background()
	gin.SetMode(gin.TestMode)

	cfg := &config.Config{}
	cfg.Database.DSN = dsn
	a, err := New(ctx, cfg, "../../migrations")
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}
	defer a.Close()

	pool, err := database.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	defer pool.Close()

	s := seedE2E(t, ctx, pool, a)

	r := gin.New()
	RegisterRoutes(r, a, auth.NewManager("e2e-secret", time.Hour))
	ts := httptest.NewServer(r)
	defer ts.Close()

	token := loginE2E(t, ts, s)

	t.Run("全流程_下单到激活_12环节", func(t *testing.T) {
		orderID, orderNo := submitOrderE2E(t, ts, token, s)
		steps := []struct {
			name string
			run  func(context.Context, int64) error
		}{
			{"checkResource", a.Order.CheckResource}, {"reservePort", a.Order.Reserve},
			{"chargeContract", a.Order.ChargeContract}, {"applyTag", a.Order.ApplyTag},
			{"createUserProfile", a.Order.CreateUserProfile}, {"preConfigOLT", a.Order.PreConfigOLT},
			{"dispatchOrder", a.Order.DispatchOrder}, {"scanBind", a.Order.ScanBind},
			{"activateUser", a.Order.ActivateUser}, {"notifyActivation", a.Order.NotifyActivation},
			{"updateMap", a.Order.UpdateMap},
		}
		for i, step := range steps {
			if err := step.run(ctx, orderID); err != nil {
				t.Fatalf("环节%s(%d/12) 失败: %v", step.name, i+2, err)
			}
		}
		o, _, err := a.Order.Track(ctx, orderID)
		if err != nil {
			t.Fatal(err)
		}
		if o.Stage != 12 || o.Status != "DONE" {
			t.Fatalf("stage=%d status=%s, want 12/DONE", o.Stage, o.Status)
		}
		waitAudit(t, ctx, pool, orderNo)
	})

	t.Run("取消订单_端口释放", func(t *testing.T) {
		orderID, _ := submitOrderE2E(t, ts, token, s)
		if err := a.Order.CheckResource(ctx, orderID); err != nil {
			t.Fatal(err)
		}
		portID, err := a.Resource.ReserveFirstAvailable(ctx, s.addressID, orderID)
		if err != nil {
			t.Fatalf("预占端口: %v", err)
		}
		if err := a.Order.Cancel(ctx, orderID); err != nil {
			t.Fatalf("取消: %v", err)
		}
		if err := a.Resource.ReleasePortByOrder(ctx, orderID); err != nil {
			t.Fatalf("端口释放: %v", err)
		}
		var status string
		if err := pool.QueryRow(ctx, `SELECT status FROM ports WHERE id=$1`, portID).Scan(&status); err != nil {
			t.Fatal(err)
		}
		if status != "IDLE" {
			t.Fatalf("端口释放后 status=%s, want IDLE", status)
		}
	})
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
