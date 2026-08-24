// Package sms 验证码短信通道抽象:按手机号区号路由到地区通道。
// 当前唯一生产通道为阿里云国际短信(dysmsapi ap-southeast-1);中国大陆(+86)后续可无缝切换国内报备通道。
package sms

import (
	"context"
	"errors"
	"fmt"
	"strings"
)

// ErrUnsupportedRegion 区号不在支持列表(当前仅 +86 中国 / +60 马来西亚)。
var ErrUnsupportedRegion = errors.New("sms: unsupported region")

// ErrContentCodeMissing 区号受支持但未配置该区号的报备模板 ContentCode。
var ErrContentCodeMissing = errors.New("sms: content code not configured")

// Sender 验证码短信发送通道。phone 为 E.164(带 + 前缀)。
type Sender interface {
	Send(ctx context.Context, phone, code, scene string) error
}

// Region 从 E.164 号码取区号(纯数字),格式非法返回空。
func Region(phone string) string {
	d := digits(phone)
	// 仅覆盖已开通市场:86 中国 / 60 马来西亚;新市场在此追加。
	for _, cc := range []string{"86", "60"} {
		if strings.HasPrefix(d, cc) && len(d) > len(cc) {
			return cc
		}
	}
	return ""
}

// NormalizeE164 宽松归一化:接受 +86138... / 86138... / 138...(裸号默认中国 +86)。
func NormalizeE164(phone string) (string, error) {
	d := digits(phone)
	switch {
	case strings.HasPrefix(phone, "+"):
		if Region("+"+d) == "" {
			return "", ErrUnsupportedRegion
		}
		return "+" + d, nil
	case Region("+"+d) != "":
		return "+" + d, nil
	case len(d) == 11 && strings.HasPrefix(d, "1"): // 无区号裸号默认中国手机号
		return "+86" + d, nil
	default:
		return "", ErrUnsupportedRegion
	}
}

// digits 去掉空格/连字符/括号后仅保留数字(前导 + 由调用方处理)。
func digits(phone string) string {
	var b strings.Builder
	for _, r := range phone {
		if r >= '0' && r <= '9' {
			b.WriteRune(r)
		}
	}
	return b.String()
}

// Router 按区号路由:ByRegion 命中优先,未命中走 Default。
// 后续接入中国国内报备通道时,只需注入 ByRegion["86"]。
type Router struct {
	Default   Sender
	ByRegion  map[string]Sender
	supported []string
}

// NewRouter 构造路由器;supported 为允许的区号列表(如 []string{"86","60"})。
func NewRouter(def Sender, byRegion map[string]Sender, supported []string) *Router {
	return &Router{Default: def, ByRegion: byRegion, supported: supported}
}

// Send 归一化号码并按区号路由发送。
func (r *Router) Send(ctx context.Context, phone, code, scene string) error {
	e164, err := NormalizeE164(phone)
	if err != nil {
		return err
	}
	cc := Region(e164)
	if !contains(r.supported, cc) {
		return fmt.Errorf("%w: +%s", ErrUnsupportedRegion, cc)
	}
	if s, ok := r.ByRegion[cc]; ok && s != nil {
		return s.Send(ctx, e164, code, scene)
	}
	if r.Default == nil {
		return errors.New("sms: no sender configured")
	}
	return r.Default.Send(ctx, e164, code, scene)
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
