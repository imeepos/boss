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

	// Open 按 object key 打开对象读流(调用方负责 Close)。
	Open(ctx context.Context, cfg MinIOConfig, key string) (io.ReadCloser, error)
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

// Open 按 object key 打开对象读流。
func (s *MinIOStorage) Open(ctx context.Context, cfg MinIOConfig, key string) (io.ReadCloser, error) {
	cli, err := newClient(cfg)
	if err != nil {
		return nil, fmt.Errorf("attachment: minio client: %w", err)
	}
	obj, err := cli.GetObject(ctx, cfg.Bucket, key, minio.GetObjectOptions{})
	if err != nil {
		return nil, fmt.Errorf("attachment: get object: %w", err)
	}
	return obj, nil
}

// Service 附件上传服务:对象入 MinIO + 元数据入 PG。
type Service struct {
	St      Store
	Obj     ObjectStorage
	Conf    MinIOConfig
	Resolve func(context.Context) (MinIOConfig, error)
}

// Upload 上传并登记;at 需已填 UploaderType/UploaderID。
// UploaderID 仅拒绝 0:合成客户为负数段 ID(隔离空间,portal.syntheticID),同样是合法上传者。
func (s *Service) Upload(ctx context.Context, at *Attachment, reader io.Reader, size int64) (*Attachment, error) {
	if !ValidUploaderType(at.UploaderType) || at.UploaderID == 0 {
		return nil, ErrInvalidUploader
	}
	cfg := s.Conf
	if s.Resolve != nil {
		resolved, err := s.Resolve(ctx)
		if err != nil {
			return nil, err
		}
		cfg = resolved
	}
	key, err := s.Obj.Put(ctx, cfg, reader, size, at.ContentType, at.FileName)
	if err != nil {
		return nil, err
	}
	at.ObjectKey = key
	at.SizeBytes = size
	return s.St.Create(ctx, at)
}

// Download 按 id 取附件元数据与对象读流(调用方负责 Close)。
func (s *Service) Download(ctx context.Context, id int64) (*Attachment, io.ReadCloser, error) {
	at, err := s.St.Get(ctx, id)
	if err != nil {
		return nil, nil, err
	}
	cfg := s.Conf
	if s.Resolve != nil {
		resolved, err := s.Resolve(ctx)
		if err != nil {
			return nil, nil, err
		}
		cfg = resolved
	}
	r, err := s.Obj.Open(ctx, cfg, at.ObjectKey)
	if err != nil {
		return nil, nil, err
	}
	return at, r, nil
}
