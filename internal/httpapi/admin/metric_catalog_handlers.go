package adminapi

// S5 指标目录与质量规则：指标 key/定义/公式/负责人/版本/血缘占位；
// 质量规则 scope/check_expr/severity/owner 持久化，异常发现后投递到补偿任务中心。
// menu:report 门禁（与补偿中心同权限）。

import (
	"errors"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/metric"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerMetricRoutes 注册指标目录路由（menu:report）。
func registerMetricRoutes(g *gin.RouterGroup, a *app.Application) {
	m := g.Group("", requirePerm(a.User, "menu:report"))
	m.GET("/metric-catalog", metricCatalogListHandler(a))
	m.GET("/metric-catalog/:key", metricCatalogGetHandler(a))
	m.PUT("/metric-catalog/:key", metricCatalogUpsertHandler(a))
	m.POST("/metric-catalog/:key/deprecate", metricCatalogDeprecateHandler(a))
	m.GET("/metric-quality-rules", metricQualityRuleListHandler(a))
	m.PUT("/metric-quality-rules/:ruleKey", metricQualityRuleUpsertHandler(a))
	m.POST("/metric-quality-rules/:ruleKey/disable", metricQualityRuleDisableHandler(a))
	m.GET("/metric-quality-scan", metricQualityScanHandler(a))
}

func metricCatalogListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		status := c.Query("status")
		if status != "" {
			switch status {
			case metric.StatusDraft, metric.StatusActive, metric.StatusDeprecated, metric.StatusArchived:
			default:
				respond(c, apitypes.CodeInvalidParam, fmt.Sprintf("invalid status: %s", status))
				return
			}
		}
		items, err := a.Metric.ListCatalog(c.Request.Context(), status)
		if err != nil {
			metricRespondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, items)
	}
}

func metricCatalogGetHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := strings.TrimSpace(c.Param("key"))
		if key == "" {
			respond(c, apitypes.CodeInvalidParam, "key required")
			return
		}
		item, err := a.Metric.GetCatalogByKey(c.Request.Context(), key)
		if err != nil {
			metricRespondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, item)
	}
}

func metricCatalogUpsertHandler(a *app.Application) gin.HandlerFunc {
	type req struct {
		Name           string   `json:"name"`
		Description    string   `json:"description"`
		Formula        string   `json:"formula"`
		Unit           string   `json:"unit"`
		Dimensions     []string `json:"dimensions"`
		RefreshCadence string   `json:"refreshCadence"`
		Owner          string   `json:"owner"`
		Domain         string   `json:"domain"`
		Status         string   `json:"status"`
	}
	return func(c *gin.Context) {
		key := strings.TrimSpace(c.Param("key"))
		if key == "" {
			respond(c, apitypes.CodeInvalidParam, "key required")
			return
		}
		var r req
		if err := c.ShouldBindJSON(&r); err != nil {
			respond(c, apitypes.CodeInvalidParam, err.Error())
			return
		}
		if r.Status == "" {
			r.Status = metric.StatusDraft
		}
		switch r.Status {
		case metric.StatusDraft, metric.StatusActive, metric.StatusDeprecated:
		default:
			respond(c, apitypes.CodeInvalidParam, fmt.Sprintf("invalid status: %s", r.Status))
			return
		}
		item, err := a.Metric.UpsertCatalog(c.Request.Context(), metric.CatalogEntry{
			Key:            key,
			Name:           r.Name,
			Description:    r.Description,
			Formula:        r.Formula,
			Unit:           r.Unit,
			Dimensions:     r.Dimensions,
			RefreshCadence: r.RefreshCadence,
			Owner:          r.Owner,
			Domain:         r.Domain,
			Status:         r.Status,
		})
		if err != nil {
			metricRespondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, item)
	}
}

func metricCatalogDeprecateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		key := strings.TrimSpace(c.Param("key"))
		if key == "" {
			respond(c, apitypes.CodeInvalidParam, "key required")
			return
		}
		item, err := a.Metric.DeprecateCatalog(c.Request.Context(), key)
		if err != nil {
			metricRespondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, item)
	}
}

func metricQualityRuleListHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		enabledOnly := c.Query("enabled") == "true"
		items, err := a.Metric.ListQualityRules(c.Request.Context(), enabledOnly)
		if err != nil {
			metricRespondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, items)
	}
}

func metricQualityRuleUpsertHandler(a *app.Application) gin.HandlerFunc {
	type req struct {
		Name          string `json:"name"`
		Scope         string `json:"scope"`
		CheckExpr     string `json:"checkExpr"`
		ThresholdExpr string `json:"thresholdExpr"`
		Severity      string `json:"severity"`
		Owner         string `json:"owner"`
		Enabled       bool   `json:"enabled"`
	}
	return func(c *gin.Context) {
		ruleKey := strings.TrimSpace(c.Param("ruleKey"))
		if ruleKey == "" {
			respond(c, apitypes.CodeInvalidParam, "ruleKey required")
			return
		}
		var r req
		if err := c.ShouldBindJSON(&r); err != nil {
			respond(c, apitypes.CodeInvalidParam, err.Error())
			return
		}
		switch r.Severity {
		case metric.SeverityInfo, metric.SeverityWarn, metric.SeverityCritical:
		case "":
			r.Severity = metric.SeverityWarn
		default:
			respond(c, apitypes.CodeInvalidParam, fmt.Sprintf("invalid severity: %s", r.Severity))
			return
		}
		item, err := a.Metric.UpsertQualityRule(c.Request.Context(), metric.QualityRule{
			RuleKey:       ruleKey,
			Name:          r.Name,
			Scope:         r.Scope,
			CheckExpr:     r.CheckExpr,
			ThresholdExpr: r.ThresholdExpr,
			Severity:      r.Severity,
			Owner:         r.Owner,
			Enabled:       r.Enabled,
		})
		if err != nil {
			metricRespondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, item)
	}
}

func metricQualityRuleDisableHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		ruleKey := strings.TrimSpace(c.Param("ruleKey"))
		if ruleKey == "" {
			respond(c, apitypes.CodeInvalidParam, "ruleKey required")
			return
		}
		if err := a.Metric.DisableQualityRule(c.Request.Context(), ruleKey); err != nil {
			metricRespondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, nil)
	}
}

func metricQualityScanHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		scope := strings.TrimSpace(c.Query("scope"))
		violations, err := a.Metric.ScanQuality(c.Request.Context(), scope)
		if err != nil {
			metricRespondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, violations)
	}
}

// metricRespondErr 映射指标域错误到 HTTP 响应。
func metricRespondErr(c *gin.Context, err error) {
	if errors.Is(err, metric.ErrNotFound) {
		respond(c, apitypes.CodeNotFound, err.Error())
		return
	}
	respond(c, apitypes.CodeInternal, err.Error())
}
