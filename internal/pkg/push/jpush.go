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
	Options      *jpushOptions   `json:"options,omitempty"`
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

type jpushOptions struct {
	TimeToLive    int64 `json:"time_to_live,omitempty"`
	ApnsProduction bool `json:"apns_production"`
}

type jpushResp struct {
	Sendno string `json:"sendno"`
	MsgID  int64  `json:"msg_id"`
	Error  *struct {
		Code int    `json:"code"`
		Msg  string `json:"message"`
	} `json:"error"`
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
		Options: &jpushOptions{TimeToLive: s.cfg.LiveTimeSec, ApnsProduction: s.cfg.ApnsProduction},
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
	var out jpushResp
	if err := json.Unmarshal(data, &out); err != nil {
		return "", fmt.Errorf("push: jpush bad response(status=%d): %s", resp.StatusCode, truncate(data))
	}
	if resp.StatusCode != http.StatusOK || out.Error != nil {
		msg := "push: jpush rejected"
		if out.Error != nil {
			msg = fmt.Sprintf("push: jpush code=%d %s", out.Error.Code, out.Error.Msg)
		}
		return "", fmt.Errorf("%s(status=%d)", msg, resp.StatusCode)
	}
	return fmt.Sprintf("%d", out.MsgID), nil
}

func truncate(b []byte) string {
	if len(b) > 200 {
		b = b[:200]
	}
	return string(b)
}
