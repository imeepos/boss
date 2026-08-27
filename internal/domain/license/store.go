package license

import (
	"context"
	"errors"
	"os"
	"path/filepath"
)

// Store 本地证书文件存取:一个 JSON 文件保存令牌原文。
// 证书路径由配置注入;写文件用临时文件+rename 保证原子。
type Store struct {
	Path string
}

// Load 读证书文件;文件不存在返回 ErrNoLicense(首次激活引导)。
func (s *Store) Load(_ context.Context) ([]byte, error) {
	data, err := os.ReadFile(s.Path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return nil, ErrNoLicense
		}
		return nil, err
	}
	if len(data) == 0 {
		return nil, ErrInvalidToken
	}
	return data, nil
}

// Save 原子写证书文件(临时文件 + rename,防半写)。
func (s *Store) Save(ctx context.Context, data []byte) error {
	if len(data) == 0 {
		return ErrInvalidToken
	}
	if err := os.MkdirAll(filepath.Dir(s.Path), 0o750); err != nil {
		return err
	}
	tmp := s.Path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o640); err != nil {
		return err
	}
	return os.Rename(tmp, s.Path)
}

// Clear 删除本地证书(管理上主动清除;文件不存在是幂等成功)。
func (s *Store) Clear(ctx context.Context) error {
	err := os.Remove(s.Path)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}