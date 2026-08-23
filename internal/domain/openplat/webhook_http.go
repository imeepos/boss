// 默认 HTTP Poster:10s 超时 POST,忽略重定向(2xx/3xx 之外都算失败重试)。
package openplat

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"time"
)

// HTTPPoster 生产实现。
type HTTPPoster struct{ client *http.Client }

// NewHTTPPoster 构造默认投递客户端。
func NewHTTPPoster() *HTTPPoster {
	return &HTTPPoster{client: &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse // 不跟随重定向,3xx 按结果落库
		},
		Transport: &http.Transport{
			DialContext:           (&net.Dialer{Timeout: 5 * time.Second}).DialContext,
			ResponseHeaderTimeout: 10 * time.Second,
		},
	}}
}

// Post 发送并返回 HTTP 状态码。
func (p *HTTPPoster) Post(url string, headers map[string]string, body []byte) (int, error) {
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(body))
	if err != nil {
		return 0, fmt.Errorf("openplat: build request: %w", err)
	}
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := p.client.Do(req)
	if err != nil {
		return 0, fmt.Errorf("openplat: post: %w", err)
	}
	defer resp.Body.Close()
	return resp.StatusCode, nil
}
