package adminapi

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/domain/attachment"
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
		if !storageApplyValues(c, a, req.Values) {
			return
		}
		httpx.RecordAudit(a, c, "数据变更", "storage_config", "minio", gin.H{"keys": len(req.Values)})
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	}
}

// adminStorageConfigTest 存储连通性自检(data-relations §6.3):解析生效配置
// (biz_params 覆盖 env,secret 解密)后零副作用探测——endpoint 可达→凭据有效→
// bucket 存在;5s 超时,不创建不写入;结果 {ok,latencyMs,message} 与 auth 自检同形。
func adminStorageConfigTest(a *app.Application) gin.HandlerFunc {
	return func(c *gin.Context) {
		if a.Attachment == nil || a.Attachment.Resolve == nil {
			respond(c, apitypes.CodeInternal, nil)
			return
		}
		cfg, err := a.Attachment.Resolve(c.Request.Context())
		if err != nil {
			respondErr(c, err)
			return
		}
		ctx, cancel := context.WithTimeout(c.Request.Context(), 5*time.Second)
		defer cancel()
		start := time.Now()
		probeErr := attachment.Probe(ctx, cfg)
		latency := time.Since(start).Milliseconds()
		result := gin.H{"ok": probeErr == nil, "latencyMs": latency}
		if probeErr != nil {
			result["message"] = probeErr.Error()
		}
		// 审计留痕不带密钥;失败原因走响应 message(用户可复制)。
		httpx.RecordAudit(a, c, "连通性自检", "storage_config", "minio",
			gin.H{"ok": probeErr == nil, "latencyMs": latency})
		respond(c, apitypes.CodeOK, result)
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
		if !storagePersistSecret(c, a, secret) {
			return
		}
		r, err := a.HostCtl.RotateSecret(secret)
		if err != nil {
			respond(c, apitypes.CodeInternal, gin.H{"error": fmt.Sprintf("hostctl: %v", err)})
			return
		}
		httpx.RecordAudit(a, c, "密钥轮换", "storage_config", "minio", gin.H{"rotatedAt": r.RotatedAt})
		respond(c, apitypes.CodeOK, gin.H{"ok": true, "rotatedAt": r.RotatedAt, "duration": r.Duration})
	}
}

// storagePersistSecret 新密码加密落 biz_params;失败已回写响应。
func storagePersistSecret(c *gin.Context, a *app.Application, secret string) bool {
	enc, err := secretbox.Seal(secret)
	if err != nil {
		respond(c, apitypes.CodeInternal, nil)
		return false
	}
	uid := httpx.ClaimsAccountID(c)
	if err := a.User.UpdateParam(c.Request.Context(), "minio.secretKey", enc, uid); err != nil {
		respondErr(c, err)
		return false
	}
	return true
}

// storageApplyValues 逐键落库:白名单校验;secretKey 空值跳过、非空加密;
// 失败已回写响应。
func storageApplyValues(c *gin.Context, a *app.Application, values map[string]string) bool {
	id := httpx.ClaimsAccountID(c)
	for key, value := range values {
		if !storageKeys[key] {
			respond(c, apitypes.CodeInvalidParam, nil)
			return false
		}
		if key == "minio.secretKey" {
			if value == "" {
				continue
			}
			enc, err := secretbox.Seal(value)
			if err != nil {
				respond(c, apitypes.CodeInternal, nil)
				return false
			}
			value = enc
		}
		if err := a.User.UpdateParam(c.Request.Context(), key, value, id); err != nil {
			respondErr(c, err)
			return false
		}
	}
	return true
}
