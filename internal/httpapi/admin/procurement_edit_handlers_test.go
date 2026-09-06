package adminapi

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/procurement"
	"github.com/ymm-001/boss/internal/pkg/audit"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/middleware"
)

// P2-W2-T2 handler 层测试:审计断言(B/C/E)+ 错误码断言(40900/40400/42200)。
// 桩内嵌 procurement.Service,只覆写被测方法;审计用 auditCapture 捕获。

type procEditStub struct {
	procurement.Service
	detail       *procurement.Order // 第 1 次 GetOrderDetail 返回(编辑前)
	detail2      *procurement.Order // 第 2 次起返回(编辑后)
	detailErr    error
	calls        int
	updErr       error
	enableErr    error
	rejErr       error
	rejectReason string
}

func (s *procEditStub) GetOrderDetail(context.Context, int64) (*procurement.Order, error) {
	if s.detailErr != nil {
		return nil, s.detailErr
	}
	s.calls++
	if s.calls == 1 {
		return s.detail, nil
	}
	return s.detail2, nil
}
func (s *procEditStub) UpdateOrderDraft(context.Context, int64, procurement.OrderDraftUpdate) error {
	return s.updErr
}
func (s *procEditStub) EnableSupplier(context.Context, int64) error { return s.enableErr }
func (s *procEditStub) RejectReceipt(_ context.Context, _ int64, reason string) error {
	s.rejectReason = reason
	return s.rejErr
}
func (s *procEditStub) UpdateSupplier(context.Context, int64, procurement.SupplierUpdate) error {
	return nil
}

type auditCapture struct{ events []audit.Event }

func (a *auditCapture) Write(_ context.Context, e audit.Event) error {
	a.events = append(a.events, e)
	return nil
}
func (a *auditCapture) List(context.Context, audit.Query) ([]audit.Entry, error) { return nil, nil }

func procEditCtx(method, path, body string) (*gin.Context, *httptest.ResponseRecorder) {
	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)
	if body == "" {
		c.Request = httptest.NewRequest(method, path, nil)
	} else {
		c.Request = httptest.NewRequest(method, path, bytes.NewBufferString(body))
		c.Request.Header.Set("Content-Type", "application/json")
	}
	c.Params = gin.Params{{Key: "id", Value: "5"}}
	c.Set(middleware.CtxClaims, &auth.Claims{AccountID: 7})
	return c, w
}

func bodyCode(t *testing.T, w *httptest.ResponseRecorder) float64 {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &m); err != nil {
		t.Fatalf("bad body %s: %v", w.Body.String(), err)
	}
	code, _ := m["code"].(float64)
	return code
}

func draftOrder(id int64, status, remark string, total float64) *procurement.Order {
	return &procurement.Order{
		ID: id, ProcurementNo: "PO-20260905-00001", Status: status, Remark: remark, TotalAmount: total,
		Items: []procurement.OrderItem{{ID: 1, OrderID: id, MaterialCode: "ONU-X", Quantity: 2, UnitAmount: 100}},
	}
}

// B 正:启用成功写审计(状态变更/procurement_supplier/操作人)。
func TestEnableSupplierHandler_Audits(t *testing.T) {
	stub := &procEditStub{}
	log := &auditCapture{}
	c, w := procEditCtx(http.MethodPost, "/procurement/suppliers/5/enable", "")
	procurementEnableSupplier(&app.Application{Procurement: stub, Audit: log})(c)
	if w.Code != http.StatusOK || bodyCode(t, w) != 0 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if len(log.events) != 1 || log.events[0].Action != "状态变更" ||
		log.events[0].TargetType != "procurement_supplier" || log.events[0].TargetID != "5" ||
		log.events[0].AccountID != 7 {
		t.Fatalf("audit=%+v", log.events)
	}
}

// B 正(幂等):重复启用仍 200 且再次写审计。
func TestEnableSupplierHandler_IdempotentStillAudits(t *testing.T) {
	stub := &procEditStub{}
	log := &auditCapture{}
	for i := 0; i < 2; i++ {
		c, w := procEditCtx(http.MethodPost, "/procurement/suppliers/5/enable", "")
		procurementEnableSupplier(&app.Application{Procurement: stub, Audit: log})(c)
		if w.Code != http.StatusOK || bodyCode(t, w) != 0 {
			t.Fatalf("call %d status=%d body=%s", i, w.Code, w.Body.String())
		}
	}
	if len(log.events) != 2 {
		t.Fatalf("events=%d want 2", len(log.events))
	}
}

