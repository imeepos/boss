package adminapi

// 地址列表过滤回归:needsReview=1 治理队列读取路径(fields.md §1.5.0b 契约对账)。
// 互斥优先级与前端 toggle 对齐:needsReview 优先,其次 unlinked,缺省走 parentId 树。

import (
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/auth"
)

// newRouterForAddress Register 全量路由(地址列表挂 /api/admin/v1/addresses)。
func newRouterForAddress(u *fakeUser, mgr *auth.Manager) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	Register(r, &app.Application{User: u}, mgr)
	return r
}

// TestAddrListNeedsReviewFilter needsReview=1 只回治理节点(此前参数被忽略返回全量,端到端断链)。
func TestAddrListNeedsReviewFilter(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	want := []user.Address{{ID: 4006, Level: 1, Name: "e2e-马尼拉市"}, {ID: 4010, Level: 5, Name: "3号楼"}}
	u := &fakeUser{permOk: true, needsReview: want}
	r := newRouterForAddress(u, mgr)
	w := getJSON(t, r, "/api/admin/v1/addresses?needsReview=1", authToken(t, mgr))
	if w.Code != http.StatusOK {
		t.Fatalf("status=%d", w.Code)
	}
	if !u.needsReviewOn || u.listCalled || u.unlinkedCall {
		t.Fatalf("应走 ListNeedsReview 分支: nr=%v list=%v unlinked=%v", u.needsReviewOn, u.listCalled, u.unlinkedCall)
	}
	body := w.Body.String()
	if !strings.Contains(body, `"e2e-马尼拉市"`) || !strings.Contains(body, `"3号楼"`) {
		t.Fatalf("治理节点未回传: %s", body)
	}
}

// TestAddrListDefaultsUnchanged needsReview=0/缺省行为不变:走 parentId 树分支。
func TestAddrListDefaultsUnchanged(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	for _, q := range []string{"", "?needsReview=0", "?parentId=42"} {
		u := &fakeUser{permOk: true, needsReview: []user.Address{{ID: 1}}}
		r := newRouterForAddress(u, mgr)
		w := getJSON(t, r, "/api/admin/v1/addresses"+q, authToken(t, mgr))
		if w.Code != http.StatusOK {
			t.Fatalf("q=%q status=%d", q, w.Code)
		}
		if u.needsReviewOn || u.unlinkedCall || !u.listCalled {
			t.Fatalf("q=%q 应走 ListAddresses 分支", q)
		}
		if q == "?parentId=42" && u.listParentID != 42 {
			t.Fatalf("parentId 未透传: %d", u.listParentID)
		}
		if q == "" && strings.Contains(w.Body.String(), "e2e-") {
			t.Fatalf("缺省不应回治理节点: %s", w.Body.String())
		}
	}
}

// TestAddrListUnlinkedBranch unlinked=1 仍走未挂接根分支。
func TestAddrListUnlinkedBranch(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	u := &fakeUser{permOk: true}
	r := newRouterForAddress(u, mgr)
	w := getJSON(t, r, "/api/admin/v1/addresses?unlinked=1", authToken(t, mgr))
	if w.Code != http.StatusOK || !u.unlinkedCall || u.needsReviewOn || u.listCalled {
		t.Fatalf("unlinked 分支失真: status=%d nr=%v list=%v unl=%v", w.Code, u.needsReviewOn, u.listCalled, u.unlinkedCall)
	}
}

// TestAddrListNeedsReviewPriority 参数并存时 needsReview 优先(与前端「待治理过滤优先」注释对齐)。
func TestAddrListNeedsReviewPriority(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	u := &fakeUser{permOk: true, needsReview: []user.Address{{ID: 7}}}
	r := newRouterForAddress(u, mgr)
	w := getJSON(t, r, "/api/admin/v1/addresses?needsReview=1&unlinked=1", authToken(t, mgr))
	if w.Code != http.StatusOK || !u.needsReviewOn || u.unlinkedCall {
		t.Fatalf("并存应 needsReview 优先: status=%d nr=%v unl=%v", w.Code, u.needsReviewOn, u.unlinkedCall)
	}
}

