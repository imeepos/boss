package adminapi

// W3 ODN 资源链批量导入路由(menu:odn 门禁;契约 admin/odn.yaml)。
// 导入中心历史经 user.RecordImportTask(kind=odn-resource-chain);写操作入审计;
// 行级失败由域层留 [odn-import] 可 grep 日志,响应逐行回带原因与行号。

import (
	"log"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/odn"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// odnChainImportReq 资源链导入请求体(exampleRowCount=说明页示例行数)。
type odnChainImportReq struct {
	ExampleRowCount int                    `json:"exampleRowCount"`
	Rows            []odn.ResourceChainRow `json:"rows" binding:"required,min=1,max=5000"`
}

// registerODNResourceChainRoutes 注册资源链导入/巡检/清单路由。
func registerODNResourceChainRoutes(g *gin.RouterGroup, a *app.Application, perm gin.HandlerFunc) {
	g.POST("/odn/resource-chains/import", perm, odnImportResourceChainsHandler(a))
	g.GET("/odn/resource-chains", perm, odnListResourceChainsHandler(a))
	g.GET("/odn/resource-chains/patrol", perm, odnChainPatrolHandler(a))
	g.POST("/odn/resource-chains/backfill-split", perm, odnBackfillSplitHandler(a))
}

// odnImportResourceChainsHandler 批量导入:说明页六规则校验 + 展开入库 + 逐行结果。
func odnImportResourceChainsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req odnChainImportReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		res, err := a.ODN.ImportResourceChains(c.Request.Context(), odn.ChainImportInput{
			ExampleRowCount: req.ExampleRowCount, Rows: req.Rows,
		})
		if err != nil {
			respondErr(c, err)
			return
		}
		recordChainImportTask(a, c, res)
		httpx.RecordAudit(a, c, "odn.resource-chain.import", "odn_resource_chain", res.BatchNo,
			map[string]any{"total": res.Total, "imported": res.Imported, "failed": res.Failed,
				"skipped": res.Skipped, "duplicate": res.Duplicate})
		emitTask(c.Request.Context(), a, refImporter,
			"odn-chain-"+strconv.FormatInt(time.Now().Unix(), 10),
			"ODN 资源链导入: 成功 "+strconv.Itoa(res.Imported)+" / 失败 "+strconv.Itoa(res.Failed)+" / 示例跳过 "+strconv.Itoa(res.Skipped)+" / 重复 "+strconv.Itoa(res.Duplicate),
			linkImporter, res.Failed > 0)
		respond(c, apitypes.CodeOK, gin.H{"result": res})
	}
}

// odnBackfillSplitHandler POST /odn/resource-chains/backfill-split:分光比回写建模(幂等全量重建)。
// W5 容量写路径;写操作入审计;失败由域层留 [odn-split-backfill] 可 grep 日志。
func odnBackfillSplitHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		res, err := a.ODN.BackfillSplitCapacity(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "odn.resource-chain.backfill-split", "odn_device_split_capacity", "rebuild",
			map[string]any{"devicesModeled": res.DevicesModeled, "level1": res.Level1,
				"level2": res.Level2, "unresolvedCodes": res.UnresolvedCodes})
		respond(c, apitypes.CodeOK, gin.H{"result": res})
	}
}

// recordChainImportTask 落导入中心历史;失败留 [odn-import] 日志不阻断响应(导入结果为准)。
func recordChainImportTask(a *app.Application, c *gin.Context, res *odn.ChainImportResult) {
	detail := map[string]any{"batchNo": res.BatchNo, "duplicate": res.Duplicate}
	if res.Failed > 0 {
		failed := make([]map[string]any, 0, res.Failed)
		for _, r := range res.Rows {
			if r.Status == odn.ChainRowFailed {
				failed = append(failed, map[string]any{"row": r.RowNo, "reason": r.Reason})
			}
		}
		detail["failedRows"] = failed
	}
	if err := a.User.RecordImportTask(c.Request.Context(), "odn-resource-chain",
		httpx.ClaimsAccountID(c), res.Total, res.Imported, res.Failed, res.Skipped, detail, ""); err != nil {
		log.Printf("[odn-import] RECORD TASK FAILED batch=%s reason=%v", res.BatchNo, err)
	}
}

// odnListResourceChainsHandler 链行清单(batch/limit 查询参数,验收核对用)。
func odnListResourceChainsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		limit, _ := strconv.Atoi(c.Query("limit"))
		rows, err := a.ODN.ListResourceChains(c.Request.Context(), c.Query("batch"), limit)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": rows})
	}
}

// odnChainPatrolHandler 孤儿/半链巡检(导入完成后链上引用的资源必须存在)。
func odnChainPatrolHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		p, err := a.ODN.PatrolResourceChains(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, p)
	}
}
