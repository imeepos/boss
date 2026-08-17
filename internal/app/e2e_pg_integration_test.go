package app

// W4 端到端集成测试(真实 PG):一期验收 = 下单→激活全流程可走通 / 可取消 / 端口释放 / 全程留痕。
// 运行: BOSS_PG_TEST_DSN="host=192.168.0.102 port=25432 user=boss password=boss dbname=boss sslmode=disable" go test ./internal/app/ -run TestE2E -v -count=1

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/domain/quadlink"
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

	t.Run("W5_扫码闭环_一致推进_不一致拒_拆机必扫码", func(t *testing.T) {
		orderID, _ := submitOrderE2E(t, ts, token, s)
		if _, err := a.Resource.ReserveFirstAvailable(ctx, s.addressID, orderID); err != nil {
			t.Fatal(err) // 环节3 预占落地(advance 只推环节/状态)
		}
		for _, step := range []func(context.Context, int64) error{
			a.Order.CheckResource, a.Order.Reserve, a.Order.ChargeContract, a.Order.ApplyTag,
			a.Order.CreateUserProfile, a.Order.PreConfigOLT, a.Order.DispatchOrder,
		} {
			if err := step(ctx, orderID); err != nil {
				t.Fatal(err)
			}
		}
		// 预绑定:建批次+资产+标签(EPC)+四码(UNLINKED)。
		batchID, err := a.Asset.CreateBatch(ctx, asset.AssetBatch{
			LegalEntityID: 1, Code: "RK-E2E-" + orderNo6(orderID), Name: "E2E批次",
		})
		if err != nil {
			t.Fatal(err)
		}
		assetID, err := a.Asset.CreateAsset(ctx, asset.Asset{
			LegalEntityID: 1, LegalEntityName: "主品牌·企业", BatchID: batchID,
			AssetCode: "A-E2E-" + orderNo6(orderID), Type: "ONU", Status: "IN_STOCK",
		})
		if err != nil {
			t.Fatal(err)
		}
		epc := "E2E-SCAN-" + orderNo6(orderID)
		tagID, err := a.Asset.CreateTag(ctx, asset.Tag{
			LegalEntityID: 1, TagNo: "T-E2E-" + orderNo6(orderID), EpcCode: epc, Band: "UHF",
			BoundAssetID: assetID, Status: "BOUND", Battery: "OK",
		})
		if err != nil {
			t.Fatal(err)
		}
		_ = tagID
		var portID int64
		if err := pool.QueryRow(ctx, `SELECT id FROM ports WHERE order_id=$1`, orderID).Scan(&portID); err != nil {
			t.Fatal(err)
		}
		if _, err := a.QuadLink.CreateLink(ctx, quadlink.QuadLink{
			AssetID: assetID, CustomerID: customerIDOf(t, ctx, pool, orderID), PortID: portID,
			AddressID: s.addressID, LegalEntityID: 1, LegalEntityName: "主品牌·企业", Status: "UNLINKED",
		}); err != nil {
			t.Fatal(err)
		}
		// 不一致 → 拒(40920),环节不推进。
		_, body := scanPost(t, ts, token, "/api/v1/tickets/TIC-X/scan-bind", `{"epc":"EPC-WRONG"}`)
		if !strings.Contains(body, `"code":40920`) && !strings.Contains(body, `"code":40400`) {
			t.Fatalf("body=%s", body)
		}
		// 工单 + 一致扫码 → MATCH 推进环节9。
		tno := "TIC-E2E-" + orderNo6(orderID)
		if _, err := pool.Exec(ctx,
			`INSERT INTO dispatch_tickets(ticket_no, order_id, legal_entity_id, legal_entity_name, status)
			 VALUES($1,$2,1,'主品牌·企业','DOING')`, tno, orderID); err != nil {
			t.Fatal(err)
		}
		_, body = scanPost(t, ts, token, "/api/v1/tickets/"+tno+"/scan-bind", `{"epc":"`+epc+`"}`)
		if !strings.Contains(body, `"MATCH"`) {
			t.Fatalf("scan-bind body=%s", body)
		}
		o, _, err := a.Order.Track(ctx, orderID)
		if err != nil {
			t.Fatal(err)
		}
		if o.Stage != 9 {
			t.Fatalf("stage=%d, want 9(扫码后推进)", o.Stage)
		}
		// 拆机不扫码 → 拦截(42200)。
		_, body = scanPost(t, ts, token, "/api/v1/tickets/"+tno+"/dismantle/scan", `{"epc":""}`)
		if !strings.Contains(body, `"code":42200`) {
			t.Fatalf("dismantle body=%s", body)
		}
		// 对账任务可执行。
		rep, err := a.QuadLink.Reconcile(ctx)
		if err != nil {
			t.Fatal(err)
		}
		if rep.Total < 1 {
			t.Fatalf("reconcile rep=%+v", rep)
		}
	})
}

// orderNo6 订单 id → 6 位尾(测试单号唯一化)。
func orderNo6(id int64) string { return fmt.Sprintf("%06d", id%1000000) }

// customerIDOf 订单 → 客户。
func customerIDOf(t *testing.T, ctx context.Context, pool *pgxpool.Pool, orderID int64) int64 {
	t.Helper()
	var cid int64
	if err := pool.QueryRow(ctx, `SELECT customer_id FROM orders WHERE id=$1`, orderID).Scan(&cid); err != nil {
		t.Fatal(err)
	}
	return cid
}

// scanPost 带登录态的 POST,返回 body 文本。
func scanPost(t *testing.T, ts *httptest.Server, token, path, body string) (int, string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodPost, ts.URL+path, strings.NewReader(body))
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}