// TestAddrNearest 契约:GET /addresses/nearest 透传半径(缺省 500);未命中 data:null;非法参数 42200。
func TestAddrNearest(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	t.Run("命中透传", func(t *testing.T) {
		u := &fakeUser{permOk: true, nearest: &user.AddressNearest{ID: 9, Path: "ph.ncr.manila", Name: "马尼拉", Level: 2, DistanceM: 120.5}}
		r := newRouterForAddress(u, mgr)
		w := getJSON(t, r, "/api/admin/v1/addresses/nearest?lat=14.599&lng=120.984", authToken(t, mgr))
		if w.Code != http.StatusOK || !strings.Contains(w.Body.String(), `"distanceM":120.5`) {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		if u.nearestInLat != 14.599 || u.nearestInR != 500 {
			t.Fatalf("入参失真: lat=%v r=%v", u.nearestInLat, u.nearestInR)
		}
	})
	t.Run("自定义半径", func(t *testing.T) {
		u := &fakeUser{permOk: true}
		r := newRouterForAddress(u, mgr)
		getJSON(t, r, "/api/admin/v1/addresses/nearest?lat=14.5&lng=120.9&radiusM=3000", authToken(t, mgr))
		if u.nearestInR != 3000 {
			t.Fatalf("radiusM 未透传: %v", u.nearestInR)
		}
	})
	t.Run("缺参拒绝", func(t *testing.T) {
		u := &fakeUser{permOk: true}
		r := newRouterForAddress(u, mgr)
		w := getJSON(t, r, "/api/admin/v1/addresses/nearest?lat=14.5", authToken(t, mgr))
		if !strings.Contains(w.Body.String(), "42200") {
			t.Fatalf("缺 lng 应回 42200: %s", w.Body.String())
		}
	})
	t.Run("半径越界拒绝", func(t *testing.T) {
		u := &fakeUser{permOk: true}
		r := newRouterForAddress(u, mgr)
		w := getJSON(t, r, "/api/admin/v1/addresses/nearest?lat=14.5&lng=120.9&radiusM=999999", authToken(t, mgr))
		if !strings.Contains(w.Body.String(), "42200") {
			t.Fatalf("radiusM>50km 应回 42200: %s", w.Body.String())
		}
	})
}

// TestAddrSetGeom 契约:PUT /addresses/:id/geom 透传坐标并留审计;越界 422 拒。
func TestAddrSetGeom(t *testing.T) {
	mgr := auth.NewManager("s", time.Hour)
	t.Run("写入透传", func(t *testing.T) {
		u := &fakeUser{permOk: true}
		r := newRouterForAddress(u, mgr)
		w := putJSONAuth(t, r, "/api/admin/v1/addresses/9/geom", `{"lat":14.599,"lng":120.984}`, authToken(t, mgr))
		if w.Code != http.StatusOK {
			t.Fatalf("status=%d body=%s", w.Code, w.Body.String())
		}
		if u.geomID != 9 || u.geomLat != 14.599 || u.geomLng != 120.984 {
			t.Fatalf("坐标未透传: id=%d lat=%v lng=%v", u.geomID, u.geomLat, u.geomLng)
		}
	})
	t.Run("越界拒绝", func(t *testing.T) {
		u := &fakeUser{permOk: true}
		r := newRouterForAddress(u, mgr)
		w := putJSONAuth(t, r, "/api/admin/v1/addresses/9/geom", `{"lat":91,"lng":120.984}`, authToken(t, mgr))
		// envelope 惯例:HTTP 200 + 业务码 42200(CodeInvalidParam)。
		if !strings.Contains(w.Body.String(), "42200") {
			t.Fatalf("lat=91 应回 42200: %s", w.Body.String())
		}
		if u.geomID != 0 {
			t.Fatalf("越界请求不应落域层: id=%d", u.geomID)
		}
	})
}
