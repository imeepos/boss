package adminapi

// 代客下单直营风控拦截契约:42300 + data.reason 可行动文案(2026-09-01 事故回归:
// 只回"资源已被占用"时客服无从下手,不知道是地址堆积还是同号多单)。

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/pkg/apitypes"
)

func TestOrderSubmitRiskBlockedReason(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mgr := auth.NewManager("s", time.Hour)
	f := &fakeOrder{submitErr: order.NewAddressCapExceeded(26, 10)}
	r := newOrderRouter(f, &fakeUser{permOk: true}, mgr)
	w := postJSONAuth(t, r, "/api/admin/v1/orders",
		`{"customerId":215,"offerId":101,"addressId":288,"channelId":104}`,
		authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	var body struct {
		Code apitypes.Code `json:"code"`
		Data struct {
			Reason string `json:"reason"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	if body.Code != apitypes.CodeResourceBusy {
		t.Fatalf("code=%d", body.Code)
	}
	if !strings.Contains(body.Data.Reason, "在途订单已有 26 笔") ||
		!strings.Contains(body.Data.Reason, "risk.direct.addressCap") {
		t.Fatalf("reason=%q", body.Data.Reason)
	}
}

func TestOrderSubmitRiskBlockedPhoneKind(t *testing.T) {
	gin.SetMode(gin.TestMode)
	mgr := auth.NewManager("s", time.Hour)
	f := &fakeOrder{submitErr: order.NewPhoneCapExceeded(20, 20)}
	r := newOrderRouter(f, &fakeUser{permOk: true}, mgr)
	w := postJSONAuth(t, r, "/api/admin/v1/orders",
		`{"customerId":213,"offerId":101,"addressId":3988,"channelId":104}`,
		authToken(t, mgr))
	var body struct {
		Code apitypes.Code `json:"code"`
		Data struct {
			Reason string `json:"reason"`
		} `json:"data"`
	}
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode: %v body=%s", err, w.Body.String())
	}
	if body.Code != apitypes.CodeResourceBusy {
		t.Fatalf("code=%d", body.Code)
	}
	if !strings.Contains(body.Data.Reason, "24小时内已下单 20 单") ||
		!strings.Contains(body.Data.Reason, "risk.direct.phoneCap") {
		t.Fatalf("reason=%q", body.Data.Reason)
	}
}
