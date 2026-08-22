package app_test

// 幂等键回归(000116):同客户同 requestId 重放下单返回已有订单,不重复建单。
// 运行: BOSS_PG_TEST_DSN 同 TestE2E_OrderLifecycle_Integration。

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/pkg/config"
	"github.com/ymm-001/boss/internal/pkg/database"
)

func TestE2E_SubmitRequestIdempotency_Integration(t *testing.T) {
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
	addr := newE2EAddress(t, ctx, pool, a, "idem")
	reqID := fmt.Sprintf("idem-%d", time.Now().UnixNano())
	base := order.SubmitReq{
		CustomerID: s.customerID, OfferID: s.offerID, AddressID: addr,
		ChannelID: s.channelID, RegionPath: "root.luzon.ncr.manila", RequestID: reqID,
	}

	o1, err := a.Order.Submit(ctx, base)
	if err != nil {
		t.Fatalf("首次下单: %v", err)
	}
	o2, err := a.Order.Submit(ctx, base)
	if err != nil {
		t.Fatalf("幂等重放: %v", err)
	}
	if o1.ID != o2.ID || o1.OrderNo != o2.OrderNo {
		t.Fatalf("幂等重放返回不同订单: %d/%s vs %d/%s", o1.ID, o1.OrderNo, o2.ID, o2.OrderNo)
	}

	// 不同 requestId → 新订单
	base.RequestID = reqID + "-b"
	o3, err := a.Order.Submit(ctx, base)
	if err != nil {
		t.Fatal(err)
	}
	if o3.ID == o1.ID {
		t.Fatalf("不同 requestId 应建新订单")
	}
}
