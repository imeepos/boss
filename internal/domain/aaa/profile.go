package aaa

import "context"

// Profile 认证档案(LOID 多模式认证凭据)。
// 权威源:docs/contract/domain-map.md AAA 域;授权决策依据,与 DB 绑定账号一一对应。
type Profile struct {
	LOID       string // 逻辑唯一标识,认证主键(optical line identifier)
	Username   string // RADIUS User-Name
	Password   string // PAP 明文;CHAP 时由 NAS 侧校验
	Status     Status // ACTIVE/SUSPENDED,停复机即时生效依据
	Bandwidth  string // 带宽模板(上行/下行),授权属性下发
	SessionTTL int    // 授权缓存 TTL 秒(默认 60,见技术栈 3.4)
}

// Status 认证账号状态。
type Status string

const (
	StatusActive    Status = "ACTIVE"
	StatusSuspended Status = "SUSPENDED"
)

// ProfileRepo 认证档案查询口(只读,供 RADIUS 认证与授权决策使用)。
type ProfileRepo interface {
	// GetByLOID 按 LOID 查认证档案;未找到返回 ErrNotFound。
	GetByLOID(ctx context.Context, loid string) (*Profile, error)
	// GetByUsername 按 RADIUS User-Name 反查(多模式认证入口)。
	GetByUsername(ctx context.Context, username string) (*Profile, error)
}
