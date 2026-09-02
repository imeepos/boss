package app_test

// gRPC 服务端真实 PG 集成测试(债务偿还验收):quadlink/aaa/device/provision v1 契约全部可走通。
// 运行: BOSS_PG_TEST_DSN="host=192.168.0.102 port=25432 user=boss password=boss dbname=boss sslmode=disable" go test ./internal/app/ -run TestE2E_GRPC -v -count=1

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"google.golang.org/grpc"

	aaav1 "github.com/ymm-001/boss/api/proto/boss/aaa/v1"
	commonv1 "github.com/ymm-001/boss/api/proto/boss/common/v1"
	devicev1 "github.com/ymm-001/boss/api/proto/boss/device/v1"
	provisionv1 "github.com/ymm-001/boss/api/proto/boss/provision/v1"
	quadlinkv1 "github.com/ymm-001/boss/api/proto/boss/quadlink/v1"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/aaa"
	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/provision"
	"github.com/ymm-001/boss/internal/pkg/config"
	"github.com/ymm-001/boss/internal/pkg/database"
)

type grpcE2ESeed struct {
	orderID, orderNo int64
	ticketNo         string
	portID           int64
	assetID          int64
	tagEpc           string
	loid             string
	loAccountID      int64
	qosCode          string
	templateCode     string
	taskNo           string
	resourceCode     string
}

// seedGRPCE2E 预置 gRPC 四服务所需全量数据:订单推至环节8 + 派单/四码/资产标签/LO账号/下发模板。
func seedGRPCE2E(t *testing.T, ctx context.Context, a *app.Application, pool *pgxpool.Pool) *grpcE2ESeed {
	t.Helper()
	s := seedE2E(t, ctx, pool, a)
	suffix := fmt.Sprintf("%d", time.Now().UnixNano()%1e12)
	seed := &grpcE2ESeed{
		ticketNo: "TIC-GRPC-" + suffix, tagEpc: "EPC-GRPC-" + suffix,
		loid: "LOID-GRPC-" + suffix, templateCode: "TPL-FTTH-" + suffix, taskNo: "TASK-GRPC-" + suffix,
	}

	o, err := a.Order.Submit(ctx, order.SubmitReq{
		CustomerID: s.customerID, OfferID: s.offerID, AddressID: s.addressID,
		// LegalEntityID 不传:归属由地址推导(5c5f2a7 契约)。
		ChannelID: s.channelID, RegionPath: "root.luzon.ncr.manila",
	})
	if err != nil {
		t.Fatalf("submit order: %v", err)
	}
	seed.orderID = o.ID
	steps := []struct {
		name string
		run  func(context.Context, int64) error
	}{
		{"checkResource", a.Order.CheckResource}, {"reserve", a.Order.Reserve},
		{"chargeContract", a.Order.ChargeContract}, {"applyTag", a.Order.ApplyTag},
		{"createUserProfile", a.Order.CreateUserProfile}, {"preConfigOLT", a.Order.PreConfigOLT},
		{"dispatchOrder", a.Order.DispatchOrder},
	}
	for _, st := range steps {
		if err := st.run(ctx, seed.orderID); err != nil {
			t.Fatalf("order %s: %v", st.name, err)
		}
	}
	// applyTag 已自动预占端口+落四码(UNLINKED,asset_id=0),dispatchOrder 已自动生成工单。
	// 此处只做对齐:读回预占端口、工单改用种子号。
	if err := pool.QueryRow(ctx,
		`SELECT id FROM ports WHERE order_id=$1`, seed.orderID).Scan(&seed.portID); err != nil {
		t.Fatalf("find reserved port: %v", err)
	}
	if _, err := pool.Exec(ctx,
		`UPDATE dispatch_tickets SET ticket_no=$2 WHERE order_id=$1`, seed.orderID, seed.ticketNo); err != nil {
		t.Fatalf("rename dispatch ticket: %v", err)
	}

	// 资产 + 标签(EPC 预绑定资产)。
	batchID, err := a.Asset.CreateBatch(ctx, asset.AssetBatch{LegalEntityID: 1, Code: "RK-GRPC-" + suffix, Name: "gRPC批次"})
	if err != nil {
		t.Fatalf("create batch: %v", err)
	}
	seed.assetID, err = a.Asset.CreateAsset(ctx, asset.Asset{
		AssetCode: "A-GRPC-" + suffix, BatchID: batchID, LegalEntityID: 1, LegalEntityName: "主品牌·企业",
		AddressID: s.addressID, RegionID: s.regionID, RegionName: s.regionName,
		Type: "光猫", Status: "DEPLOYED",
	})
	if err != nil {
		t.Fatalf("create asset: %v", err)
	}
	tagID, err := a.Asset.CreateTag(ctx, asset.Tag{
		LegalEntityID: 1, TagNo: "TAG-GRPC-" + suffix, EpcCode: seed.tagEpc,
		Band: "UHF", BoundAssetID: seed.assetID, Status: "BOUND", Battery: "100",
	})
	if err != nil {
		t.Fatalf("create tag: %v", err)
	}
	if _, err := pool.Exec(ctx, `UPDATE assets SET tag_id = $1 WHERE id = $2`, tagID, seed.assetID); err != nil {
		t.Fatalf("bind tag to asset: %v", err)
	}

	// 四码:applyTag 已按订单落(客户/端口/地址一致),置备资产后回填 asset_id 并置 LINKED。
	if _, err := pool.Exec(ctx, `
		UPDATE quad_links SET asset_id=$2, status='LINKED'
		WHERE port_id=$1`, seed.portID, seed.assetID); err != nil {
		t.Fatalf("backfill quad link: %v", err)
	}

	// LO 账号(客户 1:1,挂套餐与 QoS 模板;编码带后缀保证可重复运行)。
	seed.qosCode = "QOS-HS-" + suffix
	var qosID int64
	if err := pool.QueryRow(ctx,
		`INSERT INTO qos_templates(legal_entity_id, code, name) VALUES(1,$1,'高速') RETURNING id`,
		seed.qosCode).Scan(&qosID); err != nil {
		t.Fatalf("create qos template: %v", err)
	}
	seed.loAccountID, err = a.Aaa.CreateLoAccount(ctx, aaa.LoAccount{
		Loid: seed.loid, CustomerID: s.customerID, LegalEntityID: 1, LegalEntityName: "主品牌·企业",
		RegionID: s.regionID, RegionName: s.regionName,
		OfferID: s.offerID, QosTemplateID: qosID, Status: "ACTIVE",
	})
	if err != nil {
		t.Fatalf("create lo account: %v", err)
	}

	if _, err := a.Provision.CreateTemplate(ctx, provision.Template{
		LegalEntityID: 1, Code: seed.templateCode, Name: "FTTH标准开通",
	}); err != nil {
		t.Fatalf("create template: %v", err)
	}

	resources, err := a.Resource.ListResources(ctx)
	if err != nil || len(resources) == 0 {
		t.Fatalf("list resources: %v", err)
	}
	seed.resourceCode = resources[0].Code
	return seed
}

