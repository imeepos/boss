package adminapi

// 实名审核中心路由(后端聚合列表):横跨 customer/worker 两类主体,join 主档取实名/手机号快照。
// 不替代既有 /customers/:id/real-name/* 和 /workers/:workerId/real-name/* 的单主体核验端点;
// 该页面仅聚合"待办 + 历史轨迹",行内 PASS/驳回时按 subject_type 转发到对应单主体核验端点逻辑。

import (
	"context"
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/internal/pkg/middleware"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerRealnameReviewRoutes 注册实名审核中心路由(menu:realname-review)。
func registerRealnameReviewRoutes(g *gin.RouterGroup, a *app.Application) {
	perm := requirePerm(a.User, "menu:realname-review")
	g.GET("/verifications", perm, listVerificationsHandler(a))
	g.POST("/verifications/:subjectType/:subjectId/verify", perm, verifyFromCenterHandler(a))
}

// VerificationRow 列表项:verifications 单行 + 主体(customer/worker)快照。
// 证件号与手机号不回明文,前端展示走 masked 函数或后端按掩码规则。
type VerificationRow struct {
	ID              int64  `json:"id"`
	SubjectType     string `json:"subjectType"`
	SubjectID       int64  `json:"subjectId"`
	SubjectName     string `json:"subjectName"`
	SubjectPhone    string `json:"subjectPhone"`
	IDCardNoMasked  string `json:"idCardNoMasked"`
	Method          string `json:"method"`
	RealName        string `json:"realName"`
	Result          string `json:"result"`
	RejectReason    string `json:"rejectReason"`
	VerifiedAt      string `json:"verifiedAt"`
	OperatorName    string `json:"operatorName"`
	OperatorAccount int64  `json:"operatorAccountId"`
	IDCardFrontID   int64  `json:"idCardFrontId"`
	IDCardBackID    int64  `json:"idCardBackId"`
}

// listVerificationsHandler GET /verifications:实名审核中心列表。
// query: subjectType(customer|worker|”=全部) + result(PENDING|PASS|FAIL|”=全部)
//   - keyword(姓名/手机号/证件号 substring) + page(默认 1) + pageSize(默认 20,上限 100)。
func listVerificationsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		subjectType := strings.TrimSpace(c.Query("subjectType"))
		result := strings.TrimSpace(c.Query("result"))
		keyword := strings.TrimSpace(c.Query("keyword"))
		page, ok := httpx.ParseQueryParamInt64(c, "page", 1)
		if !ok || page < 1 {
			page = 1
		}
		pageSize, ok := httpx.ParseQueryParamInt64(c, "pageSize", 20)
		if !ok || pageSize < 1 {
			pageSize = 20
		}
		if pageSize > 100 {
			pageSize = 100
		}

		rows, total, err := queryVerifications(c.Request.Context(), a, subjectType, result, keyword, page, pageSize)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"items":    rows,
			"total":    total,
			"page":     page,
			"pageSize": pageSize,
		})
	}
}

