package userapi

// 消息口径一致性回归(2026-09-04 客户端链路修复):
// 列表 messageId 恒为数字主键(单条已读寻址键)、payload 快照 MSG- 展示编号降级
// messageNo、read/createdAt 以表列为单一事实源;单条已读/read-all/home 红点三者一致。

import (
	"context"
	"net/http"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/portal"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// newMessageRouter 装配消息链路最小桩,返回独立 Portal 存储句柄供造数。
func newMessageRouter() (*gin.Engine, *auth.Manager, portal.Service) {
	gin.SetMode(gin.TestMode)
	mgr := auth.NewManager("test-secret", time.Hour)
	ps := portal.NewMemory()
	r := gin.New()
	Register(r, &app.Application{
		Customer:  &userPortalCustSvc{c: userPortalCust()},
		Billing:   &fakeBilling{},
		Order:     &fakeOrder{},
		WorkOrder: &userPortalWo{},
		Portal:    ps,
	}, mgr)
	return r, mgr, ps
}

// seedPortalMessage 落一条带旧口径快照的站内消息(payload 含 messageId=MSG-xx/read:false)。
func seedPortalMessage(t *testing.T, ps portal.Service, cid int64, displayNo string) {
	t.Helper()
	err := ps.PutMessage(context.Background(), cid, map[string]any{
		"messageId": displayNo, "read": false, "category": "system",
		"title": "报修已受理", "content": "工单 TKT-1 已受理", "tag": "报修",
		"tagLevel": "fault", "createdAt": time.Now(),
	})
	if err != nil {
		t.Fatalf("PutMessage: %v", err)
	}
}

func messageItem(t *testing.T, data map[string]any, idx int) map[string]any {
	t.Helper()
	items, ok := data["items"].([]any)
	if !ok || len(items) <= idx {
		t.Fatalf("items missing: %v", data)
	}
	item, ok := items[idx].(map[string]any)
	if !ok {
		t.Fatalf("item shape: %v", items[idx])
	}
	return item
}

// TestPortal_Messages_IDAndReadConsistency 数字 id 寻址 + read 表列单一事实源 + 红点一致。
func TestPortal_Messages_IDAndReadConsistency(t *testing.T) {
	cust := userPortalCust()
	r, mgr, ps := newMessageRouter()
	tok, _ := signCustomerToken(mgr, cust.ID, cust.Phone)
	seedPortalMessage(t, ps, cust.ID, "MSG-101")
	seedPortalMessage(t, ps, cust.ID, "MSG-102")

	// 1) 列表:messageId=数字主键,快照展示编号降级 messageNo。
	_, data := userPortalCode(t, userPortalDo(r, http.MethodGet, "/api/user/v1/messages", "", tok))
	item := messageItem(t, data, 0) // 最新一条(id=2)
	if got := item["messageId"]; got != "2" {
		t.Fatalf("messageId want numeric 2, got %v", got)
	}
	if got := item["messageNo"]; got != "MSG-102" {
		t.Fatalf("messageNo want MSG-102, got %v", got)
	}
	if read, ok := item["read"].(bool); !ok || read {
		t.Fatalf("initial read want false, got %v", item["read"])
	}

	// 2) 单条已读:列表下发的数字 id 可直接寻址(修复前 MSG-xx 恒 40400)。
	if code, _ := userPortalCode(t, userPortalDo(r, http.MethodPut, "/api/user/v1/messages/2/read", "", tok)); code != int(apitypes.CodeOK) {
		t.Fatalf("read one by numeric id failed")
	}
	_, data = userPortalCode(t, userPortalDo(r, http.MethodGet, "/api/user/v1/messages", "", tok))
	if read, _ := messageItem(t, data, 0)["read"].(bool); !read {
		t.Fatalf("read flag not persisted to list")
	}

	// 3) 展示编号不再作为寻址键。
	if code, _ := userPortalCode(t, userPortalDo(r, http.MethodPut, "/api/user/v1/messages/MSG-101/read", "", tok)); code != int(apitypes.CodeNotFound) {
		t.Fatalf("display-no addressing want 40400, got %d", code)
	}

	// 4) read-all 后:列表全 true(快照 read:false 不再覆盖)+ home 红点灭。
	if code, _ := userPortalCode(t, userPortalDo(r, http.MethodPost, "/api/user/v1/messages/read-all", "", tok)); code != int(apitypes.CodeOK) {
		t.Fatalf("read-all failed")
	}
	_, data = userPortalCode(t, userPortalDo(r, http.MethodGet, "/api/user/v1/messages", "", tok))
	for i := range []int{0, 1} {
		if read, _ := messageItem(t, data, i)["read"].(bool); !read {
			t.Fatalf("item %d read want true after read-all", i)
		}
	}
	_, home := userPortalCode(t, userPortalDo(r, http.MethodGet, "/api/user/v1/home", "", tok))
	if unread, ok := home["hasUnread"].(bool); !ok || unread {
		t.Fatalf("home hasUnread want false, got %v", home["hasUnread"])
	}
}

// TestPortal_Messages_CategoryFilter 分类过滤仍按 payload.category(口径改动不伤过滤)。
func TestPortal_Messages_CategoryFilter(t *testing.T) {
	cust := userPortalCust()
	r, mgr, ps := newMessageRouter()
	tok, _ := signCustomerToken(mgr, cust.ID, cust.Phone)
	seedPortalMessage(t, ps, cust.ID, "MSG-201")
	_, data := userPortalCode(t, userPortalDo(r, http.MethodGet, "/api/user/v1/messages?category=system", "", tok))
	if items, _ := data["items"].([]any); len(items) != 1 {
		t.Fatalf("category filter want 1 item, got %d", len(items))
	}
}
