package app_test

// W4 端到端集成测试(真实 PG):一期验收 = 下单→激活全流程可走通 / 可取消 / 端口释放 / 全程留痕。
// 运行: BOSS_PG_TEST_DSN="host=192.168.0.102 port=25432 user=boss password=boss dbname=boss sslmode=disable" go test ./internal/app/ -run TestE2E -v -count=1

import (
	"context"
	"errors"
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

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/aaa"
	aaabilling "github.com/ymm-001/boss/internal/domain/aaa/billing"
	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/domain/device"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/provision"
	"github.com/ymm-001/boss/internal/domain/quadlink"
	"github.com/ymm-001/boss/internal/domain/resource"
	"github.com/ymm-001/boss/internal/httpapi"
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
	a, err := app.New(ctx, cfg, "../../migrations")
	if err != nil {
		t.Fatalf("app.New: %v", err)
	}
	defer a.Close()

	pool, err := database.Open(ctx, dsn)
	if err != nil {
		t.Fatal(err)
	}
	// t.Cleanup 而非 defer:seedE2E 注册的数据自清理需在池关闭前执行(LIFO)。
	t.Cleanup(pool.Close)

	s := seedE2E(t, ctx, pool, a)

	r := gin.New()
	httpapi.RegisterRoutes(r, a, auth.NewManager("e2e-secret", time.Hour))
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
			// 环节5(applyTag)后验证 quad_link 自动创建。
			if step.name == "applyTag" {
				ql, qlErr := a.QuadLink.GetByCustomer(ctx, s.customerID)
				if qlErr != nil {
					t.Fatalf("applyTag 后 quad_link 未创建: %v", qlErr)
				}
				if ql.PortID == 0 {
					t.Fatalf("applyTag 后 quad_link port_id=0,应该有值")
				}
				if ql.Status != "UNLINKED" {
					t.Fatalf("applyTag 后 quad_link status=%s, want UNLINKED", ql.Status)
				}
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
		no1 := postOK(t, ts, token, "/api/admin/v1/transfers",
			fmt.Sprintf(`{"resourceId":1,"legalEntityId":1,"legalEntityName":"主品牌","fromRegionId":11,"toRegionId":13}`), "transferNo")
		postOK(t, ts, token, "/api/admin/v1/transfers/"+no1+"/approve", "", "")
		no2 := postOK(t, ts, token, "/api/admin/v1/transfers",
			`{"resourceId":1,"legalEntityId":1,"legalEntityName":"主品牌","fromRegionId":11,"toRegionId":14}`, "transferNo")
		postOK(t, ts, token, "/api/admin/v1/transfers/"+no2+"/reject", "", "")
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
		postOK(t, ts, token, fmt.Sprintf("/api/admin/v1/reserves/%d/release", recID), "", "")
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
		_, body := scanPost(t, ts, token, "/api/admin/v1/tickets/TIC-X/scan-bind", `{"epc":"EPC-WRONG"}`)
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
		_, body = scanPost(t, ts, token, "/api/admin/v1/tickets/"+tno+"/scan-bind", `{"epc":"`+epc+`"}`)
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
		_, body = scanPost(t, ts, token, "/api/admin/v1/tickets/"+tno+"/dismantle/scan", `{"epc":""}`)
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

	t.Run("W6_AAA_停复机即时生效_话单入账", func(t *testing.T) {
		authz := aaa.NewPGAuthorizer(pool)
		customerID, err := a.Customer.Create(ctx, customer.Customer{
			Name: "E2E-AAA客户", Phone: "09171111111", IdType: "身份证", IdNo: "E2E-AAA",
			RealNameStatus: "VERIFIED", ServiceStatus: "ACTIVE",
			AddressID: s.addressID, LegalEntityID: 1, RegionID: s.regionID, RegionName: s.regionName,
		})
		if err != nil {
			t.Fatal(err)
		}
		loid := "E2E-LOID-" + orderNo6(customerID)
		loID, err := a.Aaa.CreateLoAccount(ctx, aaa.LoAccount{
			Loid: loid, CustomerID: customerID, LegalEntityID: 1, LegalEntityName: "主品牌·企业",
			OfferID: s.offerID, Status: "ACTIVE",
		})
		if err != nil {
			t.Fatal(err)
		}
		// 认证达标:ACTIVE 放行并下发套餐带宽。
		d, err := authz.Decide(ctx, loid)
		if err != nil || !d.Authorize {
			t.Fatalf("decide=%+v err=%v", d, err)
		}
		if d.Bandwidth != "300M" {
			t.Fatalf("bandwidth=%s, want 300M(套餐)", d.Bandwidth)
		}
		// 停机即时生效:HTTP STOP → SUSPENDED → 认证拒绝。
		if w := postOKStatus(t, ts, token, "/api/admin/v1/arrears/"+fmt.Sprint(customerID)+"/stop"); w != 200 {
			t.Fatalf("stop http=%d", w)
		}
		if _, err := authz.Decide(ctx, loid); !errors.Is(err, aaa.ErrSuspended) {
			t.Fatalf("停机后 err=%v, want ErrSuspended", err)
		}
		// 缴费复机:HTTP RESUME → ACTIVE → 认证恢复。
		if w := postOKStatus(t, ts, token, "/api/admin/v1/arrears/"+fmt.Sprint(customerID)+"/resume"); w != 200 {
			t.Fatalf("resume http=%d", w)
		}
		if d, err := authz.Decide(ctx, loid); err != nil || !d.Authorize {
			t.Fatalf("复机后 decide=%+v err=%v", d, err)
		}
		// 话单入账:PGEmitter 落库可查。
		em := aaabilling.NewPGEmitter(a.Aaa.(*aaa.PGStore))
		if err := em.Emit(ctx, aaabilling.CDR{
			LOID: loid, AcctStatus: 2, SessionID: "S-E2E", SessionTime: 600,
			InputOctets: 1024, OutputOctets: 2048, NASIP: "10.0.0.1",
		}); err != nil {
			t.Fatal(err)
		}
		cdrs, err := a.Aaa.ListCdrs(ctx, loid)
		if err != nil || len(cdrs) == 0 {
			t.Fatalf("cdrs=%d err=%v", len(cdrs), err)
		}
		if cdrs[0].BillingStatus != "UNBILLED" {
			t.Fatalf("billingStatus=%s", cdrs[0].BillingStatus)
		}
		_ = loID
	})

	t.Run("W7_下发重试留痕_指标告警", func(t *testing.T) {
		// 下发:建模板+任务 → 失败留痕 → 重试 → 执行成功。
		tplID, err := a.Provision.CreateTemplate(ctx, provision.Template{
			LegalEntityID: 1, Code: "TPL-E2E-" + orderNo6(time.Now().UnixNano()%1e12), Name: "E2E模板",
		})
		if err != nil {
			t.Fatal(err)
		}
		taskID, err := a.Provision.CreateTask(ctx, provision.Task{
			LoAccountID: 1, TemplateID: tplID, Status: "PENDING",
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := a.Provision.ExecuteTask(ctx, taskID); err != nil {
			t.Fatalf("ExecuteTask: %v", err)
		}
		logs, err := a.Provision.ListLogs(ctx, taskID)
		if err != nil || len(logs) == 0 || logs[len(logs)-1].Result != "SUCCESS" {
			t.Fatalf("logs=%+v err=%v", logs, err)
		}
		// 失败重试链路:再建一单,FailTask → RetryTask。
		task2, err := a.Provision.CreateTask(ctx, provision.Task{
			LoAccountID: 1, TemplateID: tplID, Status: "PENDING",
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := a.Provision.FailTask(ctx, task2, "e2e 模拟失败"); err != nil {
			t.Fatal(err)
		}
		if err := a.Provision.RetryTask(ctx, task2, 0); err != nil {
			t.Fatal(err)
		}
		logs2, err := a.Provision.ListLogs(ctx, task2)
		if err != nil || len(logs2) != 2 || logs2[1].Retries != 1 {
			t.Fatalf("retry logs=%+v err=%v", logs2, err)
		}

		// 采集:越限样本 → 指标入库 + CRITICAL 告警。
		threshold := 5.0
		resID, _ := a.Resource.CreateResource(ctx, resSeed(s.addressID, orderNo6(time.Now().UnixNano()%1e12)))
		col := &device.Collector{Dev: a.Device, Alarm: a.Alarm, PacketLossAlarmPct: &threshold}
		loss := 9.9
		if err := col.Ingest(ctx, device.Sample{
			ResourceID: resID, PacketLoss: &loss, Status: "FAULT", CollectedAt: time.Now(),
		}); err != nil {
			t.Fatal(err)
		}
		metrics, err := a.Device.ListMetrics(ctx, resID)
		if err != nil || len(metrics) == 0 {
			t.Fatalf("metrics=%d err=%v", len(metrics), err)
		}
		alarms, err := a.Alarm.ListAlarms(ctx, resID)
		if err != nil || len(alarms) == 0 || alarms[0].Level != "CRITICAL" {
			t.Fatalf("alarms=%+v err=%v", alarms, err)
		}
	})

	t.Run("W8_下单到激活全自动_人工只收费扫码", func(t *testing.T) {
		pub := &capPub{}
		m := app.NewAutomation(a.Order, pub)
		// 独立客户(quad_links.customer_id 唯一,不能与其它子测试共用种子客户)。
		custID, err := a.Customer.Create(ctx, customer.Customer{
			Name: "E2E-W8客户", Phone: "09172222222", IdType: "身份证", IdNo: "E2E-W8",
			RealNameStatus: "VERIFIED", ServiceStatus: "ACTIVE",
			AddressID: s.addressID, LegalEntityID: 1, RegionID: s.regionID, RegionName: s.regionName,
		})
		if err != nil {
			t.Fatal(err)
		}
		o0, err := a.Order.Submit(ctx, order.SubmitReq{
			CustomerID: custID, OfferID: s.offerID, AddressID: s.addressID,
			ChannelID: s.channelID, LegalEntityID: 1, RegionPath: "root.luzon.ncr.manila",
		})
		if err != nil {
			t.Fatal(err)
		}
		orderID := o0.ID
		// 独立地址+设备+端口(quad_links.address_id/port_id 唯一)。
		suffix := orderNo6(time.Now().UnixNano() % 1e12)
		var w8Addr int64
		if err := pool.QueryRow(ctx,
			`INSERT INTO addresses(path, level, name) VALUES($1, 1, $2) RETURNING id`,
			"w8"+suffix, "W8测试市").Scan(&w8Addr); err != nil {
			t.Fatal(err)
		}
		w8Res, err := a.Resource.CreateResource(ctx, resSeed(w8Addr, suffix))
		if err != nil {
			t.Fatal(err)
		}
		w8Port, err := a.Resource.CreatePort(ctx, portSeed(w8Res, &e2eSeed{
			addressID: w8Addr, regionID: s.regionID, regionName: s.regionName,
		}, suffix, 1))
		if err != nil {
			t.Fatal(err)
		}
		auto := []func(context.Context, int64) error{
			a.Order.CheckResource, a.Order.Reserve, a.Order.ChargeContract,
		}
		for _, step := range auto { // 人工:收费
			if err := step(ctx, orderID); err != nil {
				t.Fatal(err)
			}
		}
		if err := a.Resource.ReservePort(ctx, w8Port, orderID); err != nil {
			t.Fatal(err)
		}
		// 自动段一(5-8)+人工扫码(9,走 VerifyScan)+自动段二(10-12)。
		if err := m.AutoPreScan(ctx, orderID); err != nil {
			t.Fatalf("AutoPreScan: %v", err)
		}
		o, _, err := a.Order.Track(ctx, orderID)
		if err != nil || o.Stage != 8 {
			t.Fatalf("preScan stage=%d err=%v", o.Stage, err)
		}
		// 人工扫码:建资产/标签/四码。
		batchID, err := a.Asset.CreateBatch(ctx, asset.AssetBatch{LegalEntityID: 1,
			Code: "RK-W8-" + orderNo6(orderID), Name: "W8批次"})
		if err != nil {
			t.Fatal(err)
		}
		assetID, err := a.Asset.CreateAsset(ctx, asset.Asset{
			LegalEntityID: 1, LegalEntityName: "主品牌·企业", BatchID: batchID,
			AssetCode: "A-W8-" + orderNo6(orderID), Type: "ONU", Status: "IN_STOCK",
		})
		if err != nil {
			t.Fatal(err)
		}
		epc := "EPC-W8-" + orderNo6(orderID)
		if _, err := a.Asset.CreateTag(ctx, asset.Tag{
			LegalEntityID: 1, TagNo: "T-W8-" + orderNo6(orderID), EpcCode: epc, Band: "UHF",
			BoundAssetID: assetID, Status: "BOUND", Battery: "OK",
		}); err != nil {
			t.Fatal(err)
		}
		if _, err := a.QuadLink.CreateLink(ctx, quadlink.QuadLink{
			AssetID: assetID, CustomerID: custID, PortID: w8Port,
			AddressID: w8Addr, LegalEntityID: 1, LegalEntityName: "主品牌·企业", Status: "UNLINKED",
		}); err != nil {
			t.Fatal(err)
		}
		if res, err := a.QuadLink.VerifyScan(ctx, quadlink.ScanReq{
			OrderID: orderID, WorkerID: 1, WorkerName: "E2E", ScannedEPC: epc,
		}); err != nil || res != "MATCH" {
			t.Fatalf("scan res=%s err=%v", res, err)
		}
		if err := a.Order.ScanBind(ctx, orderID); err != nil {
			t.Fatal(err)
		}
		// 自动段二:10-12 全自动到 DONE。
		if err := m.AutoPostScan(ctx, orderID); err != nil {
			t.Fatalf("AutoPostScan: %v", err)
		}
		o, _, err = a.Order.Track(ctx, orderID)
		if err != nil || o.Stage != 12 || o.Status != "DONE" {
			t.Fatalf("final stage=%d status=%s err=%v", o.Stage, o.Status, err)
		}
		if len(pub.got) != 7 { // 4(段一)+3(段二) 条状态变更事件
			t.Fatalf("events=%d, want 7", len(pub.got))
		}
	})
}

// postOKStatus 带登录态 POST,仅断言 HTTP 200。
func postOKStatus(t *testing.T, ts *httptest.Server, token, path string) int {
	t.Helper()
	code, _ := scanPost(t, ts, token, path, "")
	return code
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
