package realid

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

// aliyunCloudauth 阿里云实人认证·身份二要素核验
// (cloudauth.aliyuncs.com Id2MetaVerify RPC, POP V1 HMAC-SHA1 签名, 零 SDK 直调)。
// 签名实现与 internal/pkg/sms/aliyun_intl.go 同构(各通道自持,不跨包抽公共层)。
type aliyunCloudauth struct {
	accessKeyID     string
	accessKeySecret string
	endpoint        string
	client          *http.Client
}

// DefaultEndpoint 云auth 服务地址(官方文档: IPv4 cloudauth.aliyuncs.com)。
const DefaultEndpoint = "https://cloudauth.aliyuncs.com/"

// AliyunCloudauthConfig 阿里云实人认证二要素参数。
type AliyunCloudauthConfig struct {
	AccessKeyID     string
	AccessKeySecret string
	Endpoint        string // 空 = DefaultEndpoint;测试注入用
}

// NewAliyunCloudauth 构造二要素核验通道;凭据缺失返回 nil(=通道未启用)。
func NewAliyunCloudauth(cfg AliyunCloudauthConfig) Verifier {
	if cfg.AccessKeyID == "" || cfg.AccessKeySecret == "" {
		return nil
	}
	if cfg.Endpoint == "" {
		cfg.Endpoint = DefaultEndpoint
	}
	return &aliyunCloudauth{
		accessKeyID:     cfg.AccessKeyID,
		accessKeySecret: cfg.AccessKeySecret,
		endpoint:        cfg.Endpoint,
		client:          &http.Client{Timeout: 10 * time.Second},
	}
}

// Verify 调 Id2MetaVerify(ParamType=normal 明文);BizCode 1=一致(PASS) 2=不一致 3=查无(FAIL)。
func (s *aliyunCloudauth) Verify(ctx context.Context, name, idNo string) (string, error) {
	body, err := s.call(ctx, map[string]string{
		"Action":      "Id2MetaVerify",
		"Version":     "2019-03-07",
		"ParamType":   "normal",
		"UserName":    name,
		"IdentifyNum": idNo,
	})
	if err != nil {
		return "", fmt.Errorf("realid: aliyun cloudauth: %w", err)
	}
	// 外部协议键名非 lowerCamelCase(PascalCase),契约门禁 B 禁止违约 json tag,
	// 走 map 解码(与 stripe/sms 外部响应同口径)。
	var resp map[string]any
	if err := json.Unmarshal(body, &resp); err != nil {
		return "", fmt.Errorf("realid: aliyun cloudauth: bad response: %w", err)
	}
	code, _ := resp["Code"].(string)
	if code != "200" {
		msg, _ := resp["Message"].(string)
		return "", fmt.Errorf("realid: aliyun cloudauth: %s: %s", code, msg)
	}
	bizCode := ""
	if ro, ok := resp["ResultObject"].(map[string]any); ok {
		bizCode, _ = ro["BizCode"].(string)
	}
	switch bizCode {
	case "1":
		return Pass, nil
	case "2", "3":
		return Fail, nil
	}
	return "", fmt.Errorf("realid: aliyun cloudauth: unknown BizCode %q", bizCode)
}

// call 签发 POP V1 RPC 请求并返回响应体(同 sms 通道实现)。
func (s *aliyunCloudauth) call(ctx context.Context, params map[string]string) ([]byte, error) {
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
		s.endpoint, strings.NewReader(q.Encode()+"&Signature="+percentEncode(sig)))
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
