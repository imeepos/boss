package attachment

// Probe 零副作用连通性自检(管理端 POST /storage-config/test 专用):
// endpoint/bucket 必填 → 客户端构造 → BucketExists;不创建、不写入。
// 错误带步骤上下文,管理端可直接展示给配置人(data-relations §6.3)。
import (
	"context"
	"errors"
	"fmt"
)

func Probe(ctx context.Context, cfg MinIOConfig) error {
	if cfg.Endpoint == "" {
		return errors.New("endpoint 未配置")
	}
	if cfg.Bucket == "" {
		return errors.New("bucket 未配置")
	}
	cli, err := newClient(cfg)
	if err != nil {
		return fmt.Errorf("客户端构造失败: %w", err)
	}
	ok, err := cli.BucketExists(ctx, cfg.Bucket)
	if err != nil {
		return fmt.Errorf("连接失败: %w", err)
	}
	if !ok {
		return fmt.Errorf("bucket %q 不存在", cfg.Bucket)
	}
	return nil
}