// B 反:不存在 40400,不写审计。
func TestEnableSupplierHandler_NotFoundNoAudit(t *testing.T) {
	stub := &procEditStub{enableErr: procurement.ErrNotFound}
	log := &auditCapture{}
	c, w := procEditCtx(http.MethodPost, "/procurement/suppliers/5/enable", "")
	procurementEnableSupplier(&app.Application{Procurement: stub, Audit: log})(c)
	if code := bodyCode(t, w); code != 40400 {
		t.Fatalf("code=%v body=%s", code, w.Body.String())
	}
	if len(log.events) != 0 {
		t.Fatalf("失败路径不得写审计: %+v", log.events)
	}
}

// C 正:DRAFT 编辑成功,审计含变更前后键值。
func TestUpdateOrderHandler_AuditsBeforeAfter(t *testing.T) {
	stub := &procEditStub{
		detail:  draftOrder(5, "DRAFT", "旧备注", 240),
		detail2: draftOrder(5, "DRAFT", "新备注", 140),
	}
	log := &auditCapture{}
	c, w := procEditCtx(http.MethodPut, "/procurement/orders/5",
		`{"remark":"新备注","items":[{"materialCode":"ONU-X","quantity":1,"unitAmount":140}]}`)
	procurementUpdateOrder(&app.Application{Procurement: stub, Audit: log})(c)
	if w.Code != http.StatusOK || bodyCode(t, w) != 0 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if len(log.events) != 1 || log.events[0].Action != "数据变更" || log.events[0].TargetType != "procurement_order" {
		t.Fatalf("audit=%+v", log.events)
	}
	before, _ := log.events[0].Detail["before"].(map[string]any)
	after, _ := log.events[0].Detail["after"].(map[string]any)
	if before["totalAmount"] != 240.0 || after["totalAmount"] != 140.0 ||
		before["remark"] != "旧备注" || after["remark"] != "新备注" {
		t.Fatalf("before=%v after=%v", before, after)
	}
}

// C 反:非 DRAFT 状态冲突 40900,不写审计。
func TestUpdateOrderHandler_NonDraft40900(t *testing.T) {
	stub := &procEditStub{
		detail:  draftOrder(5, "SUBMITTED", "旧", 240),
		detail2: draftOrder(5, "SUBMITTED", "旧", 240),
		updErr:  fmt.Errorf("procurement: order 5 status=SUBMITTED: %w", procurement.ErrStateConflict),
	}
	log := &auditCapture{}
	c, w := procEditCtx(http.MethodPut, "/procurement/orders/5", `{"remark":"x"}`)
	procurementUpdateOrder(&app.Application{Procurement: stub, Audit: log})(c)
	if code := bodyCode(t, w); code != 40900 {
		t.Fatalf("code=%v body=%s", code, w.Body.String())
	}
	if len(log.events) != 0 {
		t.Fatalf("失败路径不得写审计: %+v", log.events)
	}
}

// C 反:非 DRAFT 但返回 404 时(桩模拟),错误透传 40400,不写审计。
func TestUpdateOrderHandler_NotFound40400(t *testing.T) {
	stub := &procEditStub{detailErr: procurement.ErrNotFound}
	log := &auditCapture{}
	c, w := procEditCtx(http.MethodPut, "/procurement/orders/5", `{"remark":"x"}`)
	procurementUpdateOrder(&app.Application{Procurement: stub, Audit: log})(c)
	if code := bodyCode(t, w); code != 40400 {
		t.Fatalf("code=%v body=%s", code, w.Body.String())
	}
	if len(log.events) != 0 {
		t.Fatalf("失败路径不得写审计: %+v", log.events)
	}
}

// A 正:入参无 code 字段,多余 JSON 键被忽略(编码结构性不可改);禁用态由域层保证可改。
func TestUpdateSupplierHandler_CodeNotBindable(t *testing.T) {
	stub := &procEditStub{}
	c, w := procEditCtx(http.MethodPut, "/procurement/suppliers/5",
		`{"code":"HACKED-NEW","name":"新名称"}`)
	procurementUpdateSupplier(&app.Application{Procurement: stub})(c)
	if w.Code != http.StatusOK || bodyCode(t, w) != 0 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
}

