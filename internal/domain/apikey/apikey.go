// Package apikey 提供免登录 API key 管理:key 与三类主体(account/worker/customer)绑定。
//
// 主体模型(与迁移 000043 三表登录边界、000044 主体扩展对齐):
//   - account  → accounts.id,注入完整 RBAC 身份,可访问全部管理接口
//   - worker   → workers.id,注入师傅身份,用于扫码/工单类接口;不持有菜单权限
//   - customer → customers.id,注入客户身份,用于下单/查询类接口;不持有菜单权限
//
// 安全原则:
//   - 只存 sha256(key) 哈希,永不落明文;
//   - 创建成功时返回完整密钥(仅一次),后续查询只返回元数据;
//   - 停用主体即停用其所有 key(级联 + status 过滤)。
package apikey

import "context"

// 主体类型枚举。
const (
	SubjectAccount  = "account"
	SubjectWorker   = "worker"
	SubjectCustomer = "customer"
)

// APIKey 元数据(不含明文密钥)。
type APIKey struct {
	ID           int64  `json:"id"`
	SubjectType  string `json:"subjectType"`            // account | worker | customer
	SubjectRef   int64  `json:"subjectRef"`             // 主体表主键
	SubjectName  string `json:"subjectName"`            // 冗余展示:账号名/师傅名/客户名
	Name         string `json:"name"`                   // 用途说明,如 ci-pipeline
	KeyPrefix    string `json:"keyPrefix"`              // 密钥前 8 位
	TemplateCode string `json:"templateCode,omitempty"` // 受限权限模板
	Status       int16  `json:"status"`                 // 1启用 0停用
	LastUsedAt   string `json:"lastUsedAt"`             // ISO8601,空=从未使用
	CreatedAt    string `json:"createdAt"`
}

// CreateResult 创建成功返回(含完整密钥,仅在此时返回一次)。
type CreateResult struct {
	APIKey
	PlainKey string `json:"plainKey"` // 完整密钥,如 boss_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
}

// PermissionTemplate 是可用于受限密钥的权限模板。
type PermissionTemplate struct {
	Code        string   `json:"code"`
	Name        string   `json:"name"`
	Description string   `json:"description"`
	Permissions []string `json:"permissions"`
}

// Subject Lookup 解析出的主体标识。
type Subject struct {
	Type         string // account | worker | customer
	Ref          int64  // 主体表主键
	TemplateCode string // 受限权限模板
}

// Service 免登录 API key 管理接口。
type Service interface {
	// Create 为指定主体创建 API key,返回完整密钥。
	// subjectType ∈ {account, worker, customer};subjectRef 为主体表主键。
	Create(ctx context.Context, subjectType string, subjectRef, createdBy int64, name, templateCode string) (*CreateResult, error)

	// List 列出所有 API key 元数据(不含明文密钥)。
	List(ctx context.Context) ([]APIKey, error)
	// ListTemplates 列出可用于签发受限密钥的权限模板。
	ListTemplates(ctx context.Context) ([]PermissionTemplate, error)

	// Revoke 停用指定 API key(软删除)。
	Revoke(ctx context.Context, id int64) error

	// Lookup 通过密钥哈希查找绑定主体;status=0 的 key 返回 ErrNotFound。
	Lookup(ctx context.Context, keyHash string) (*Subject, error)

	// Touch 更新 last_used_at(尽力而为)。
	Touch(ctx context.Context, keyHash string)
}

// ValidSubjectType 校验主体类型合法。
func ValidSubjectType(t string) bool {
	return t == SubjectAccount || t == SubjectWorker || t == SubjectCustomer
}
