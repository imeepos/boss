package push

// JPushSender REST v3 契约测试:BasicAuth、信封字段、audience 优先级、错误透传。

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestJPushSendRegistrationIDPriority(t *testing.T) {
	var gotAuth, gotPath string
	var env map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotPath = r.URL.Path
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &env)
		_, _ = w.Write([]byte(`{"sendno":"1","msg_id":123456}`))
	}))
	defer srv.Close()

	s := NewJPush(ChannelConfig{
		AppKey: "app-key", MasterSecret: "master-sec", APIURL: srv.URL,
		ApnsProduction: true, LiveTimeSec: 3600,
	}, srv.Client())
	id, err := s.Send(context.Background(), Request{
		Title: "BOSS", Alert: "test", Alias: []string{"worker:1"}, RegistrationIDs: []string{"reg-1"},
		Extras: map[string]string{"orderNo": "O1"},
	})
	if err != nil || id != "123456" {
		t.Fatalf("send=%q err=%v", id, err)
	}
	if !strings.HasPrefix(gotAuth, "Basic ") || !strings.Contains(gotPath, "/push") {
		t.Fatalf("auth/path: %q %q", gotAuth, gotPath)
	}
	aud := env["audience"].(map[string]any)
	if _, has := aud["registration_id"]; !has {
		t.Fatalf("registration_id should win: %v", aud)
	}
	if _, has := aud["alias"]; has {
		t.Fatalf("alias should be absent: %v", aud)
	}
	opts := env["options"].(map[string]any)
	if opts["time_to_live"] != float64(3600) || opts["apns_production"] != true {
		t.Fatalf("options: %v", opts)
	}
	notif := env["notification"].(map[string]any)
	android := notif["android"].(map[string]any)
	if android["title"] != "BOSS" || android["extras"].(map[string]any)["orderNo"] != "O1" {
		t.Fatalf("android notify: %v", android)
	}
}

func TestJPushSendAlias(t *testing.T) {
	var env map[string]any
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(body, &env)
		_, _ = w.Write([]byte(`{"sendno":"1","msg_id":1}`))
	}))
	defer srv.Close()

	s := NewJPush(ChannelConfig{AppKey: "k", MasterSecret: "s", APIURL: srv.URL}, srv.Client())
	if _, err := s.Send(context.Background(), Request{Alert: "a", Alias: []string{"user:9"}}); err != nil {
		t.Fatal(err)
	}
	aud := env["audience"].(map[string]any)
	if aud["alias"].([]any)[0] != "user:9" {
		t.Fatalf("alias audience: %v", aud)
	}
}

func TestJPushErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte(`{"error":{"code":1010,"message":"auth failed"}}`))
	}))
	defer srv.Close()
	s := NewJPush(ChannelConfig{AppKey: "k", MasterSecret: "s", APIURL: srv.URL}, srv.Client())

	if _, err := s.Send(context.Background(), Request{Alert: "x", Alias: []string{"a"}}); err == nil ||
		!strings.Contains(err.Error(), "1010") {
		t.Fatalf("jpush error not surfaced: %v", err)
	}
	if _, err := s.Send(context.Background(), Request{Alert: "x"}); err == nil ||
		!strings.Contains(err.Error(), "empty audience") {
		t.Fatalf("empty audience not rejected: %v", err)
	}
	noCred := NewJPush(ChannelConfig{APIURL: srv.URL}, srv.Client())
	if _, err := noCred.Send(context.Background(), Request{Alert: "x", Alias: []string{"a"}}); err == nil ||
		!strings.Contains(err.Error(), "credentials missing") {
		t.Fatalf("missing credentials not rejected: %v", err)
	}
}

func TestDynamicDisabledAndFallback(t *testing.T) {
	d := NewDynamic(func(context.Context) (ChannelConfig, error) {
		return ChannelConfig{Enabled: false}, nil
	})
	if _, err := d.Send(context.Background(), Request{Alert: "x", Alias: []string{"a"}}); err != ErrDisabled {
		t.Fatalf("disabled should error, got %v", err)
	}
	d2 := NewDynamic(func(context.Context) (ChannelConfig, error) {
		return ChannelConfig{Enabled: true}, nil // 凭据缺失 → LogSender
	})
	if _, err := d2.Send(context.Background(), Request{Alert: "x", Alias: []string{"a"}}); err != nil {
		t.Fatalf("log fallback should pass, got %v", err)
	}
}