// queryVerifications 联表聚合 verifications ∪ customers / verifications ∪ workers。
// 客户无证件号/姓名时回退空串,师傅同理;ORDER BY verified_at DESC, id DESC。
func queryVerifications(ctx context.Context, a *app.Application, subjectType, result, keyword string, page, pageSize int64) ([]VerificationRow, int64, error) {
	var (
		conds []string
		args  []any
	)
	if subjectType == "customer" || subjectType == "worker" {
		conds = append(conds, fmt.Sprintf("v.subject_type = $%d", len(args)+1))
		args = append(args, subjectType)
	}
	if result == "PENDING" || result == "PASS" || result == "FAIL" {
		conds = append(conds, fmt.Sprintf("v.result = $%d", len(args)+1))
		args = append(args, result)
	}
	if keyword != "" {
		// 关键词命中:主体姓名/手机号/证件号 任一 substring;ILIKE 大小写不敏感。
		p := "%" + keyword + "%"
		conds = append(conds, fmt.Sprintf(
			"(COALESCE(c.name, w.name, '') ILIKE $%d OR COALESCE(c.phone, w.phone, '') ILIKE $%d OR v.id_card_no ILIKE $%d OR v.real_name ILIKE $%d)",
			len(args)+1, len(args)+1, len(args)+1, len(args)+1,
		))
		args = append(args, p)
	}
	where := ""
	if len(conds) > 0 {
		where = "WHERE " + strings.Join(conds, " AND ")
	}

	var total int64
	if err := a.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM verifications v
		   LEFT JOIN customers c ON v.subject_type='customer' AND c.id = v.subject_id
		   LEFT JOIN workers   w ON v.subject_type='worker'   AND w.id = v.subject_id
		   `+where, args...).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("realname-review: count: %w", err)
	}

	offset := (page - 1) * pageSize
	args = append(args, pageSize, offset)
	rows, err := a.Pool.Query(ctx, fmt.Sprintf(`
SELECT v.id, v.subject_type, v.subject_id,
       COALESCE(c.name, w.name, '')              AS subject_name,
       COALESCE(c.phone, w.phone, '')            AS subject_phone,
       v.id_card_no,
       v.method, COALESCE(v.real_name, ''), v.result, COALESCE(v.reject_reason, ''),
       to_char(v.verified_at, 'YYYY-MM-DD"T"HH24:MI:SS"Z"'),
       COALESCE(v.operator_name, ''),
       COALESCE(v.operator_account_id, 0),
       v.id_card_front_id, v.id_card_back_id
  FROM verifications v
  LEFT JOIN customers c ON v.subject_type='customer' AND c.id = v.subject_id
  LEFT JOIN workers   w ON v.subject_type='worker'   AND w.id = v.subject_id
  %s
 ORDER BY v.verified_at DESC, v.id DESC
 LIMIT $%d OFFSET $%d`, where, len(args)-1, len(args)), args...)
	if err != nil {
		return nil, 0, fmt.Errorf("realname-review: list: %w", err)
	}
	defer rows.Close()

	out := make([]VerificationRow, 0)
	for rows.Next() {
		var r VerificationRow
		var rawIDCard string
		if err := rows.Scan(&r.ID, &r.SubjectType, &r.SubjectID,
			&r.SubjectName, &r.SubjectPhone, &rawIDCard,
			&r.Method, &r.RealName, &r.Result, &r.RejectReason,
			&r.VerifiedAt, &r.OperatorName, &r.OperatorAccount,
			&r.IDCardFrontID, &r.IDCardBackID); err != nil {
			return nil, 0, fmt.Errorf("realname-review: scan: %w", err)
		}
		r.IDCardNoMasked = maskIDCard(rawIDCard)
		out = append(out, r)
	}
	return out, total, rows.Err()
}

// maskIDCard 证件号脱敏:保留首 4 尾 2,中间 ***;非空长度 <6 全掩。
func maskIDCard(s string) string {
	if s == "" {
		return ""
	}
	if len(s) < 6 {
		return strings.Repeat("*", len(s))
	}
	return s[:4] + strings.Repeat("*", len(s)-6) + s[len(s)-2:]
}

// verifyFromCenterHandler POST /verifications/:subjectType/:subjectId/verify:行内 PASS/FAIL,
// 转发到既有单主体核验端点(避免在两处维护审核逻辑)。
func verifyFromCenterHandler(a *app.Application) gin.HandlerFunc {
	type verifyReq struct {
		Result string `json:"result" binding:"required"`
		Reason string `json:"reason"`
	}
	return func(c *gin.Context) {
		subjectType := c.Param("subjectType")
		subjectID, ok := httpx.ParsePathParamInt64(c, "subjectId")
		if !ok {
			return
		}
		var req verifyReq
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		claims := c.MustGet(middleware.CtxClaims).(*auth.Claims)
		switch subjectType {
		case "customer":
			if err := a.CustomerRealName.Verify(c.Request.Context(), subjectID, req.Result, req.Reason, claims.Username, claims.AccountID); err != nil {
				respondErr(c, err)
				return
			}
		case "worker":
			if err := a.WorkerRealName.Verify(c.Request.Context(), subjectID, req.Result, claims.Username, claims.AccountID); err != nil {
				respondErr(c, err)
				return
			}
		default:
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "subjectType must be customer|worker"})
			return
		}
		httpx.RecordAudit(a, c, "realname_review.verify", "verification",
			fmt.Sprintf("%s/%d", subjectType, subjectID),
			gin.H{"result": req.Result, "reason": req.Reason})
		resolveRealnameTodo(a, c, subjectType, subjectID)
		if subjectType == "customer" {
			notifyCustomerRealnameResult(a, c, subjectID, req.Result, req.Reason)
		} else if subjectType == "worker" {
			notifyWorkerRealnameResult(a, c, subjectID, req.Result)
		}
		respond(c, apitypes.CodeOK, gin.H{"result": req.Result})
	}
}
