package userapi

// 用户端客户端版本检查(免登录,启动即查;契约 fields.md 8F)。
// 与 worker 端同构:latest 判定 + 灰度门控下载;白名单尽力解析客户 token。

import (
	"io"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/apprelease"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerUserClientReleaseRoutes 版本检查 pub 路由。
func registerUserClientReleaseRoutes(pub *gin.RouterGroup, a *app.Application, mgr *auth.Manager) {
	pub.GET("/client/latest", userClientLatest(a, mgr))
	pub.GET("/client/apk/:id", userClientAPK(a, mgr))
}

// customerSubjectBestEffort 可选解析客户 JWT(失败/未带返回 0,不阻断匿名检查)。
func customerSubjectBestEffort(c *gin.Context, mgr *auth.Manager) int64 {
	h := c.GetHeader("Authorization")
	if !strings.HasPrefix(h, "Bearer ") || mgr == nil {
		return 0
	}
	claims, err := mgr.Verify(strings.TrimPrefix(h, "Bearer "))
	if err != nil || claims.Aud != auth.AudUser {
		return 0
	}
	return customerIDFromToken(claims)
}

// userClientLatest 升级判定:query versionCode/deviceId 必填。
func userClientLatest(a *app.Application, mgr *auth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.AppRelease == nil {
			respond(c, apitypes.CodeOK, gin.H{"updateAvailable": false})
			return
		}
		vc, _ := strconv.Atoi(c.Query("versionCode"))
		deviceID := strings.TrimSpace(c.Query("deviceId"))
		if vc <= 0 || deviceID == "" {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		res, err := a.AppRelease.Check(c.Request.Context(), apprelease.AppUser,
			vc, deviceID, customerSubjectBestEffort(c, mgr))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"updateAvailable": res.UpdateAvailable, "force": res.Force,
			"version": res.Version, "versionCode": res.VersionCode,
			"notes": res.Notes, "sha256": res.Sha256, "size": res.Size,
			"downloadUrl": "/api/user/v1/client/apk/" + strconv.FormatInt(res.ReleaseID, 10),
		})
	}
}

// userClientAPK 灰度门控下载:versionCode=0 重放判定,目标一致才放行。
func userClientAPK(a *app.Application, mgr *auth.Manager) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.AppRelease == nil {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		deviceID := strings.TrimSpace(c.Query("deviceId"))
		if id <= 0 || deviceID == "" {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		res, err := a.AppRelease.Check(c.Request.Context(), apprelease.AppUser,
			0, deviceID, customerSubjectBestEffort(c, mgr))
		if err != nil {
			respondErr(c, err)
			return
		}
		if !res.UpdateAvailable || res.ReleaseID != id {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		r, err := a.AppRelease.St.Get(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		rc, err := a.AppRelease.OpenAPK(c.Request.Context(), r)
		if err != nil {
			respondErr(c, err)
			return
		}
		defer rc.Close()
		c.Header("Content-Disposition", `attachment; filename="boss-`+r.App+`-`+r.Version+`.apk"`)
		c.Header("Content-Type", "application/vnd.android.package-archive")
		if r.ApkSize > 0 {
			c.Header("Content-Length", strconv.FormatInt(r.ApkSize, 10))
		}
		c.Status(200)
		_, _ = io.Copy(c.Writer, rc)
	}
}
