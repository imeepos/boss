package app_test

// 回归(000110):同地址第二单环节5 不再被历史 UNLINKED 链路阻塞。
// 现场:2026-08-23 102 验收 ORD-20260823-000445 charge 50000,
// applyTag 建 quad_links 撞 uq_quad_links_address (23505)。
// 运行: BOSS_PG_TEST_DSN 同 TestE2E_OrderLifecycle_Integration。

import (
	"context"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/httpapi"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/config"
	"github.com/ymm-001/boss/internal/pkg/database"
)

func TestE2E_QuadLinkAddressReopen_Integration(t *testing.T) {
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
	t.Cleanup(pool.Close)

	s := seedE2E(t, ctx, pool, a)
	r := gin.New()
	httpapi.RegisterRoutes(r, a, auth.NewManager("e2e-secret", time.Hour))
	ts := httptest.NewServer(r)
	defer ts.Close()
	token := loginE2E(t, ts, s)

	addr := newE2EAddress(t, ctx, pool, a, "reopen")

	// 第一单只走到环节5:落一条 UNLINKED 四码后停,模拟历史遗留/未扫码释放。
	id1, _ := submitOrderE2E(t, ts, token, s, addr)
	for _, step := range []func(context.Context, int64) error{
		a.Order.CheckResource, a.Order.Reserve, a.Order.ChargeContract, a.Order.ApplyTag,
	} {
		if err := step(ctx, id1); err != nil {
			t.Fatalf("第一单推进失败: %v", err)
		}
	}

	// 同地址第二单:核查→预占(用剩余 IDLE 口)→环节5 必须成功(修复前 23505)。
	id2, _ := submitOrderE2E(t, ts, token, s, addr)
	if err := a.Order.CheckResource(ctx, id2); err != nil {
		t.Fatal(err)
	}
	if err := a.Order.Reserve(ctx, id2); err != nil {
		t.Fatal(err)
	}
	if err := a.Order.ChargeContract(ctx, id2); err != nil {
		t.Fatal(err)
	}
	if err := a.Order.ApplyTag(ctx, id2); err != nil {
		t.Fatalf("同地址重开单 applyTag 失败(000110 回归): %v", err)
	}

	// 同地址两条链路并存且都 UNLINKED;端口不同(各占一个 IDLE 口)。
	var unlinkCnt, portOverlap int
	if err := pool.QueryRow(ctx, `
		SELECT
			count(*) FILTER (WHERE address_id = $1 AND status = 'UNLINKED'),
			count(*) FILTER (WHERE address_id = $1 AND port_id IS NOT NULL)
				- count(DISTINCT port_id) FILTER (WHERE address_id = $1)
		FROM quad_links`, addr).Scan(&unlinkCnt, &portOverlap); err != nil {
		t.Fatal(err)
	}
	if unlinkCnt != 2 {
		t.Fatalf("同地址 UNLINKED 链路数=%d, want 2", unlinkCnt)
	}
	if portOverlap != 0 {
		t.Fatalf("两条链路端口重叠(各应占独立端口)")
	}

	// 第二单推到环节12:订单 DONE 后端口必须 RESERVED→USED(terms.md §4 生命周期)。
	for _, step := range []func(context.Context, int64) error{
		a.Order.CreateUserProfile, a.Order.PreConfigOLT, a.Order.DispatchOrder,
		a.Order.ScanBind, a.Order.ActivateUser, a.Order.NotifyActivation, a.Order.UpdateMap,
	} {
		if err := step(ctx, id2); err != nil {
			t.Fatalf("推进到 DONE 失败: %v", err)
		}
	}
	var usedCnt int
	if err := pool.QueryRow(ctx,
		`SELECT count(*) FROM ports WHERE order_id = $2 AND status = 'USED' AND address_id = $1`,
		addr, id2).Scan(&usedCnt); err != nil {
		t.Fatal(err)
	}
	if usedCnt != 1 {
		t.Fatalf("订单 DONE 后 USED 端口数=%d, want 1", usedCnt)
	}
}
