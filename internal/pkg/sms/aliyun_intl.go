package sms

import (
	"context"
	"crypto/hmac"
	crand "crypto/rand"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"sort"
	"strings"
	"time"
)

// aliyunIntl 阿里云国际短信(dysmsapi ap-southeast-1, SendSms RPC, POP V1 签名)。
// 2026-08-26 实测修正:旧端点 dysmsapiintl.aliyuncs.com 已不存在(NXDOMAIN),
// 真实端点/参数以测试账号实证为准(PhoneNumbers+ContentCode, 响应 ResultCode/ResultMessage)。
type aliyunIntl struct {
	accessKeyID     string
	accessKeySecret string
	contentCodes    map[string]string // 区号 → 控制台报备模板 ContentCode
	client          *http.Client
}

// AliyunIntlConfig 阿里云国际短信参数;ContentCodes 键为区号(86/60)。
type AliyunIntlConfig struct {
	AccessKeyID     string
	AccessKeySecret string
	ContentCodes    map[string]string
}

// NewAliyunIntl 构造阿里云国际短信通道。
func NewAliyunIntl(cfg AliyunIntlConfig) Sender {
	return &aliyunIntl{
		accessKeyID:     cfg.AccessKeyID,
		accessKeySecret: cfg.AccessKeySecret,
		contentCodes:    cfg.ContentCodes,
		client:          &http.Client{Timeout: 10 * time.Second},
	}
}

// Send 调用国际短信 SendSms(Type=OTP, 验证码走 VerificationCode)。
// PhoneNumbers 为纯数字(不带 +,带 + 报 MOBILE_NUMBER_ILLEGAL,已实测)。
func (s *aliyunIntl) Send(ctx context.Context, phone, code, _ string) error {
	cc := Region(phone)
	tplCode := s.contentCodes[cc]
	if tplCode == "" {
		if cc == "" {
			return fmt.Errorf("%w: %s", ErrUnsupportedRegion, phone)
		}
		return fmt.Errorf("%w: +%s", ErrContentCodeMissing, cc)
	}
	body, err := s.call(ctx, map[string]string{
		"Action":           "SendSms",
		"Version":          "2018-05-01",
		"PhoneNumbers":     digits(phone),
		"ContentCode":      tplCode,
		"Type":             "OTP",
		"VerificationCode": code,
	})
	if err != nil {
		return fmt.Errorf("sms: aliyun intl: %w", err)
	}
	// 成功判定:ResultCode == "OK"(阿里云国际 POP 风格 JSON 响应,实测键名)。
	// 外部协议键名非 lowerCamelCase,走 map 解码而非结构体 tag(契约门禁红线)。
	var resp map[string]string
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("sms: aliyun intl: bad response: %w", err)
	}
	if resp["ResultCode"] != "OK" {
		return fmt.Errorf("sms: aliyun intl: %s: %s", resp["ResultCode"], resp["ResultMessage"])
	}
	return nil
}

// aliyunEndpoint POP 入口(测试可注入非法 URL 触发构造失败分支)。
var aliyunEndpoint = "https://dysmsapi.ap-southeast-1.aliyuncs.com/"

// call 签发 POP V1 RPC 请求(HMAC-SHA1)并返回响应体。
func (s *aliyunIntl) call(ctx context.Context, params map[string]string) ([]byte, error) {
	q := url.Values{}
	for k, v := range params {
		q.Set(k, v)
	}
	q.Set("AccessKeyId", s.accessKeyID)
	q.Set("SignatureMethod", "HMAC-SHA1")
	q.Set("SignatureVersion", "1.0")
	q.Set("SignatureNonce", nonce())
	q.Set("Timestamp", time.Now().UTC().Format("2006-01-02T15:04:05Z"))
	q.Set("Format", "JSON")

	keys := make([]string, 0, len(q))
	for k := range q {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	var canon strings.Builder
	for i, k := range keys {
		if i > 0 {
			canon.WriteByte('&')
		}
		canon.WriteString(percentEncode(k) + "=" + percentEncode(q.Get(k)))
	}
	stringToSign := "POST&" + percentEncode("/") + "&" + percentEncode(canon.String())
	mac := hmac.New(sha1.New, []byte(s.accessKeySecret+"&"))
	mac.Write([]byte(stringToSign))
	sig := base64.StdEncoding.EncodeToString(mac.Sum(nil))

	req, err := http.NewRequestWithContext(ctx, http.MethodPost,
		aliyunEndpoint, strings.NewReader(q.Encode()+"&Signature="+percentEncode(sig)))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	return io.ReadAll(io.LimitReader(resp.Body, 1<<16))
}

// percentEncode POP 规范的 RFC3986 编码(+ → %20, * → %2A, ~ 不编码)。
func percentEncode(s string) string {
	s = url.QueryEscape(s)
	s = strings.ReplaceAll(s, "+", "%20")
	s = strings.ReplaceAll(s, "*", "%2A")
	return s
}

// nonce 随机防重放串。
func nonce() string {
	b := make([]byte, 8)
	_, _ = crand.Read(b)
	return hex.EncodeToString(b)
}
