package attachment

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"path/filepath"
	"strings"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// MinIOConfig MinIO 连接配置(来自 config.Load 的 MinIO 段)。
type MinIOConfig struct {
	Endpoint  string // host:port,如 192.168.0.102:29000
	AccessKey string
	SecretKey string
	Bucket    string
	UseSSL    bool
}

// ObjectStorage 对象存储抽象,便于测试替身。
type ObjectStorage interface {
	// Put 上传对象并返回 object key。
	Put(ctx context.Context, cfg MinIOConfig, reader io.Reader, size int64, contentType, origName string) (string, error)
}

// MinIOStorage 基于 minio-go 的实现。
type MinIOStorage struct{}

// NewMinIOStorage 构造 MinIO 存储客户端(无状态,client 按需创建)。
func NewMinIOStorage() *MinIOStorage { return &MinIOStorage{} }

func newClient(cfg MinIOConfig) (*minio.Client, error) {
	return minio.New(cfg.Endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(cfg.AccessKey, cfg.SecretKey, ""),
		Secure: cfg.UseSSL,
	})
}

// ensureBucket 桶不存在则创建(幂等)。
func ensureBucket(ctx context.Context, cli *minio.Client, bucket string) error {
	ok, err := cli.BucketExists(ctx, bucket)
	if err != nil {
		return fmt.Errorf("attachment: bucket exists: %w", err)
	}
	if ok {
		return nil
	}
	if err := cli.MakeBucket(ctx, bucket, minio.MakeBucketOptions{}); err != nil {
		// 并发创建撞已存在视为成功。
		if exists, chk := cli.BucketExists(ctx, bucket); chk == nil && exists {
			return nil
		}
		return fmt.Errorf("attachment: make bucket: %w", err)
	}
	return nil
}

// newObjectKey 生成不可猜测的对象键:bucket 根按日期分目录 + 随机前缀保底唯一。
func newObjectKey(origName string) string {
	ext := strings.ToLower(filepath.Ext(origName))
	if len(ext) > 16 {
		ext = ""
	}
	b := make([]byte, 12)
	_, _ = rand.Read(b)
	return time.Now().UTC().Format("20060102") + "/" + hex.EncodeToString(b) + ext
}

// Put 上传对象,返回 object key。
func (s *MinIOStorage) Put(ctx context.Context, cfg MinIOConfig, reader io.Reader, size int64, contentType, origName string) (string, error) {
	cli, err := newClient(cfg)
	if err != nil {
		return "", fmt.Errorf("attachment: minio client: %w", err)
	}
	if err := ensureBucket(ctx, cli, cfg.Bucket); err != nil {
		return "", err
	}
	if contentType == "" {
		contentType = "application/octet-stream"
	}
	key := newObjectKey(origName)
	if _, err := cli.PutObject(ctx, cfg.Bucket, key, reader, size,
		minio.PutObjectOptions{ContentType: contentType}); err != nil {
		return "", fmt.Errorf("attachment: put object: %w", err)
	}
	return key, nil
}

// Service 附件上传服务:对象入 MinIO + 元数据入 PG。
type Service struct {
	St   Store
	Obj  ObjectStorage
	Conf MinIOConfig
}

// Upload 上传并登记;at 需已填 UploaderType/UploaderID。
func (s *Service) Upload(ctx context.Context, at *Attachment, reader io.Reader, size int64) (*Attachment, error) {
	if !ValidUploaderType(at.UploaderType) || at.UploaderID <= 0 {
		return nil, ErrInvalidUploader
	}
	key, err := s.Obj.Put(ctx, s.Conf, reader, size, at.ContentType, at.FileName)
	if err != nil {
		return nil, err
	}
	at.ObjectKey = key
	at.SizeBytes = size
	return s.St.Create(ctx, at)
}
