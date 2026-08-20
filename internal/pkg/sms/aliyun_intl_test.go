package sms

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
)

// stubTransport 按 HTTP 状态/响应体/错误打桩,并捕获请求体。
type stubTransport struct {
	status  int
	body    string
	err     error
	reqBody string
}

func (s *stubTransport) RoundTrip(r *http.Request) (*http.Response, error) {
	b, _ := io.ReadAll(r.Body)
	s.reqBody = string(b)
	if s.err != nil {
		return nil, s.err
	}
	return &http.Response{
		StatusCode: s.status, Body: io.NopCloser(strings.NewReader(s.body)),
		Header: make(http.Header), Request: r,
	}, nil
}

func newStubIntl(rt http.RoundTripper) *aliyunIntl {
	return &aliyunIntl{
		accessKeyID: "ak", accessKeySecret: "sk", from: "BOSS",
		templates: DefaultTemplates(), client: &http.Client{Transport: rt},
	}
}

func TestDefaultTemplates(t *testing.T) {
	tpl := DefaultTemplates()
	if tpl["86"] == "" || tpl["60"] == "" {
		t.Fatalf("tpl=%v", tpl)
	}
}

func TestNewAliyunIntl(t *testing.T) {
	s := NewAliyunIntl(AliyunIntlConfig{AccessKeyID: "ak", AccessKeySecret: "sk", From: "F"})
	if s == nil {
		t.Fatal("nil sender")
	}
	// Templates 为空时用 DefaultTemplates。
	a, ok := s.(*aliyunIntl)
	if !ok || a.templates["86"] == "" {
		t.Fatalf("default templates missing: %+v", a)
	}
}

func TestAliyunIntl_Send(t *testing.T) {
	ctx := context.Background()

	// 成功:ResponseCode=OK,模板 {code} 已渲染。
	rt := &stubTransport{status: 200, body: `{"ResponseCode":"OK"}`}
	if err := newStubIntl(rt).Send(ctx, "+8613800138000", "123456", "login"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	if !strings.Contains(rt.reqBody, "Message=%E6%82%A8") || !strings.Contains(rt.reqBody, "123456") {
		t.Fatalf("reqBody=%s", rt.reqBody)
	}
	if !strings.Contains(rt.reqBody, "AccessKeyId=ak") || !strings.Contains(rt.reqBody, "Signature=") {
		t.Fatalf("signed params missing: %s", rt.reqBody)
	}

	// 业务失败:ResponseCode != OK。
	rt = &stubTransport{status: 200, body: `{"ResponseCode":"isv.BUSY","ResponseDescription":"busy"}`}
	err := newStubIntl(rt).Send(ctx, "+60123456789", "1", "login")
	if err == nil || !strings.Contains(err.Error(), "isv.BUSY") {
		t.Fatalf("err=%v, want ResponseCode", err)
	}

	// 响应非 JSON。
	rt = &stubTransport{status: 200, body: "not-json"}
	err = newStubIntl(rt).Send(ctx, "+8613800138000", "1", "login")
	if err == nil || !strings.Contains(err.Error(), "bad response") {
		t.Fatalf("err=%v, want bad response", err)
	}

	// 网络失败。
	rt = &stubTransport{err: errors.New("dial fail")}
	err = newStubIntl(rt).Send(ctx, "+8613800138000", "1", "login")
	if err == nil || !strings.Contains(err.Error(), "aliyun intl") {
		t.Fatalf("err=%v, want transport error wrap", err)
	}

	// 区号无模板。
	err = newStubIntl(&stubTransport{}).Send(ctx, "+19995551234", "1", "login")
	if !errors.Is(err, ErrUnsupportedRegion) {
		t.Fatalf("err=%v, want ErrUnsupportedRegion", err)
	}
}

func TestPercentEncode(t *testing.T) {
	cases := map[string]string{
		"a b": "a%20b",
		"a*b": "a%2Ab",
		"a~b": "a~b",
		"abc": "abc",
	}
	for in, want := range cases {
		if got := percentEncode(in); got != want {
			t.Errorf("percentEncode(%q)=%q, want %q", in, got, want)
		}
	}
}

func TestNonce(t *testing.T) {
	n1, n2 := nonce(), nonce()
	if len(n1) != 16 || n1 == n2 {
		t.Fatalf("nonce=%q,%q", n1, n2)
	}
}

func TestAliyunIntl_CallBadEndpoint(t *testing.T) {
	old := aliyunEndpoint
	defer func() { aliyunEndpoint = old }()
	aliyunEndpoint = "http://bad\x7f" // 非法 URL 触发 NewRequest 失败分支
	s := newStubIntl(&stubTransport{})
	if _, err := s.call(context.Background(), map[string]string{"Action": "SendSMS"}); err == nil {
		t.Fatal("want request build error")
	}
}
