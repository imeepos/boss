package app

import (
	"context"

	"github.com/ymm-001/boss/internal/domain/attachment"
	"github.com/ymm-001/boss/internal/domain/user"
	"github.com/ymm-001/boss/internal/pkg/config"
)

func minioConfigResolver(svc user.Service, fallback attachment.MinIOConfig) func(context.Context) (attachment.MinIOConfig, error) {
	return func(ctx context.Context) (attachment.MinIOConfig, error) {
		cfg := fallback
		list, err := svc.ListParams(ctx)
		if err != nil {
			return cfg, nil
		}
		stored := make(map[string]string, len(list))
		for _, p := range list {
			stored[p.Key] = p.Value
		}
		if v := stored["minio.endpoint"]; v != "" {
			cfg.Endpoint = v
		}
		if v := stored["minio.accessKey"]; v != "" {
			cfg.AccessKey = v
		}
		if v := stored["minio.secretKey"]; v != "" {
			cfg.SecretKey = decryptConfigSecret("minio.secretKey", v)
		}
		if v := stored["minio.bucket"]; v != "" {
			cfg.Bucket = v
		}
		if v := stored["minio.useSSL"]; v != "" {
			cfg.UseSSL = v == "true"
		}
		return cfg, nil
	}
}

func minioFallback(cfg *config.Config) attachment.MinIOConfig {
	return attachment.MinIOConfig{
		Endpoint: cfg.MinIO.Endpoint, AccessKey: cfg.MinIO.AccessKey,
		SecretKey: cfg.MinIO.SecretKey, Bucket: cfg.MinIO.Bucket, UseSSL: cfg.MinIO.UseSSL,
	}
}
