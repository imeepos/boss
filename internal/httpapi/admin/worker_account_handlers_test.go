package adminapi

// 师傅账号录入契约测试(worker.yaml POST /workers + PUT /workers/{workerId}/password):
// 录入落主档+登录密码;密码/工号/关联 ID 必填;重置密码透传域校验(过短 42200)。

import (
	"net/http"
	"testing"
	"time"

	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/pkg/apitypes"
)

func TestCreateWorkerHandler(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	f := &fakeWorkerOps{}
	r := newWorkerRouter(f, mgr)
	tok := authToken(t, mgr)

	w := postBodyAuth(t, r, "/api/admin/v1/workers",
		`{"staffNo":"WK-2001","name":"赵师傅","groupId":1,"regionId":11,"phone":"13900002233","password":"secret-66"}`, tok)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if f.createdWorker == nil || f.createdWorker.StaffNo != "WK-2001" ||
		f.createdWorker.Name != "赵师傅" || f.createdWorker.GroupID != 1 ||
		f.createdWorker.RegionID != 11 || f.createdWorker.Phone != "13900002233" ||
		f.createdWorker.Status != 1 {
		t.Fatalf("createdWorker=%+v", f.createdWorker)
	}
	if f.createdPassword != "secret-66" {
		t.Fatalf("password=%q", f.createdPassword)
	}

	// 缺密码拒绝(42200):登录密码是录入必填项。
	f2 := &fakeWorkerOps{}
	r2 := newWorkerRouter(f2, mgr)
	w = postBodyAuth(t, r2, "/api/admin/v1/workers",
		`{"staffNo":"WK-2002","name":"钱师傅","groupId":1,"regionId":11,"phone":"13900002244"}`, authToken(t, mgr))
	if w.Code != http.StatusOK || envCode(t, w) != apitypes.CodeInvalidParam {
		t.Fatalf("missing password should 42200: status=%d body=%s", w.Code, w.Body.String())
	}
	if f2.createdWorker != nil {
		t.Fatalf("rejected create must not reach domain: %+v", f2.createdWorker)
	}
}

func TestSetWorkerPasswordHandler(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	f := &fakeWorkerOps{}
	r := newWorkerRouter(f, mgr)
	tok := authToken(t, mgr)

	w := putAuth(t, r, "/api/admin/v1/workers/5/password", `{"password":"new-pass-66"}`, tok)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if f.pwdWorkerID != 5 || f.pwdPassword != "new-pass-66" {
		t.Fatalf("pwdWorkerID=%d pwdPassword=%q", f.pwdWorkerID, f.pwdPassword)
	}
}

// TestCreateWorkerWithRegionIds 契约(000175):录入支持 regionIds 多负责区域,首位为主区域。
func TestCreateWorkerWithRegionIds(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	f := &fakeWorkerOps{}
	r := newWorkerRouter(f, mgr)
	tok := authToken(t, mgr)

	w := postBodyAuth(t, r, "/api/admin/v1/workers",
		`{"staffNo":"WK-2003","name":"孙师傅","groupId":1,"regionIds":[12,11],"phone":"13900002255","password":"secret-66"}`, tok)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if f.createdWorker == nil || f.createdWorker.RegionID != 12 ||
		len(f.createdWorker.RegionIDs) != 2 || f.createdWorker.RegionIDs[0] != 12 || f.createdWorker.RegionIDs[1] != 11 {
		t.Fatalf("createdWorker=%+v", f.createdWorker)
	}

	// regionId 与 regionIds 均缺省拒绝(42200)。
	f2 := &fakeWorkerOps{}
	w2 := postBodyAuth(t, newWorkerRouter(f2, mgr), "/api/admin/v1/workers",
		`{"staffNo":"WK-2004","name":"李师傅","groupId":1,"phone":"13900002266","password":"secret-66"}`, authToken(t, mgr))
	if w2.Code != http.StatusOK || envCode(t, w2) != apitypes.CodeInvalidParam {
		t.Fatalf("missing regions should 42200: status=%d body=%s", w2.Code, w2.Body.String())
	}
	if f2.createdWorker != nil {
		t.Fatalf("rejected create must not reach domain: %+v", f2.createdWorker)
	}
}

// TestSetWorkerRegionsHandler 契约(000175):PUT /workers/{id}/regions 覆盖式配置负责区域。
func TestSetWorkerRegionsHandler(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	f := &fakeWorkerOps{}
	r := newWorkerRouter(f, mgr)
	tok := authToken(t, mgr)

	w := putAuth(t, r, "/api/admin/v1/workers/7/regions", `{"regionIds":[12,11,13]}`, tok)
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
	}
	if f.regionsWorkerID != 7 || len(f.regionsSet) != 3 ||
		f.regionsSet[0] != 12 || f.regionsSet[1] != 11 || f.regionsSet[2] != 13 {
		t.Fatalf("regionsWorkerID=%d regionsSet=%v", f.regionsWorkerID, f.regionsSet)
	}

	// 非法元素(0/负数)拒绝(42200),不触达域层。
	f2 := &fakeWorkerOps{}
	w2 := putAuth(t, newWorkerRouter(f2, mgr), "/api/admin/v1/workers/7/regions",
		`{"regionIds":[12,0]}`, authToken(t, mgr))
	if w2.Code != http.StatusOK || envCode(t, w2) != apitypes.CodeInvalidParam {
		t.Fatalf("invalid regionIds should 42200: status=%d body=%s", w2.Code, w2.Body.String())
	}
	if f2.regionsSet != nil {
		t.Fatalf("rejected set must not reach domain: %v", f2.regionsSet)
	}
}
