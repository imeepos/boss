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

// aliyunIntl 阿里云国际短信(dysmsapiintl.aliyuncs.com, SendSMS RPC, POP V1 签名)。
// 中国大陆与马来西亚现阶段统一走此通道;国内报备通道就绪后改路由即可。
type aliyunIntl struct {
	accessKeyID     string
	accessKeySecret string
	from            string            // 发送方 SenderID(阿里云国际控制台申请)
	templates       map[string]string // 区号 → 文案模板,{code} 占位
	client          *http.Client
}

// AliyunIntlConfig 阿里云国际短信参数。
type AliyunIntlConfig struct {
	AccessKeyID     string
	AccessKeySecret string
	From            string
	Templates       map[string]string
}

// DefaultTemplates 内置双语文案:86 中文 / 60 英文(马来西亚)。
func DefaultTemplates() map[string]string {
	return map[string]string{
		"86": "您的验证码为{code}，5分钟内有效，请勿泄露。",
		"60": "Your verification code is {code}. Valid for 5 minutes. Do not share it.",
	}
}

// NewAliyunIntl 构造阿里云国际短信通道;Templates 为 nil 时用 DefaultTemplates。
func NewAliyunIntl(cfg AliyunIntlConfig) Sender {
	tpl := cfg.Templates
	if len(tpl) == 0 {
		tpl = DefaultTemplates()
	}
	return &aliyunIntl{
		accessKeyID:     cfg.AccessKeyID,
		accessKeySecret: cfg.AccessKeySecret,
		from:            cfg.From,
		templates:       tpl,
		client:          &http.Client{Timeout: 10 * time.Second},
	}
}

// Send 调用国际短信 SendSMS(Type=NONOTP, Message=渲染后文案)。
func (s *aliyunIntl) Send(ctx context.Context, phone, code, _ string) error {
	tpl, ok := s.templates[Region(phone)]
	if !ok {
		return fmt.Errorf("%w: +%s", ErrUnsupportedRegion, Region(phone))
	}
	body, err := s.call(ctx, map[string]string{
		"Action":     "SendSMS",
		"Version":    "2018-05-01",
		"To":         phone,
		"From":       s.from,
		"Type":       "NONOTP",
		"Message":    strings.ReplaceAll(tpl, "{code}", code),
		"MessageTag": Region(phone),
	})
	if err != nil {
		return fmt.Errorf("sms: aliyun intl: %w", err)
	}
	// 成功判定:ResponseCode == "OK"(阿里云国际 POP 风格 JSON 响应)。
	// 外部协议键名非 lowerCamelCase,走 map 解码而非结构体 tag(契约门禁红线)。
	var resp map[string]string
	if err := json.Unmarshal(body, &resp); err != nil {
		return fmt.Errorf("sms: aliyun intl: bad response: %w", err)
	}
	if resp["ResponseCode"] != "OK" {
		return fmt.Errorf("sms: aliyun intl: %s: %s", resp["ResponseCode"], resp["ResponseDescription"])
	}
	return nil
}

// aliyunEndpoint POP 入口(测试可注入非法 URL 触发构造失败分支)。
var aliyunEndpoint = "https://dysmsapiintl.aliyuncs.com/"

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