// D 正:详情返回单头+明细行+状态。
func TestGetOrderHandler_DetailOK(t *testing.T) {
	stub := &procEditStub{detail: draftOrder(5, "DRAFT", "r", 240), detail2: draftOrder(5, "DRAFT", "r", 240)}
	c, w := procEditCtx(http.MethodGet, "/procurement/orders/5", "")
	procurementGetOrder(&app.Application{Procurement: stub})(c)
	if w.Code != http.StatusOK || bodyCode(t, w) != 0 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	var m map[string]any
	_ = json.Unmarshal(w.Body.Bytes(), &m)
	data, _ := m["data"].(map[string]any)
	item, _ := data["item"].(map[string]any)
	items, _ := item["items"].([]any)
	if item["status"] != "DRAFT" || len(items) != 1 {
		t.Fatalf("detail=%v", m)
	}
}

// D 反:未命中沿用既有 not found 语义(40400)。
func TestGetOrderHandler_NotFound40400(t *testing.T) {
	stub := &procEditStub{detailErr: procurement.ErrNotFound}
	c, w := procEditCtx(http.MethodGet, "/procurement/orders/5", "")
	procurementGetOrder(&app.Application{Procurement: stub})(c)
	if code := bodyCode(t, w); code != 40400 {
		t.Fatalf("code=%v body=%s", code, w.Body.String())
	}
}

// E 正:驳回写审计带原因;空请求体用缺省文案。
func TestRejectReceiptHandler_AuditsReason(t *testing.T) {
	stub := &procEditStub{}
	log := &auditCapture{}
	c, w := procEditCtx(http.MethodPost, "/procurement/receipts/5/reject", `{"reason":"质检破损"}`)
	procurementRejectReceipt(&app.Application{Procurement: stub, Audit: log})(c)
	if w.Code != http.StatusOK || bodyCode(t, w) != 0 {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if stub.rejectReason != "质检破损" {
		t.Fatalf("reason=%q", stub.rejectReason)
	}
	if len(log.events) != 1 || log.events[0].Action != "状态变更" ||
		log.events[0].TargetType != "procurement_receipt" || log.events[0].Detail["reason"] != "质检破损" {
		t.Fatalf("audit=%+v", log.events)
	}
	// 空请求体 → 缺省文案
	stub2 := &procEditStub{}
	c2, w2 := procEditCtx(http.MethodPost, "/procurement/receipts/5/reject", "")
	procurementRejectReceipt(&app.Application{Procurement: stub2, Audit: log})(c2)
	if w2.Code != http.StatusOK || stub2.rejectReason != "入库驳回" {
		t.Fatalf("default reason=%q status=%d", stub2.rejectReason, w2.Code)
	}
}

// E 反:CONFIRMED 不可驳回 → 40900,不写审计。
func TestRejectReceiptHandler_Confirmed40900(t *testing.T) {
	stub := &procEditStub{rejErr: fmt.Errorf("procurement: receipt 5 status=CONFIRMED: %w", procurement.ErrStateConflict)}
	log := &auditCapture{}
	c, w := procEditCtx(http.MethodPost, "/procurement/receipts/5/reject", `{"reason":"x"}`)
	procurementRejectReceipt(&app.Application{Procurement: stub, Audit: log})(c)
	if code := bodyCode(t, w); code != 40900 {
		t.Fatalf("code=%v body=%s", code, w.Body.String())
	}
	if len(log.events) != 0 {
		t.Fatalf("失败路径不得写审计: %+v", log.events)
	}
}

// E 反:原因超 255 字 → 42200(域层 ErrInvalidInput)。
func TestRejectReceiptHandler_ReasonTooLong42200(t *testing.T) {
	long := make([]byte, 256)
	for i := range long {
		long[i] = 'a'
	}
	stub := &procEditStub{rejErr: fmt.Errorf("procurement: receipt 5 reject reason 256 bytes: %w", procurement.ErrInvalidInput)}
	c, w := procEditCtx(http.MethodPost, "/procurement/receipts/5/reject", `{"reason":"`+string(long)+`"}`)
	procurementRejectReceipt(&app.Application{Procurement: stub, Audit: &auditCapture{}})(c)
	if code := bodyCode(t, w); code != 42200 {
		t.Fatalf("code=%v body=%s", code, w.Body.String())
	}
}
