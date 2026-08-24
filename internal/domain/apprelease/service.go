package apprelease

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"

	"github.com/ymm-001/boss/internal/domain/attachment"
)

// maxAPKBytes APK 上传上限 256MB(attachment 32MB 不够安装包,单独放大)。
const maxAPKBytes = 256 << 20

// Service 发版服务:元数据入 PG,APK 对象入 MinIO(复用 attachment 存储抽象)。
type Service struct {
	St      Store
	Obj     attachment.ObjectStorage
	Conf    attachment.MinIOConfig
	Resolve func(context.Context) (attachment.MinIOConfig, error)
}

func (s *Service) config(ctx context.Context) (attachment.MinIOConfig, error) {
	cfg := s.Conf
	if s.Resolve != nil {
		resolved, err := s.Resolve(ctx)
		if err != nil {
			return cfg, err
		}
		cfg = resolved
	}
	return cfg, nil
}

// Upload APK 上传:边传边算 sha256,对象落 MinIO 后回填 key/size/sha256。
// 调用方负责随后 St.Create 落元数据;失败时对象可能已写入,按 key 可追责清理。
func (s *Service) Upload(ctx context.Context, r *Release, reader io.Reader, size int64, origName string) error {
	if size <= 0 || size > maxAPKBytes {
		return fmt.Errorf("%w: apk size out of range", ErrInvalid)
	}
	cfg, err := s.config(ctx)
	if err != nil {
		return err
	}
	hasher := sha256.New()
	key, err := s.Obj.Put(ctx, cfg, io.TeeReader(reader, hasher), size,
		"application/vnd.android.package-archive", origName)
	if err != nil {
		return fmt.Errorf("apprelease: put apk: %w", err)
	}
	r.ApkObjectKey = key
	r.ApkSize = size
	r.Sha256 = hex.EncodeToString(hasher.Sum(nil))
	return nil
}

// OpenAPK 打开 APK 读流(下载端点流式回传,调用方负责 Close)。
func (s *Service) OpenAPK(ctx context.Context, r *Release) (io.ReadCloser, error) {
	if r.ApkObjectKey == "" {
		return nil, fmt.Errorf("%w: no apk object", ErrNotFound)
	}
	cfg, err := s.config(ctx)
	if err != nil {
		return nil, err
	}
	rc, err := s.Obj.Open(ctx, cfg, r.ApkObjectKey)
	if err != nil {
		return nil, fmt.Errorf("apprelease: open apk: %w", err)
	}
	return rc, nil
}

// Check 客户端检查升级:候选取 Store.Actives 后走 Decide 判定。
// deviceID 用于灰度分桶;subjectID(登录后)用于白名单豁免,匿名传 0。
func (s *Service) Check(ctx context.Context, app string, clientVersionCode int, deviceID string, subjectID int64) (CheckResult, error) {
	candidates, err := s.St.Actives(ctx, app, PlatformAndroid)
	if err != nil {
		return CheckResult{}, err
	}
	return Decide(clientVersionCode, deviceID, subjectID, candidates), nil
}

// LatestPublished 官网匿名下载入口:取 app 的最新 PUBLISHED(不掺灰度)。
func (s *Service) LatestPublished(ctx context.Context, app string) (*Release, error) {
	candidates, err := s.St.Actives(ctx, app, PlatformAndroid)
	if err != nil {
		return nil, err
	}
	for i := range candidates {
		if candidates[i].Status == StatusPublished {
			return &candidates[i], nil
		}
	}
	return nil, ErrNotFound
}
