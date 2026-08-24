package workerapi

// 师傅端客户端版本检查(免登录,启动即查;契约 fields.md 8F)。
// 白名单豁免尽力而为:带有效师傅 token 时取 workerID 参与 whitelist 命中。

import (
	"io"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/apprelease"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerWorkerClientReleaseRoutes 版本检查 pub 路由:latest 判定 + 灰度门控下载。
func registerWorkerClientReleaseRoutes(pub *gin.RouterGroup, a *app.Application) {
	pub.GET("/client/latest", workerClientLatest(a))
	pub.GET("/client/apk/:id", workerClientAPK(a))
}

// workerSubjectBestEffort 可选解析师傅 token(失败/未带返回 0,不阻断匿名检查)。
func workerSubjectBestEffort(c *gin.Context) int64 {
	h := c.GetHeader("Authorization")
	if !strings.HasPrefix(h, "Bearer ") {
		return 0
	}
	claims, err := verifyWorkerToken(strings.TrimPrefix(h, "Bearer "))
	if err != nil {
		return 0
	}
	return claims.WorkerID
}

// workerClientLatest 升级判定:query versionCode/deviceId 必填。
// 响应附 downloadUrl,指向同域灰度门控下载端点。
func workerClientLatest(a *app.Application) gin.HandlerFunc {
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
		res, err := a.AppRelease.Check(c.Request.Context(), apprelease.AppWorker,
			vc, deviceID, workerSubjectBestEffort(c))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{
			"updateAvailable": res.UpdateAvailable, "force": res.Force,
			"version": res.Version, "versionCode": res.VersionCode,
			"notes": res.Notes, "sha256": res.Sha256, "size": res.Size,
			"downloadUrl": "/api/worker/v1/client/apk/" + strconv.FormatInt(res.ReleaseID, 10),
		})
	}
}

// workerClientAPK 灰度门控下载:以 versionCode=0 重放判定,仅当目标就是 :id 时放行,
// 防止灰度包被未命中设备直链下载;PUBLISHED 包对所有人可见。
func workerClientAPK(a *app.Application) gin.HandlerFunc {
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
		res, err := a.AppRelease.Check(c.Request.Context(), apprelease.AppWorker,
			0, deviceID, workerSubjectBestEffort(c))
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
		streamClientAPK(c, a, r)
	}
}

// streamClientAPK APK 回传流(两客户端端点共用形态)。
func streamClientAPK(c *gin.Context, a *app.Application, r *apprelease.Release) {
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
