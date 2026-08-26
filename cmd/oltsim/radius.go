// 光猫上线模拟:RADIUS 客户端向 boss-aaa 发 Access-Request 并解析回执。
// 契约对齐 internal/domain/aaa/radius/handler.go:
//
//	User-Name = LOID(入网凭证);Access-Accept 携带 FramedPool(带宽)/SessionTimeout。
package main

import (
	"context"
	"fmt"
	"log"
	"net"

	"layeh.com/radius"
	"layeh.com/radius/rfc2865"
	"layeh.com/radius/rfc2869"
)

const codeAccessAccept = radius.CodeAccessAccept

// radiusAuth 向 boss-aaa 发 Access-Request,返回应答码与带宽(套餐生效证据)。
// layeh.com/radius 客户端模式:Exchange → 校验 Code → 读取 FramedPool/String。
func (s *Sim) radiusAuth(ctx context.Context, loid string) (radius.Code, string, error) {
	pkt := radius.New(radius.CodeAccessRequest, []byte(s.RadiusSecret))
	if err := rfc2865.UserName_SetString(pkt, loid); err != nil {
		return 0, "", fmt.Errorf("oltsim: set username: %w", err)
	}
	if err := rfc2865.NASIPAddress_Set(pkt, netIPv4("127.0.0.1")); err != nil {
		return 0, "", fmt.Errorf("oltsim: set nasip: %w", err)
	}
	resp, err := radius.Exchange(ctx, pkt, s.RadiusAddr)
	if err != nil {
		return 0, "", fmt.Errorf("oltsim: radius exchange: %w", err)
	}
	bw := rfc2869.FramedPool_GetString(resp)
	ttl := rfc2865.SessionTimeout_Get(resp)
	log.Printf("oltsim: ont online loid=%s code=%v bandwidth=%q ttl=%d",
		loid, resp.Code, bw, ttl)
	return resp.Code, bw, nil
}

// netIPv4 解析 IPv4 字符串;失败回环兜底。
func netIPv4(s string) net.IP {
	ip := net.ParseIP(s)
	if ip == nil {
		return net.IPv4(127, 0, 0, 1)
	}
	return ip
}
