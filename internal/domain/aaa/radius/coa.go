package radius

// RFC 5176 动态授权客户端:向 NAS CoA/DM 端口(默认 3799)发 Disconnect-Request 实现强制下线。
// 实现 aaa.DisconnectSender;由 cmd/aaa 与 cmd/server 共用(仅作 UDP 客户端,不占本进程监听端口)。

import (
	"context"
	"fmt"
	"net"
	"strconv"
	"time"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2866"
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
