package middleware

import (
	"context"
	"net/http"
	"reflect"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/domain/license"
)

// LicenseChecker 门禁依赖的最小接口(便于测试注入;真实现为 license.Service)。
type LicenseChecker interface {
	Check(ctx context.Context) (license.Status, error)
}

// LicenseGate 系统级授权门禁:未激活/证书失效时拦截业务接口。
//
// 设计(adopted license-gate note):
//   - svc 为 nil = 未启用(开发/演示完全放行);
//   - 豁免前缀(exempt):登录/注册/版检/授权页自身,避免"没证书进不去、进不去没法激活";
//   - 拦截后返回 403 LICENSE_REQUIRED,前端引导到授权激活页。
func LicenseGate(svc LicenseChecker, exempt ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if isNilChecker(svc) {
			c.Next()
			return
		}
		path := c.Request.URL.Path
		for _, p := range exempt {
			if strings.HasPrefix(path, p) {
				c.Next()
				return
			}
		}
		if _, err := svc.Check(c.Request.Context()); err != nil {
			// 失败留可 grep 日志:授权失败不可静默放行。
			c.AbortWithStatusJSON(http.StatusForbidden, gin.H{
				"code": "LICENSE_REQUIRED",
				"msg":  "system license required",
			})
			return
		}
		c.Next()
	}
}

// isNilChecker 判接口是否持有空值(含 nil 指针装箱为接口的场景;
// 否则 nil *license.Service 赋给接口后 != nil,门禁误触发 panic)。
func isNilChecker(svc LicenseChecker) bool {
	if svc == nil {
		return true
	}
	v := reflect.ValueOf(svc)
	switch v.Kind() {
	case reflect.Ptr, reflect.Interface, reflect.Map, reflect.Slice, reflect.Func, reflect.Chan:
		return v.IsNil()
	}
	return false
}
