// Package apikey 提供免登录 API key 管理:key 与账号绑定,权限随账号角色。
//
// 安全原则:
//   - 只存 sha256(key) 哈希,永不落明文;
//   - 创建成功时返回完整密钥(仅一次),后续查询只返回元数据;
//   - 停用账号即停用其所有 key(ON DELETE CASCADE + status 过滤)。
package apikey

import "context"

// APIKey 元数据(不含明文密钥)。
type APIKey struct {
	ID          int64  `json:"id"`
	AccountID   int64  `json:"accountId"`
	AccountName string `json:"accountName,omitempty"` // 冗余展示,来自关联查询
	Name        string `json:"name"`
	KeyPrefix   string `json:"keyPrefix"`   // 密钥前 8 位(用于识别)
	Status      int16  `json:"status"`      // 1启用 0停用
	LastUsedAt  string `json:"lastUsedAt"`  // ISO8601,空=从未使用
	ExpiresAt   string `json:"expiresAt"`   // ISO8601,空=永不过期
	CreatedAt   string `json:"createdAt"`
}

// CreateResult 创建成功返回(含完整密钥,仅在此时返回一次)。
type CreateResult struct {
	APIKey
	PlainKey string `json:"plainKey"` // 完整密钥,如 boss_xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx
}

// Service 免登录 API key 管理接口。
type Service interface {
	// Create 为指定账号创建 API key。返回 CreateResult 包含完整密钥。
	// name 为用途说明,如 "ci-pipeline-01"。
	Create(ctx context.Context, accountID, createdBy int64, name string) (*CreateResult, error)

	// List 列出所有 API key 元数据(不含明文密钥)。
	List(ctx context.Context) ([]APIKey, error)

	// Revoke 停用指定 API key(软删除,status=0)。
	Revoke(ctx context.Context, id int64) error

	// Lookup 通过密钥哈希查找绑定的账号ID;status=0 的 key 返回 ErrNotFound。
	// 用于认证中间件。
	Lookup(ctx context.Context, keyHash string) (accountID int64, err error)

	// Touch 更新 last_used_at(认证成功时调用,尽力而为)。
	Touch(ctx context.Context, keyHash string)
}