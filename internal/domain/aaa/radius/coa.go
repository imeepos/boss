package radius

// RFC 5176 动态授权客户端:向 NAS CoA/DM 端口(默认 3799)发 Disconnect-Request 实现强制下线。
// 实现 aaa.DisconnectSender;由 cmd/aaa 与 cmd/server 共用(仅作 UDP 客户端,不占本进程监听端口)。

import (
	"context"
	"errors"
	"fmt"
	"log"
	"net"
	"strconv"
	"time"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2866"

	"github.com/ymm-001/boss/internal/domain/aaa"
)

// DefaultCoAPort NAS 动态授权默认端口(RFC 5176 约定俗成,可经 BOSS_AAA_COA_PORT 覆盖)。
const DefaultCoAPort = 3799

// CoAClient aaa.DisconnectSender 实现。
type CoAClient struct {
	Secret  []byte        // NAS 共享密钥(全局,与 RADIUS 认证/计费同源)
	Port    int           // NAS CoA/DM 端口
	Timeout time.Duration // 单次交换超时
}

// NewCoAClient 构造;port<=0 回退 3799,timeout<=0 回退 5s。
func NewCoAClient(secret []byte, port int, timeout time.Duration) *CoAClient {
	if port <= 0 {
		port = DefaultCoAPort
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &CoAClient{Secret: secret, Port: port, Timeout: timeout}
}

// SendDisconnect 发 Disconnect-Request;ACK=nil,NAK/超时/网络错误=err(NAS 不可达即走重试链路)。
func (c *CoAClient) SendDisconnect(ctx context.Context, nasIP, loid, sessionID string) error {
	if nasIP == "" {
		return fmt.Errorf("aaa: disconnect %s: nas ip empty", loid)
	}
	addr := net.JoinHostPort(nasIP, strconv.Itoa(c.Port))
	if c.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.Timeout)
		defer cancel()
	}
	resp, err := (&radius.Client{}).Exchange(ctx, buildDisconnectPacket(c.Secret, loid, sessionID), addr)
	if err != nil {
		return fmt.Errorf("aaa: disconnect exchange %s: %w", addr, err)
	}
	if resp.Code != radius.CodeDisconnectACK {
		return fmt.Errorf("aaa: disconnect rejected by %s: code=%s", addr, resp.Code)
	}
	return nil
}

// buildDisconnectPacket 组包:User-Name + Acct-Session-Id(NAS 定位会话的最小属性集)。
func buildDisconnectPacket(secret []byte, loid, sessionID string) *radius.Packet {
	pkt := radius.New(radius.CodeDisconnectRequest, secret)
	_ = rfc2865.UserName_SetString(pkt, loid)
	_ = rfc2866.AcctSessionID_SetString(pkt, sessionID)
	return pkt
}

// NasCoAClient aaa.DisconnectSender 实现(A5):按目标 NAS 注册表取自己的密钥与端口;
// 未注册且兼容开关开启时回退全局密钥/端口,其余未命中路径拒绝并留痕。
// 密钥不符由 NAS 侧验证失败体现(超时/NAK),经 [aaa] COA SEND FAILED 留痕。
type NasCoAClient struct {
	Registry aaa.NasResolver
	Global   []byte        // 兼容回退密钥(cfg.AAA.Secret)
	Port     int           // 兼容回退端口(cfg.AAA.CoAPort,默认 3799)
	Timeout  time.Duration // 单次交换超时
	Compat   bool          // BOSS_AAA_GLOBAL_SECRET_COMPAT
}

// NewNasCoAClient 构造;port<=0 回退 3799,timeout<=0 回退 5s。
func NewNasCoAClient(registry aaa.NasResolver, global []byte, port int, timeout time.Duration, compat bool) *NasCoAClient {
	if port <= 0 {
		port = DefaultCoAPort
	}
	if timeout <= 0 {
		timeout = 5 * time.Second
	}
	return &NasCoAClient{Registry: registry, Global: global, Port: port, Timeout: timeout, Compat: compat}
}

// SendDisconnect 按注册表解析目标 NAS 密钥/端口后发 Disconnect-Request;ACK=nil。
func (c *NasCoAClient) SendDisconnect(ctx context.Context, nasIP, loid, sessionID string) error {
	if nasIP == "" {
		return fmt.Errorf("aaa: disconnect %s: nas ip empty", loid)
	}
	secret, port, err := c.resolve(ctx, nasIP, loid, sessionID)
	if err != nil {
		return err
	}
	addr := net.JoinHostPort(nasIP, strconv.Itoa(port))
	if c.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, c.Timeout)
		defer cancel()
	}
	resp, err := (&radius.Client{}).Exchange(ctx, buildDisconnectPacket(secret, loid, sessionID), addr)
	if err != nil {
		log.Printf("[aaa] COA SEND FAILED addr=%s loid=%s session=%s: %v", addr, loid, sessionID, err)
		return fmt.Errorf("aaa: disconnect exchange %s: %w", addr, err)
	}
	if resp.Code != radius.CodeDisconnectACK {
		log.Printf("[aaa] COA NAK addr=%s loid=%s session=%s code=%s", addr, loid, sessionID, resp.Code)
		return fmt.Errorf("aaa: disconnect rejected by %s: code=%s", addr, resp.Code)
	}
	return nil
}

// resolve 目标 NAS 密钥与端口:注册表命中=per-NAS;未注册+兼容开关=全局;其余拒绝留痕。
func (c *NasCoAClient) resolve(ctx context.Context, nasIP, loid, sessionID string) ([]byte, int, error) {
	nas, err := c.Registry.LookupNas(ctx, nasIP)
	if err == nil {
		return nas.Secret, nas.Client.CoAPort, nil
	}
	if errors.Is(err, aaa.ErrNasNotFound) && c.Compat && len(c.Global) > 0 {
		log.Printf("[aaa] COA COMPAT GLOBAL SECRET ip=%s loid=%s session=%s", nasIP, loid, sessionID)
		return c.Global, c.Port, nil
	}
	switch {
	case errors.Is(err, aaa.ErrNasNotFound):
		log.Printf("[aaa] COA NAS REJECT UNREGISTERED ip=%s loid=%s session=%s", nasIP, loid, sessionID)
	case errors.Is(err, aaa.ErrNasDisabled):
		log.Printf("[aaa] COA NAS REJECT DISABLED ip=%s loid=%s: %v", nasIP, loid, err)
	default:
		log.Printf("[aaa] COA NAS LOOKUP FAILED ip=%s loid=%s: %v", nasIP, loid, err)
	}
	return nil, 0, err
}
