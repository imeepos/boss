package adminapi

import (
	"fmt"
	"io"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// monthlyImportMaxBytes 上传 CSV 上限(51 区域×多年,10MB 足够)。
const monthlyImportMaxBytes = 10 << 20

// monthlyImportHandler POST /monthly/{table}/import:multipart CSV 上传导入
// (表头与 docs/books 模板逐字节一致;幂等 upsert;行级错误报告)。
// 留痕复用 import_tasks(数据导入中心可见)。
func monthlyImportHandler(a *app.Application, kind string) gin.HandlerFunc {
	return func(c *gin.Context) {
		fh, err := c.FormFile("file")
		if err != nil {
			respond(c, apitypes.CodeInvalidParam, gin.H{"reason": "file 字段缺失"})
			return
		}
		if fh.Size > monthlyImportMaxBytes {
			respond(c, apitypes.CodeInvalidParam, gin.H{"reason": "CSV 超过 10MB 上限"})
			return
		}
		src, err := fh.Open()
		if err != nil {
			respondErr(c, err)
			return
		}
		defer src.Close()
		data, err := io.ReadAll(src)
		if err != nil {
			respondErr(c, err)
			return
		}
		res, err := a.Monthly.ImportCSV(c.Request.Context(), kind, data)
		if err != nil {
			respondErr(c, err)
			return
		}
		detail := map[string]any{"imported": res.Imported, "failed": res.Failed, "total": res.Total}
		if len(res.Errors) > 0 {
			detail["errors"] = res.Errors
		}
		// 导入留痕:失败路径也要可 grep([monthly] IMPORT FAILED)。
		if terr := a.User.RecordImportTask(c.Request.Context(), "monthly-"+kind,
			httpx.ClaimsAccountID(c), res.Total, res.Imported, res.Failed, 0, detail, ""); terr != nil {
			fmt.Printf("[monthly] IMPORT FAILED table=%s record task: %v payload=%+v\n", kind, terr, detail)
		}
		code := apitypes.CodeOK
		if res.Failed > 0 {
			code = apitypes.CodeInvalidParam
		}
		respond(c, code, res)
	}
}

// monthlyExportHandler GET /monthly/{table}/export?month=:CSV 导出
// (与导入完全同格式同表头,BOM+CRLF,行序 month,region)。
func monthlyExportHandler(a *app.Application, kind string) gin.HandlerFunc {
	return func(c *gin.Context) {
		month := c.Query("month")
		out, err := a.Monthly.ExportCSV(c.Request.Context(), kind, month)
		if err != nil {
			respondErr(c, err)
			return
		}
		tag2 := month
		if tag2 == "" {
			tag2 = "all"
		}
		name := "monthly-" + strings.ReplaceAll(kind, "-", "_") + "-" + tag2 + ".csv"
		c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=\"%s\"", name))
		c.Data(200, "text/csv; charset=utf-8", out)
	}
}
