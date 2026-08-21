package adminapi

import (
	"fmt"
	"strings"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/httpx"
	"github.com/ymm-001/boss/internal/pkg/secretbox"
	"github.com/ymm-001/boss/pkg/apitypes"
)

var storageKeys = map[string]bool{
	"minio.endpoint": true, "minio.accessKey": true, "minio.secretKey": true,
	"minio.bucket": true, "minio.useSSL": true,
}

func adminStorageConfigGet(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		list, err := a.User.ListParams(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		out := gin.H{}
		for _, p := range list {
			if !storageKeys[p.Key] {
				continue
			}
			if p.Key == "minio.secretKey" {
				out[p.Key] = gin.H{"value": "", "hasValue": p.Value != ""}
			} else {
				out[p.Key] = gin.H{"value": p.Value, "hasValue": p.Value != ""}
			}
		}
		respond(c, apitypes.CodeOK, gin.H{"fields": out})
	}
}

func adminStorageConfigPut(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Values map[string]string `json:"values" binding:"required"`
		}
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		id := httpx.ClaimsAccountID(c)
		for key, value := range req.Values {
			if !storageKeys[key] {
				respond(c, apitypes.CodeInvalidParam, nil)
				return
			}
			if key == "minio.secretKey" {
				if value == "" {
					continue
				}
				enc, err := secretbox.Seal(value)
				if err != nil {
					respond(c, apitypes.CodeInternal, nil)
					return
				}
				value = enc
			}
			if err := a.User.UpdateParam(c.Request.Context(), key, value, id); err != nil {
				respondErr(c, err)
				return
			}
		}
		httpx.RecordAudit(a, c, "数据变更", "storage_config", "minio", gin.H{"keys": len(req.Values)})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// adminStorageConfigRotateSecret 轮换 MinIO 根密码。
//
// 流程:校验密码 → 更新 biz_params(secretKey,加密存储) → 调 hostctl 写宿主机密码文件 + 重启 minio。
func adminStorageConfigRotateSecret(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		var req struct {
			Secret string `json:"secret" binding:"required"`
		}
		if !httpx.BindAndValidate(c, &req) {
			return
		}
		secret := strings.TrimSpace(req.Secret)
		if n := len(secret); n < 8 || n > 128 {
			respond(c, apitypes.CodeInvalidParam, nil)
			return
		}
		if a.HostCtl == nil {
			respond(c, apitypes.CodeInternal, nil)
			return
		}

		// 先更新 biz_params(加密存储)
		enc, err := secretbox.Seal(secret)
		if err != nil {
			respond(c, apitypes.CodeInternal, nil)
			return
		}
		uid := httpx.ClaimsAccountID(c)
		if err := a.User.UpdateParam(c.Request.Context(), "minio.secretKey", enc, uid); err != nil {
			respondErr(c, err)
			return
		}

		// 调 hostctl 写宿主机密码文件 + 重启 minio
		r, err := a.HostCtl.RotateSecret(secret)
		if err != nil {
			respond(c, apitypes.CodeInternal, gin.H{"error": fmt.Sprintf("hostctl: %v", err)})
			return
		}

		httpx.RecordAudit(a, c, "密钥轮换", "storage_config", "minio", gin.H{"rotatedAt": r.RotatedAt})
		respond(c, apitypes.CodeOK, gin.H{"ok": true, "rotatedAt": r.RotatedAt, "duration": r.Duration})
	}
}
