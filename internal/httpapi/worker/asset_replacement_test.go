// 师傅端换新任务 handler 测试:列表归属/完成回写/资产联动/越权拒止。
package workerapi

import (
	"context"
	"testing"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/internal/domain/portal"
	"github.com/ymm-001/boss/internal/domain/worker"
)

// fakeReplacementAsset 桩 asset.AssetService:内嵌接口,仅实现被测方法。
type fakeReplacementAsset struct {
	asset.AssetService
	repl      *asset.Replacement
	byWorker  []asset.Replacement
	statusSet map[int64]string
	logs      []asset.AssetLifecycle
}

func (f *fakeReplacementAsset) GetReplacement(_ context.Context, id int64) (*asset.Replacement, error) {
	if f.repl == nil || f.repl.ID != id {
		return nil, asset.ErrNotFound
	}
	return f.repl, nil
}
func (f *fakeReplacementAsset) ListReplacementsByWorker(_ context.Context, workerID int64) ([]asset.Replacement, error) {
	return f.byWorker, nil
}
func (f *fakeReplacementAsset) CompleteReplacement(_ context.Context, id int64, result string) (*asset.Replacement, error) {
	f.repl.Status = result
	return f.repl, nil
}
func (f *fakeReplacementAsset) SetAssetStatus(_ context.Context, assetID int64, status string) error {
	if f.statusSet == nil {
		f.statusSet = map[int64]string{}
	}
	f.statusSet[assetID] = status
	return nil
}
func (f *fakeReplacementAsset) GetAsset(_ context.Context, id int64) (*asset.Asset, error) {
	return &asset.Asset{AssetID: id, AssetCode: "A-20260002"}, nil
}
func (f *fakeReplacementAsset) ListTags(context.Context) ([]asset.Tag, error) {
	return []asset.Tag{{TagID: 9, EpcCode: "EPC-NEW", BoundAssetID: 3}}, nil
}
func (f *fakeReplacementAsset) AppendLifecycle(_ context.Context, l asset.AssetLifecycle) (int64, error) {
	f.logs = append(f.logs, l)
	return int64(len(f.logs)), nil
}

// fakeReplacementEvents 桩 worker.WorkerEventService:仅 AppendReplaceLog。
type fakeReplacementEvents struct {
	worker.WorkerEventService
	logs []worker.ReplaceLog
}

func (f *fakeReplacementEvents) AppendReplaceLog(_ context.Context, r worker.ReplaceLog) (int64, error) {
	f.logs = append(f.logs, r)
	return 1, nil
}

func replacementTestRouter(t *testing.T, fa *fakeReplacementAsset, fe *fakeReplacementEvents) *gin.Engine {
	return replacementTestRouterWith(t, fa, fe)
}

func replacementTestRouterWith(t *testing.T, fa *fakeReplacementAsset, fe *fakeReplacementEvents) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	t.Setenv("BOSS_JWT_SECRET", "portal-test-secret")
	r := gin.New()
	a := &app.Application{
		Asset: fa, WorkerEvent: fe, Portal: portal.NewMemory(),
		Automation: app.NewAutomation(nil, nil),
	}
	Register(r, a, newWorkerJWTManager())
	return r
}

func TestWorkerReplacementFlow(t *testing.T) {
	rpl := &asset.Replacement{ID: 1, ReplacementNo: "RPL-1", AssetID: 2,
		LegalEntityName: "主品牌", Reason: "光猫故障", Priority: "MEDIUM", Status: "DOING", WorkerID: 7}
	fa := &fakeReplacementAsset{repl: rpl, byWorker: []asset.Replacement{*rpl}}
	fe := &fakeReplacementEvents{}
	r := replacementTestRouter(t, fa, fe)
	tok := mustWorkerTokenStr(t, 7)

	// 列表:本人 DOING 任务,带资产码展示快照。
	res := portalWorkerDo(r, "GET", "/api/worker/v1/replacements", "", tok)
	items := res["data"].(map[string]any)["items"].([]any)
	if len(items) != 1 || items[0].(map[string]any)["assetCode"] != "A-20260002" {
		t.Fatalf("list=%v", res)
	}

	// 完成:落换件流水(RPL 号) + DONE + 资产联动(旧2→MAINTENANCE,新3→DEPLOYED)。
	res = portalWorkerDo(r, "POST", "/api/worker/v1/replacements/1/complete",
		`{"oldEpc":"EPC-OLD","newEpc":"EPC-NEW","result":"SUCCESS"}`, tok)
	if res["code"].(float64) != 0 || res["data"].(map[string]any)["status"] != "DONE" {
		t.Fatalf("complete=%v", res)
	}
	if len(fe.logs) != 1 || fe.logs[0].TicketNo != "RPL-1" || fe.logs[0].WorkerID != 7 {
		t.Fatalf("logs=%+v", fe.logs)
	}
	if fa.statusSet[2] != "MAINTENANCE" || fa.statusSet[3] != "DEPLOYED" || len(fa.logs) != 2 {
		t.Fatalf("statusSet=%v logs=%d", fa.statusSet, len(fa.logs))
	}

	// 越权:师傅 9 访问师傅 7 的任务 → 404 防探测。
	tok9 := mustWorkerTokenStr(t, 9)
	res = portalWorkerDo(r, "POST", "/api/worker/v1/replacements/1/complete",
		`{"newEpc":"EPC-NEW","result":"SUCCESS"}`, tok9)
	if res["code"].(float64) != 40400 {
		t.Fatalf("cross worker=%v", res)
	}

	// 非法 result → 42200 参数非法。
	res = portalWorkerDo(r, "POST", "/api/worker/v1/replacements/1/complete",
		`{"newEpc":"EPC-NEW","result":"DOING"}`, tok)
	if res["code"].(float64) != 42200 {
		t.Fatalf("bad result=%v", res)
	}
}

func TestWorkerReplacementOwnershipMismatch(t *testing.T) {
	// 单未派给任何人(workerID=0)时师傅不可见亦不可完成。
	fa := &fakeReplacementAsset{repl: &asset.Replacement{ID: 2, ReplacementNo: "RPL-2",
		AssetID: 5, Status: "PENDING"}}
	fe := &fakeReplacementEvents{}
	r := replacementTestRouter(t, fa, fe)
	tok := mustWorkerTokenStr(t, 7)

	res := portalWorkerDo(r, "GET", "/api/worker/v1/replacements", "", tok)
	if items := res["data"].(map[string]any)["items"].([]any); len(items) != 0 {
		t.Fatalf("unassigned task leaked: %v", res)
	}
	res = portalWorkerDo(r, "POST", "/api/worker/v1/replacements/2/complete",
		`{"newEpc":"EPC-NEW","result":"SUCCESS"}`, tok)
	if res["code"].(float64) != 40400 {
		t.Fatalf("pending task=%v", res)
	}
	if len(fe.logs) != 0 {
		t.Fatalf("log leaked: %+v", fe.logs)
	}
}

// mustWorkerTokenStr 签发指定师傅的测试 JWT。
func mustWorkerTokenStr(t *testing.T, workerID int64) string {
	t.Helper()
	tok, err := signWorkerToken(workerID, "张师傅")
	if err != nil {
		t.Fatalf("sign: %v", err)
	}
	return tok
}
