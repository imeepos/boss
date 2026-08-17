package app

// W4 端到端集成测试(真实 PG):一期验收 = 下单→激活全流程可走通 / 可取消 / 端口释放 / 全程留痕。
// 运行: BOSS_PG_TEST_DSN="host=192.168.0.102 port=25432 user=boss password=boss dbname=boss sslmode=disable" go test ./internal/app/ -run TestE2E -v -count=1

import (
	"context"
	"fmt"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/resource"
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

	t.Run("台账写侧_调拨审批_释放预占", func(t *testing.T) {
		// 调拨:建单→审批→驳回第二条。
		no1 := postOK(t, ts, token, "/api/v1/transfers",
			fmt.Sprintf(`{"resourceId":1,"legalEntityId":1,"legalEntityName":"主品牌","fromRegionId":11,"toRegionId":13}`), "transferNo")
		postOK(t, ts, token, "/api/v1/transfers/"+no1+"/approve", "", "")
		no2 := postOK(t, ts, token, "/api/v1/transfers",
			`{"resourceId":1,"legalEntityId":1,"legalEntityName":"主品牌","fromRegionId":11,"toRegionId":14}`, "transferNo")
		postOK(t, ts, token, "/api/v1/transfers/"+no2+"/reject", "", "")
		for _, tc := range []struct{ no, want string }{{no1, "DOING"}, {no2, "DONE"}} {
			var st string
			if err := pool.QueryRow(ctx, `SELECT status FROM transfers WHERE transfer_no=$1`, tc.no).Scan(&st); err != nil {
				t.Fatal(err)
			}
			if st != tc.want {
				t.Fatalf("调拨 %s status=%s, want %s", tc.no, st, tc.want)
			}
		}

		// 手动释放预占:预占端口+写 HELD 记录 → HTTP 释放 → 端口 IDLE、记录 RELEASED。
		orderID, _ := submitOrderE2E(t, ts, token, s)
		portID, err := a.Resource.ReserveFirstAvailable(ctx, s.addressID, orderID)
		if err != nil {
			t.Fatal(err)
		}
		recID, err := a.ResourceSub.AppendReserveRecord(ctx, resource.ReserveRecord{PortID: portID, OrderID: orderID, Status: "HELD"})
		if err != nil {
			t.Fatal(err)
		}
		postOK(t, ts, token, fmt.Sprintf("/api/v1/reserves/%d/release", recID), "", "")
		var portSt, recSt string
		if err := pool.QueryRow(ctx, `SELECT status FROM ports WHERE id=$1`, portID).Scan(&portSt); err != nil {
			t.Fatal(err)
		}
		if err := pool.QueryRow(ctx, `SELECT status FROM reserve_records WHERE id=$1`, recID).Scan(&recSt); err != nil {
			t.Fatal(err)
		}
		if portSt != "IDLE" || recSt != "RELEASED" {
			t.Fatalf("port=%s record=%s, want IDLE/RELEASED", portSt, recSt)
		}
	})
}
