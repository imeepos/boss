package adminapi

import (
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
