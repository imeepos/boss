package adminapi

import (
	"io"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/apprelease"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/pkg/apitypes"
)

// registerClientReleaseRoutes 版本发布管理(menu:release 权限)。
func registerClientReleaseRoutes(g *gin.RouterGroup, a *app.Application) {
	p := requirePerm(a.User, "menu:release")
	g.GET("/client-releases", p, clientReleaseList(a))
	g.POST("/client-releases", p, clientReleaseCreate(a))
	g.GET("/client-releases/:id", p, clientReleaseGet(a))
	g.PATCH("/client-releases/:id", p, clientReleaseUpdate(a))
	g.GET("/client-releases/:id/apk", p, clientReleaseDownload(a))
}

// registerClientReleasePublicRoutes 官网匿名下载入口(仅最新 PUBLISHED):
// /site/downloads 元数据投影(两端一起回,首页一次拉全)+ /site/downloads/apk 包流。
// 不挂 /client-releases/latest:gin 同段 param(:id) 与 static 冲突。
func registerClientReleasePublicRoutes(api *gin.RouterGroup, a *app.Application) {
	api.GET("/site/downloads", siteDownloadsHandler(a))
	api.GET("/site/downloads/apk", siteDownloadAPKHandler(a))
}

func clientReleaseList(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.AppRelease == nil {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		items, err := a.AppRelease.St.List(c.Request.Context(), c.Query("app"))
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, gin.H{"items": items})
	}
}

// clientReleaseCreate multipart 上传:file(APK) + 表单字段 app/version/versionCode/
// minSupportedCode/notes/force/status/rolloutPercent/whitelistIds(逗号分隔)。
func clientReleaseCreate(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.AppRelease == nil {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		fh, err := c.FormFile("file")
		if err != nil {
			respond(c, apitypes.CodeInvalidParam, gin.H{"field": "file"})
			return
		}
		f, err := fh.Open()
		if err != nil {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		defer f.Close()
		r := &apprelease.Release{
			App: c.PostForm("app"), Version: strings.TrimSpace(c.PostForm("version")),
			Notes: c.PostForm("notes"), Status: c.PostForm("status"),
		}
		if r.Status == "" {
			r.Status = apprelease.StatusDraft
		}
		if r.VersionCode, _ = strconv.Atoi(c.PostForm("versionCode")); r.VersionCode == 0 {
			respond(c, apitypes.CodeInvalidParam, gin.H{"field": "versionCode"})
			return
		}
		if r.MinSupportedCode, _ = strconv.Atoi(c.PostForm("minSupportedCode")); r.MinSupportedCode < 1 {
			r.MinSupportedCode = 1 // 缺省=1:任何存量客户端都不被强升(需求默认不强制)
		}
		r.RolloutPercent, _ = strconv.Atoi(c.PostForm("rolloutPercent"))
		r.Force = c.PostForm("force") == "true"
		r.WhitelistIDs = parseIDList(c.PostForm("whitelistIds"))
		if err := a.AppRelease.Upload(c.Request.Context(), r, f, fh.Size, fh.Filename); err != nil {
			respondErr(c, err)
			return
		}
		if err := a.AppRelease.St.Create(c.Request.Context(), r); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "client-release.create", "client_release",
			strconv.FormatInt(r.ID, 10), nil)
		respond(c, apitypes.CodeOK, r)
	}
}

// clientReleaseUpdate 元数据/状态迁移(全量 PATCH;不含 APK 重传)。
func clientReleaseUpdate(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.AppRelease == nil {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		r, err := a.AppRelease.St.Get(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		var body releasePatchBody
		if !httpx.BindAndValidate(c, &body) {
			return
		}
		applyReleasePatch(r, &body)
		if err := a.AppRelease.St.Update(c.Request.Context(), r); err != nil {
			respondErr(c, err)
			return
		}
		httpx.RecordAudit(a, c, "client-release.update", "client_release",
			strconv.FormatInt(id, 10), nil)
		respond(c, apitypes.CodeOK, r)
	}
}

func clientReleaseGet(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.AppRelease == nil {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		r, err := a.AppRelease.St.Get(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		respond(c, apitypes.CodeOK, r)
	}
}

// clientReleaseDownload 管理端下载(任意状态,便于回滚取旧包)。
func clientReleaseDownload(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.AppRelease == nil {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		id, _ := strconv.ParseInt(c.Param("id"), 10, 64)
		r, err := a.AppRelease.St.Get(c.Request.Context(), id)
		if err != nil {
			respondErr(c, err)
			return
		}
		streamAPK(c, a, r)
	}
}

// siteDownloadsHandler 首页下载卡片数据:两端最新 PUBLISHED 各一行,无发版给空段。
func siteDownloadsHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.AppRelease == nil {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		ctx := c.Request.Context()
		out := make([]gin.H, 0, 2)
		for _, app := range []string{apprelease.AppUser, apprelease.AppWorker} {
			r, err := a.AppRelease.LatestPublished(ctx, app)
			if err != nil {
				continue // 该端暂无全量版,首页隐藏对应卡片
			}
			out = append(out, gin.H{
				"app": r.App, "version": r.Version, "sha256": r.Sha256, "size": r.ApkSize,
				"downloadPath": "/api/admin/v1/site/downloads/apk?app=" + r.App,
			})
		}
		respond(c, apitypes.CodeOK, gin.H{"items": out})
	}
}

// siteDownloadAPKHandler 官网匿名包流(不泄露灰度/回滚包,仅最新 PUBLISHED)。
func siteDownloadAPKHandler(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.AppRelease == nil {
			respond(c, apitypes.CodeNotFound, nil)
			return
		}
		app := c.Query("app")
		if app != apprelease.AppUser && app != apprelease.AppWorker {
			respond(c, apitypes.CodeInvalidParam, gin.H{"field": "app"})
			return
		}
		r, err := a.AppRelease.LatestPublished(c.Request.Context(), app)
		if err != nil {
			respondErr(c, err)
			return
		}
		streamAPK(c, a, r)
	}
}

// streamAPK 公共回传流(admin 下载与公开 latest 复用):Content-Disposition 带文件名。
func streamAPK(c *gin.Context, a *app.Application, r *apprelease.Release) {
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

// parseIDList 逗号分隔 id 串 → int64 切片;空白返回 nil。
func parseIDList(s string) []int64 {
	var out []int64
	for _, part := range strings.Split(s, ",") {
		if v, err := strconv.ParseInt(strings.TrimSpace(part), 10, 64); err == nil && v > 0 {
			out = append(out, v)
		}
	}
	return out
}
