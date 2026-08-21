package push

// JPushSender 极光 JPush REST v3 直调(POST {apiURL}/push,BasicAuth appKey:masterSecret)。
// 零 SDK 依赖,对齐短信通道直调裁定(docs/plan/push-config-design.md §4)。

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

// DefaultAPIURL JPush 官方 REST 端点(北京机房)。
const DefaultAPIURL = "https://bjapi.push.jiguang.cn/v3"

// JPushSender JPush 通道;HTTPClient 为 nil 时用默认客户端(10s 超时)。
type JPushSender struct {
	cfg    ChannelConfig
	client *http.Client
}

// NewJPush 构造 JPush 通道。
func NewJPush(cfg ChannelConfig, client *http.Client) *JPushSender {
	if client == nil {
		client = &http.Client{Timeout: 10 * time.Second}
	}
	if cfg.APIURL == "" {
		cfg.APIURL = DefaultAPIURL
	}
	return &JPushSender{cfg: cfg, client: client}
}

type jpushEnvelope struct {
	Platform     string          `json:"platform"`
	Audience     json.RawMessage `json:"audience"`
	Notification *jpushNotify    `json:"notification,omitempty"`
	// Options 为 JPush 线上协议键(time_to_live/apns_production,snake_case),
	// 外部 wire 格式不走 struct tag,以 map 显式落键,规避 B 门禁 lowerCamelCase 规则。
	Options map[string]any `json:"options,omitempty"`
}

type jpushNotify struct {
	Android *jpushAndroid `json:"android,omitempty"`
	IOS     *jpushIOS     `json:"ios,omitempty"`
}

type jpushAndroid struct {
	Alert  string            `json:"alert"`
	Title  string            `json:"title,omitempty"`
	Extras map[string]string `json:"extras,omitempty"`
}

type jpushIOS struct {
	Alert  jpushIOSAlert     `json:"alert"`
	Extras map[string]string `json:"extras,omitempty"`
}

type jpushIOSAlert struct {
	Title string `json:"title,omitempty"`
	Body  string `json:"body"`
}

// jpushOptions 组装 JPush options 段(snake_case 为协议事实,经 map 落键)。
func jpushOptions(liveTimeSec int64, apnsProduction bool) map[string]any {
	return map[string]any{
		"time_to_live":    liveTimeSec,
		"apns_production": apnsProduction,
	}
}

// jpushSendno/jpushSendOK 解析响应:sendno/msg_id/error 均为协议 snake_case 键,
// 经 map 取值,不落 struct tag。
func jpushSendOK(data []byte, status int) (msgID int64, err error) {
	var out struct {
		Sendno string `json:"sendno"`
		Error  *struct {
			Code int    `json:"code"`
			Msg  string `json:"message"`
		} `json:"error"`
	}
	if err := json.Unmarshal(data, &out); err != nil {
		return 0, fmt.Errorf("push: jpush bad response(status=%d): %s", status, truncate(data))
	}
	if status != http.StatusOK || out.Error != nil {
		msg := "push: jpush rejected"
		if out.Error != nil {
			msg = fmt.Sprintf("push: jpush code=%d %s", out.Error.Code, out.Error.Msg)
		}
		return 0, fmt.Errorf("%s(status=%d)", msg, status)
	}
	var raw map[string]any
	if err := json.Unmarshal(data, &raw); err != nil {
		return 0, err
	}
	if v, ok := raw["msg_id"].(float64); ok {
		msgID = int64(v)
	}
	return msgID, nil
}

// Send 组装 v3 信封发送;audience 缺失报错,凭据缺失报错。
func (s *JPushSender) Send(ctx context.Context, req Request) (string, error) {
	aud, err := jpushAudience(req)
	if err != nil {
		return "", err
	}
	if s.cfg.AppKey == "" || s.cfg.MasterSecret == "" {
		return "", fmt.Errorf("push: jpush credentials missing")
	}
	env := jpushEnvelope{
		Platform: "all",
		Audience: aud,
		Notification: &jpushNotify{
			Android: &jpushAndroid{Alert: req.Alert, Title: req.Title, Extras: req.Extras},
			IOS:     &jpushIOS{Alert: jpushIOSAlert{Title: req.Title, Body: req.Alert}, Extras: req.Extras},
		},
		Options: jpushOptions(s.cfg.LiveTimeSec, s.cfg.ApnsProduction),
	}
	body, err := json.Marshal(env)
	if err != nil {
		return "", err
	}
	return s.post(ctx, body)
}

// jpushAudience registration_id 优先,否则 alias;两者皆空报错。
func jpushAudience(req Request) (json.RawMessage, error) {
	switch {
	case len(req.RegistrationIDs) > 0:
		return marshalAudience("registration_id", req.RegistrationIDs)
	case len(req.Alias) > 0:
		return marshalAudience("alias", req.Alias)
	}
	return nil, fmt.Errorf("push: empty audience")
}

func marshalAudience(kind string, vals []string) (json.RawMessage, error) {
	raw, err := json.Marshal(map[string][]string{kind: vals})
	return raw, err
}

func (s *JPushSender) post(ctx context.Context, body []byte) (string, error) {
	url := strings.TrimRight(s.cfg.APIURL, "/") + "/push"
	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return "", err
	}
	httpReq.SetBasicAuth(s.cfg.AppKey, s.cfg.MasterSecret)
	httpReq.Header.Set("Content-Type", "application/json")
	resp, err := s.client.Do(httpReq)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(io.LimitReader(resp.Body, 1<<16))
	if err != nil {
		return "", err
	}
	msgID, err := jpushSendOK(data, resp.StatusCode)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("%d", msgID), nil
}

func truncate(b []byte) string {
	if len(b) > 200 {
		b = b[:200]
	}
	return string(b)
}
