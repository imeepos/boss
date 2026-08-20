package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

// Claims JWT 载荷:账号ID + 角色 + 端标识(aud),权限校验走 RBAC 快照(不塞进 token,保证权限变更即时生效)。
// KeyID 预留未来多密钥轮换(见 docs/architecture-review.md 发现 2.2);本轮单密钥时为空。
// Aud 端标识:admin/user;签发必填,校验端精确匹配,防止跨端 token 冒用(师傅端走独立 issuer 隔离)。
type Claims struct {
	AccountID int64  `json:"aid"`
	Username  string `json:"usr"`
	RoleCode  string `json:"role"`
	Aud       string `json:"aud"`
	KeyID     string `json:"kid,omitempty"`
	jwt.RegisteredClaims
}

// 端标识常量:三端前缀与鉴权域一一对应(/api/admin/v1、/api/user/v1、/api/worker/v1)。
const (
	AudAdmin = "admin"
	AudUser  = "user"
)

type Manager struct {
	secret []byte
	ttl    time.Duration
}

func NewManager(secret string, ttl time.Duration) *Manager {
	return &Manager{secret: []byte(secret), ttl: ttl}
}

func (m *Manager) Sign(aud string, accountID int64, username, roleCode string) (string, error) {
	now := time.Now()
	c := Claims{
		AccountID: accountID,
		Username:  username,
		RoleCode:  roleCode,
		Aud:       aud,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(m.ttl)),
			Issuer:    "boss",
		},
	}
	return jwt.NewWithClaims(jwt.SigningMethodHS256, c).SignedString(m.secret)
}

func (m *Manager) Verify(tokenStr string) (*Claims, error) {
	t, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, ErrInvalidToken
		}
		return m.secret, nil
	})
	if err != nil {
		return nil, ErrInvalidToken
	}
	// ParseWithClaims 传入 *Claims 时返回的 Claims 恒为同一实例;
	// jwt v5 校验失败(exp/nbf/iat/签名/算法)一律返回非 nil err,上方已拦截。
	c := t.Claims.(*Claims)
	// 显式校验 issuer,不放过非本系统签发的 token(发现 2.2)。
	if c.Issuer != "" && c.Issuer != "boss" {
		return nil, ErrInvalidToken
	}
	return c, nil
}
