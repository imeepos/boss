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
		accessKeyID: "ak", accessKeySecret: "sk",
		contentCodes: map[string]string{"86": "CC86", "60": "CC60"},
		client:       &http.Client{Transport: rt},
	}
}

func TestNewAliyunIntl(t *testing.T) {
	s := NewAliyunIntl(AliyunIntlConfig{AccessKeyID: "ak", AccessKeySecret: "sk"})
	if s == nil {
		t.Fatal("nil sender")
	}
}

func TestAliyunIntl_Send(t *testing.T) {
	ctx := context.Background()

	// 成功:ResultCode=OK;PhoneNumbers 纯数字不带 +,ContentCode 按区号取。
	rt := &stubTransport{status: 200, body: `{"ResultCode":"OK","ResultMessage":"ok"}`}
	if err := newStubIntl(rt).Send(ctx, "+8613800138000", "123456", "login"); err != nil {
		t.Fatalf("Send: %v", err)
	}
	for _, want := range []string{
		"Action=SendSms", "Version=2018-05-01", "PhoneNumbers=8613800138000",
		"ContentCode=CC86", "Type=OTP", "VerificationCode=123456",
		"AccessKeyId=ak", "Signature=",
	} {
		if !strings.Contains(rt.reqBody, want) {
			t.Fatalf("reqBody missing %q: %s", want, rt.reqBody)
		}
	}

	// 业务失败:ResultCode != OK,带 ResultMessage。
	rt = &stubTransport{status: 200, body: `{"ResultCode":"MOBILE_NUMBER_ILLEGAL","ResultMessage":"bad"}`}
	err := newStubIntl(rt).Send(ctx, "+60123456789", "1", "login")
	if err == nil || !strings.Contains(err.Error(), "MOBILE_NUMBER_ILLEGAL") {
		t.Fatalf("err=%v, want ResultCode", err)
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
}

func TestAliyunIntl_SendMissingContentCode(t *testing.T) {
	// 区号支持但未配 ContentCode → ErrContentCodeMissing。
	s := &aliyunIntl{accessKeyID: "ak", accessKeySecret: "sk", client: &http.Client{}}
	err := s.Send(context.Background(), "+8613800138000", "1", "login")
	if !errors.Is(err, ErrContentCodeMissing) {
		t.Fatalf("err=%v, want ErrContentCodeMissing", err)
	}
	// 区号不支持 → ErrUnsupportedRegion。
	err = s.Send(context.Background(), "+19995551234", "1", "login")
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
	if _, err := s.call(context.Background(), map[string]string{"Action": "SendSms"}); err == nil {
		t.Fatal("want request build error")
	}
}
