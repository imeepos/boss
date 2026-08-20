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

	// ListByUploader 列出上传者的附件(新→旧)。
	ListByUploader(ctx context.Context, uploaderType string, uploaderID int64, limit int) ([]Attachment, error)
}

// ValidUploaderType 校验上传者类型合法。
func ValidUploaderType(t string) bool {
	return t == UploaderAccount || t == UploaderWorker || t == UploaderCustomer
}