// TestE2E_GRPC_Services_Integration 四契约服务经真实 PG 全链路验证。
func TestE2E_GRPC_Services_Integration(t *testing.T) {
	dsn := os.Getenv("BOSS_PG_TEST_DSN")
	if dsn == "" {
		t.Skip("BOSS_PG_TEST_DSN 未设置,跳过集成测试")
	}
	ctx := context.Background()
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
	defer pool.Close()

	seed := seedGRPCE2E(t, ctx, a, pool)
	conn := newBufConnServer(t, func(s *grpc.Server) { app.RegisterGRPC(s, a) })
	aaaCli := aaav1.NewAaaServiceClient(conn)
	devCli := devicev1.NewDeviceIngestServiceClient(conn)
	provCli := provisionv1.NewProvisionServiceClient(conn)
	quadCli := quadlinkv1.NewQuadLinkServiceClient(conn)

	t.Run("aaa_停复机与授权话单", func(t *testing.T) {
		auth, err := aaaCli.GetAuthorization(ctx, &aaav1.GetAuthorizationRequest{Loid: seed.loid})
		if err != nil {
			t.Fatal(err)
		}
		if auth.Code != commonv1.Code_CODE_OK || auth.Status != "ACTIVE" || auth.Bandwidth != "300M" || auth.QosTemplate != seed.qosCode {
			t.Fatalf("auth=%+v", auth)
		}

		sp, err := aaaCli.Suspend(ctx, &aaav1.SuspendRequest{LoAccountId: seed.loAccountID, Reason: "ARREARS"})
		if err != nil || sp.Code != commonv1.Code_CODE_OK || !sp.Effective {
			t.Fatalf("suspend resp=%v err=%v", sp, err)
		}
		auth2, _ := aaaCli.GetAuthorization(ctx, &aaav1.GetAuthorizationRequest{Loid: seed.loid})
		if auth2.Status != "SUSPENDED" {
			t.Fatalf("after suspend status=%s", auth2.Status)
		}

		rs, err := aaaCli.Resume(ctx, &aaav1.ResumeRequest{LoAccountId: seed.loAccountID})
		if err != nil || rs.Code != commonv1.Code_CODE_OK || !rs.Effective {
			t.Fatalf("resume resp=%v err=%v", rs, err)
		}

		cr, err := aaaCli.EmitCDR(ctx, &aaav1.CDR{
			Loid: seed.loid, SessionId: "S-1", EventType: "Stop",
			InputOctets: 1024, OutputOctets: 2048, StartedAt: time.Now().Unix() - 120, FinishedAt: time.Now().Unix(),
		})
		if err != nil || cr.Code != commonv1.Code_CODE_OK {
			t.Fatalf("emit cdr resp=%v err=%v", cr, err)
		}
		cdrs, err := a.Aaa.ListCdrs(ctx, seed.loid)
		if err != nil || len(cdrs) != 1 {
			t.Fatalf("cdrs=%d err=%v", len(cdrs), err)
		}
	})

	t.Run("device_指标上报与Trap告警", func(t *testing.T) {
		rm, err := devCli.ReportMetrics(ctx, &devicev1.ReportMetricsRequest{
			Samples: []*devicev1.MetricSample{{
				ResourceCode: seed.resourceCode,
				Gauges:       map[string]float64{"packet_loss": 12, "optical_power": -24},
				CollectedAt:  time.Now().Unix(),
			}},
		})
		if err != nil || rm.Accepted != 1 {
			t.Fatalf("report metrics resp=%v err=%v", rm, err)
		}
		rt, err := devCli.ReportTrap(ctx, &devicev1.Trap{
			ResourceCode: seed.resourceCode, Oid: "1.3.6.1.4.1.0.1", Severity: "CRITICAL",
		})
		if err != nil || rt.Accepted != 1 {
			t.Fatalf("report trap resp=%v err=%v", rt, err)
		}
		alarms, err := a.Alarm.ListAlarms(ctx, 0)
		if err != nil || len(alarms) < 2 {
			t.Fatalf("alarms=%d err=%v", len(alarms), err)
		}
	})

	t.Run("provision_入队重试查询", func(t *testing.T) {
		en, err := provCli.EnqueueTask(ctx, &provisionv1.Task{
			OrderId: seed.orderID, TaskNo: seed.taskNo, StageEvent: "preConfigOLT", TemplateCode: seed.templateCode,
		})
		if err != nil || en.Code != commonv1.Code_CODE_OK || en.Status != "PENDING" {
			t.Fatalf("enqueue resp=%v err=%v", en, err)
		}
		tk, err := a.Provision.GetTaskByNo(ctx, seed.taskNo)
		if err != nil {
			t.Fatal(err)
		}
		if err := a.Provision.FailTask(ctx, tk.ID, "simulated olt timeout", provision.ExecTrace{}); err != nil {
			t.Fatalf("fail task: %v", err)
		}
		rt, err := provCli.RetryTask(ctx, &provisionv1.RetryTaskRequest{TaskNo: seed.taskNo})
		if err != nil || rt.Code != commonv1.Code_CODE_OK || rt.Status != "PENDING" {
			t.Fatalf("retry resp=%v err=%v", rt, err)
		}
		got, err := provCli.GetTask(ctx, &provisionv1.GetTaskRequest{TaskNo: seed.taskNo})
		if err != nil {
			t.Fatal(err)
		}
		if got.TemplateCode != seed.templateCode || got.StageEvent != "preConfigOLT" {
			t.Fatalf("got=%+v", got)
		}
	})

	t.Run("quadlink_扫码绑定到拆机解绑", func(t *testing.T) {
		sb, err := quadCli.ScanBind(ctx, &quadlinkv1.ScanBindRequest{
			TicketNo: seed.ticketNo, Epc: seed.tagEpc, WorkerId: 9, WorkerName: "师傅甲",
		})
		if err != nil || sb.Code != commonv1.Code_CODE_OK || sb.Result != "MATCH" || !sb.StageAdvanced {
			t.Fatalf("scan bind resp=%+v err=%v", sb, err)
		}
		o, _, _ := a.Order.Track(ctx, seed.orderID)
		if o.Stage != 9 {
			t.Fatalf("stage=%d, want 9", o.Stage)
		}

		q, err := quadCli.Query(ctx, &quadlinkv1.QueryRequest{Key: &quadlinkv1.QueryRequest_AssetId{AssetId: seed.assetID}})
		if err != nil || q.Status != "LINKED" {
			t.Fatalf("query resp=%+v err=%v", q, err)
		}

		ds, err := quadCli.DismantleScan(ctx, &quadlinkv1.DismantleScanRequest{TicketNo: seed.ticketNo, Epc: seed.tagEpc})
		if err != nil || ds.Code != commonv1.Code_CODE_OK || !ds.Unlinked {
			t.Fatalf("dismantle resp=%+v err=%v", ds, err)
		}
		rep, err := quadCli.Reconcile(ctx, &quadlinkv1.ReconcileRequest{})
		if err != nil || rep.Total < 1 {
			t.Fatalf("reconcile resp=%+v err=%v", rep, err)
		}
	})
}
