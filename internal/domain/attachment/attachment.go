// Package attachment 附件域:三端登录后上传文件到 MinIO 并登记上传者身份。
// uploader_type 与 apikey 主体模型对齐(account/worker/customer),便于审计追溯。
package attachment

import (
	"context"
	"errors"
)

// 上传者类型枚举(与 apikeys.subject_type 同域)。
const (
	UploaderAccount  = "account"
	UploaderWorker   = "worker"
	UploaderCustomer = "customer"
)

// ErrInvalidUploader 上传者类型非法。
var ErrInvalidUploader = errors.New("attachment: invalid uploader type")

// Attachment 附件元数据(对象实体在 MinIO,DB 只存登记)。
type Attachment struct {
	ID           int64  `json:"id"`
	ObjectKey    string `json:"objectKey"`
	FileName     string `json:"fileName"`
	ContentType  string `json:"contentType"`
	SizeBytes    int64  `json:"sizeBytes"`
	UploaderType string `json:"uploaderType"` // account | worker | customer
	UploaderID   int64  `json:"uploaderId"`
	CreatedAt    string `json:"createdAt"`
}

// Store 附件登记存取接口。
type Store interface {
	// Create 登记附件元数据,回填自增 id/createdAt。
	Create(ctx context.Context, at *Attachment) (*Attachment, error)

	// Get 按 id 查附件;不存在返回 ErrNotFound。
	Get(ctx context.Context, id int64) (*Attachment, error)

	// ListByUploader 列出上传者的附件(新→旧,不含已删)。
	ListByUploader(ctx context.Context, uploaderType string, uploaderID int64, limit int) ([]Attachment, error)

	// List 按筛选查附件(新→旧,不含已删),返回当页 items 与命中总数。
	List(ctx context.Context, f ListFilter) ([]Attachment, int, error)

	// Delete 软删除(置 deleted_at);不存在或已删返回 ErrNotFound。
	Delete(ctx context.Context, id int64) error

	// GetByIDs 按 id 批量查(不含已删)。
	GetByIDs(ctx context.Context, ids []int64) ([]Attachment, error)
}

// ListFilter 附件列表筛选:各条件可选,零值 = 不过滤。
type ListFilter struct {
	UploaderType string // 空查全部
	UploaderID   int64  // 0 查全部
	Keyword      string // 文件名 ILIKE 模糊匹配
	Limit        int
	Offset       int
}

// ValidUploaderType 校验上传者类型合法。
func ValidUploaderType(t string) bool {
	return t == UploaderAccount || t == UploaderWorker || t == UploaderCustomer
}
