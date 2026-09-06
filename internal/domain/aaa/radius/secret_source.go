package radius

// AAA-A5(G7):按请求来源 IP 的 per-NAS 密钥校验。未注册/停用 NAS 一律拒绝
// (静默丢弃报文——RADIUS 协议对非注册客户端无合法可签名响应),并在本层输出
// [aaa] 可 grep 告警留痕;兼容开关(BOSS_AAA_GLOBAL_SECRET_COMPAT)开启时未注册
// NAS 回退全局密钥(迁移缓冲,默认关;停用 NAS 不回退,显式拒绝)。

import (
	"context"
	"errors"
	"log"
	"net"

	"github.com/ymm-001/boss/internal/domain/aaa"
)

// RegistrySecretSource layeh radius.SecretSource 实现:查 NAS 注册表取 per-NAS 密钥。
type RegistrySecretSource struct {
	Registry aaa.NasResolver
	Global   []byte // 兼容开关开启时对未注册 NAS 使用的全局密钥
	Compat   bool   // BOSS_AAA_GLOBAL_SECRET_COMPAT=true 启用全局回退
}

// RADIUSSecret 实现 radius.SecretSource;返回错误=库层丢弃报文(库层仅内部日志,
// 故分类留痕在本层先行完成)。
func (s *RegistrySecretSource) RADIUSSecret(ctx context.Context, remoteAddr net.Addr) ([]byte, error) {
	ip := hostOnly(remoteAddr)
	nas, err := s.Registry.LookupNas(ctx, ip)
	if err == nil {
		return nas.Secret, nil
	}
	switch {
	case errors.Is(err, aaa.ErrNasNotFound) && s.Compat && len(s.Global) > 0:
		log.Printf("[aaa] NAS COMPAT GLOBAL SECRET ip=%s", ip)
		return s.Global, nil
	case errors.Is(err, aaa.ErrNasNotFound):
		log.Printf("[aaa] NAS REJECT UNREGISTERED ip=%s", ip)
	case errors.Is(err, aaa.ErrNasDisabled):
		log.Printf("[aaa] NAS REJECT DISABLED ip=%s: %v", ip, err)
	default:
		log.Printf("[aaa] NAS LOOKUP FAILED ip=%s: %v", ip, err)
	}
	return nil, err
}

// hostOnly 取远端地址的 IP 部分(剥端口;解析失败原样返回,交由注册表未命中路径拒绝)。
func hostOnly(addr net.Addr) string {
	if addr == nil {
		return ""
	}
	host, _, err := net.SplitHostPort(addr.String())
	if err != nil {
		return addr.String()
	}
	return host
}
