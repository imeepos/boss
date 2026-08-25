package adminapi

// 聚合搜索 handler:按账号持有权限并行检索 客户/用户/师傅/订单 四域,分组返回。
// 权限映射:customer→menu:customer, user→menu:user, worker→menu:dispatch, order→menu:order。
// 客户/订单域透传数据范围(子公司+区域),用户/师傅域与既有列表接口口径一致(不传范围)。

import (
	"context"
	"reflect"
	"sync"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/customer"
	"github.com/ymm-001/boss/internal/domain/order"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// searchMaxHits 单域检索条数上限(命令面板展示 top N)。
const searchMaxHits = 5

// searchTask 单域检索任务:域标识 + 门禁权限码 + 实际查询。
type searchTask struct {
	domain string
	perm   string
	fetch  func(ctx context.Context) (any, error)
}

// searchAggregateHandler GET /search?keyword=:四域聚合检索。
func searchAggregateHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		kw := c.Query("keyword")
		if kw == "" {
			respond(c, apitypes.CodeInvalidParam, gin.H{"error": "keyword is required"})
			return
		}
		ctx := c.Request.Context()
		accountID := httpx.ClaimsAccountID(c)
		scope, err := a.User.GetDataScope(ctx, accountID)
		if err != nil {
			respondErr(c, err)
			return
		}
		tasks := searchTasks(a, kw, scope)
		respond(c, apitypes.CodeOK, gin.H{"groups": runSearchTasks(ctx, a, accountID, tasks)})
	}
}

// searchTasks 组装四域检索任务(声明顺序即返回顺序)。
func searchTasks(a *app.Application, kw string, scope user.DataScope) []searchTask {
	return []searchTask{
		{"customer", "menu:customer", func(ctx context.Context) (any, error) {
			return a.Customer.List(ctx, customer.CustomerQuery{
				NameKeyword: kw, LegalEntityID: scope.LegalEntityID, RegionScope: scope.RegionScope, Limit: searchMaxHits,
			})
		}},
		{"user", "menu:user", func(ctx context.Context) (any, error) {
			list, err := a.UserData.ListUsers(ctx, kw)
			return truncateSlice(list, searchMaxHits), err
		}},
		{"worker", "menu:dispatch", func(ctx context.Context) (any, error) {
			list, err := a.Worker.ListWorkers(ctx, 0, kw)
			return truncateSlice(list, searchMaxHits), err
		}},
		{"order", "menu:order", func(ctx context.Context) (any, error) {
			return a.Order.List(ctx, order.OrderQuery{
				Keyword: kw, LegalEntityID: scope.LegalEntityID, RegionScope: scope.RegionScope, Limit: searchMaxHits,
			})
		}},
	}
}

// runSearchTasks 并发执行有权限的检索任务;无权限/失败/空结果的域不出组,顺序按声明保持。
func runSearchTasks(ctx context.Context, a *app.Application, accountID int64, tasks []searchTask) []gin.H {
	groups := make([]gin.H, len(tasks))
	var wg sync.WaitGroup
	for i, t := range tasks {
		wg.Add(1)
		go func(i int, t searchTask) {
			defer wg.Done()
			ok, err := a.User.HasPermission(ctx, accountID, t.perm)
			if err != nil || !ok {
				return // 权限判定失败与无权限同视:该域不检索
			}
			items, err := t.fetch(ctx)
			if err != nil || !nonEmpty(items) {
				return // 单域失败/无命中静默跳过,不拖垮整体搜索
			}
			groups[i] = gin.H{"domain": t.domain, "items": items}
		}(i, t)
	}
	wg.Wait()
	out := make([]gin.H, 0, len(groups))
	for _, g := range groups {
		if g != nil {
			out = append(out, g)
		}
	}
	return out
}

// truncateSlice 截断任意切片至前 n 个元素(用户/师傅域查询无 Limit 参数,handler 侧收敛)。
func truncateSlice(s any, n int) any {
	v := reflect.ValueOf(s)
	if v.Kind() != reflect.Slice {
		return s
	}
	if v.Len() > n {
		v = v.Slice(0, n)
	}
	return v.Interface()
}

// nonEmpty 判断任意切片是否非空。
func nonEmpty(s any) bool {
	v := reflect.ValueOf(s)
	return v.Kind() == reflect.Slice && v.Len() > 0
}
