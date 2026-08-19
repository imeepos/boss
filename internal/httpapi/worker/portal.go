package workerapi

// W 师傅端门户(/api/worker/v1)唯一注册入口 + 师傅 JWT(workerJWT)签发/校验。
// 决策:admin 的 auth.Manager.Claims(aid/usr/role)无 subject 维度,师傅 token 若复用
// 会被 admin Authn 当作账号 token 接受(横向越权);故本文件自实现 workerJWT,
// 复用同一密钥配置(config.JWT),issuer 固定 boss-worker 与 admin(boss)互不可冒充。

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"github.com/ymm-001/boss/internal/app"
	"github.com/ymm-001/boss/internal/pkg/auth"
	"github.com/ymm-001/boss/internal/pkg/config"
	"github.com/ymm-001/boss/pkg/apitypes"
)

const (
	ctxPortalWorkerID   = "portal.workerId"
	ctxPortalWorkerName = "portal.workerName"
	workerIssuer        = "boss-worker"
)

// workerClaims 师傅 token 载荷:subject=worker,只带师傅身份,不带 RBAC。
type workerClaims struct {
	WorkerID   int64  `json:"wid"`
	WorkerName string `json:"wname"`
	jwt.RegisteredClaims
}

// newWorkerJWTManager 复用 admin 同一密钥配置(BOSS_JWT_SECRET/TTL),单事实源。
func newWorkerJWTManager() *auth.Manager {
	cfg := config.Load()
	return auth.NewManager(cfg.JWT.Secret, cfg.JWT.TTL)
}

// signWorkerToken 签发师傅 JWT(issuer=boss-worker,subject=worker)。
func signWorkerToken(workerID int64, workerName string) (string, error) {
	now := time.Now()
	cfg := config.Load()
	c := workerClaims{
		WorkerID: workerID, WorkerName: workerName,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(cfg.JWT.TTL)),
			Subject:   "worker", Issuer: workerIssuer,
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString([]byte(cfg.JWT.Secret))
}

// verifyWorkerToken 校验师傅 JWT;非 boss-worker 签发一律拒绝。
func verifyWorkerToken(tokenStr string) (*workerClaims, error) {
	cfg := config.Load()
	t, err := jwt.ParseWithClaims(tokenStr, &workerClaims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, auth.ErrInvalidToken
		}
		return []byte(cfg.JWT.Secret), nil
	})
	if err != nil || !t.Valid {
		return nil, auth.ErrInvalidToken
	}
	c, ok := t.Claims.(*workerClaims)
	if !ok || c.Issuer != workerIssuer || c.Subject != "worker" {
		return nil, auth.ErrInvalidToken
	}
	return c, nil
}

// workerAuth 师傅端鉴权中间件:注入 ctxPortalWorkerID/Name。
func workerAuth() gin.HandlerFunc {
	return func(c *gin.Context) {
		h := c.GetHeader("Authorization")
		if !strings.HasPrefix(h, "Bearer ") {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "missing bearer token"})
			return
		}
		claims, err := verifyWorkerToken(strings.TrimPrefix(h, "Bearer "))
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"code": 401, "msg": "invalid worker token"})
			return
		}
		c.Set(ctxPortalWorkerID, claims.WorkerID)
		c.Set(ctxPortalWorkerName, claims.WorkerName)
		c.Next()
	}
}

// portalWorker 取当前师傅(id/name);缺省返回 0。
func portalWorker(c *gin.Context) (int64, string) {
	id, _ := c.Get(ctxPortalWorkerID)
	name, _ := c.Get(ctxPortalWorkerName)
	i, _ := id.(int64)
	n, _ := name.(string)
	return i, n
}

// registerWorkerPortalRoutes 师傅端门户唯一注册入口(接线由父会话调用)。
// mgr 参数保留给未来统一 token 服务;当前 workerJWT 独立签发(见文件头注释)。
func Register(r *gin.Engine, a *app.Application, mgr *auth.Manager) {
	_ = mgr
	pub := r.Group("/api/worker/v1")
	registerWorkerSelfRegistration(pub, a)
	registerWorkerPortalAuth(pub, a)
	wauth := r.Group("/api/worker/v1", workerAuth())
	registerWorkerPortalTicketRoutes(wauth, a)
	registerWorkerPortalScanRoutes(wauth, a)
	registerWorkerPortalAssetRoutes(wauth, a)
	registerWorkerPortalProfileRoutes(wauth, a)
	registerWorkerPortalMiscRoutes(wauth, a)
}

// registerWorkerPortalAuth 公开组:验证码/登录/退出(worker/auth.yaml)。
func registerWorkerPortalAuth(g *gin.RouterGroup, a *app.Application) {
	g.POST("/auth/sms-code", workerSmsCodeHandler(a))
	g.POST("/auth/login", workerLoginHandler(a))
	g.POST("/auth/logout", func(c *gin.Context) {
		respond(c, apitypes.CodeOK, gin.H{"ok": true})
	})
}
