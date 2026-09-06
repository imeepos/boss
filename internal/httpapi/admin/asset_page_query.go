package adminapi

// 资产/标签列表分页查询参数解析(P3-T1):offset/limit 归一钳制+排序白名单 400。
// 白名单校验在进 domain 前完成,是防注入第一道闸;domain 层 sortCol 再兜底一道。

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/asset"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// parseListQuery 解析列表分页查询参数:
//   - offset 非法/负值归 0;limit 非法/缺省回 50,>200 钳 200(契约:钳制不报错);
//   - status/type/q 去首尾空格,modelId 非法归 0(不过滤);
//   - sort 空=created_at;不在白名单回 HTTP 400(envelope code 42200)并返回 false。
func parseListQuery(c *gin.Context, whitelist map[string]string) (asset.ListQuery, bool) {
	q := asset.ListQuery{
		Offset:  int(queryInt64(c, "offset")),
		Limit:   int(queryInt64(c, "limit")),
		Status:  strings.TrimSpace(c.Query("status")),
		Type:    strings.TrimSpace(c.Query("type")),
		ModelID: queryInt64(c, "modelId"),
		Q:       strings.TrimSpace(c.Query("q")),
		Sort:    c.Query("sort"),
	}
	if q.Offset < 0 {
		q.Offset = 0
	}
	if q.ModelID < 0 {
		q.ModelID = 0
	}
	if q.Limit <= 0 {
		q.Limit = asset.ListDefaultLimit
	}
	if q.Limit > asset.ListMaxLimit {
		q.Limit = asset.ListMaxLimit
	}
	key := q.Sort
	if key == "" {
		key = "created_at"
	}
	if _, ok := whitelist[key]; !ok {
		respondBadRequest(c, "sort must be one of: created_at, asset_code, status")
		return q, false
	}
	return q, true
}

// respondBadRequest HTTP 400+统一 envelope:排序白名单是硬契约,调用方必须显式
// 看到 400(与 HTTP 200+42200 的软校验错区分),前端网关错误条可直读。
func respondBadRequest(c *gin.Context, reason string) {
	c.JSON(http.StatusBadRequest, gin.H{
		"code": apitypes.CodeInvalidParam,
		"msg":  apitypes.CodeInvalidParam.Message(),
		"data": gin.H{"reason": reason},
	})
}
