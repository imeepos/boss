package auth

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

var ErrInvalidToken = errors.New("invalid token")

// Claims JWT 载荷:账号ID + 角色,权限校验走 RBAC 快照(不塞进 token,保证权限变更即时生效)。
// KeyID 预留未来多密钥轮换(见 docs/architecture-review.md 发现 2.2);本轮单密钥时为空。
type Claims struct {
	AccountID int64  `json:"aid"`
	Username  string `json:"usr"`
	RoleCode  string `json:"role"`
	KeyID     string `json:"kid,omitempty"`
	jwt.RegisteredClaims
}

type Manager struct {
	secret []byte
	ttl    time.Duration
}

func NewManager(secret string, ttl time.Duration) *Manager {
	return &Manager{secret: []byte(secret), ttl: ttl}
}

func (m *Manager) Sign(accountID int64, username, roleCode string) (string, error) {
	now := time.Now()
	c := Claims{
		AccountID: accountID,
		Username:  username,
		RoleCode:  roleCode,
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
	c, ok := t.Claims.(*Claims)
	if !ok {
		return nil, ErrInvalidToken
	}
	// jwt v5 已在校验时运行默认 validator(校验 exp/nbf/iat),t.Valid 为结果;
	// 这里再显式校验 issuer,不放过非本系统签发的 token(发现 2.2)。
	if !t.Valid {
		return nil, ErrInvalidToken
	}
	if c.Issuer != "" && c.Issuer != "boss" {
		return nil, ErrInvalidToken
	}
	return c, nil
}
