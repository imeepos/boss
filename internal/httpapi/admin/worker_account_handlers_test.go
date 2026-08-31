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
