// WebhookDispatcher 测试:fake store + httptest 接收端,验证签名头、成功/失败落库与退避。
package openplat

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// fakeWebhookStore 记录型 store 桩。
type fakeWebhookStore struct {
	mu         sync.Mutex
	inserted   int64
	due        []DueDelivery
	results    map[int64]string // id -> "ok"/"fail"
	markCalls  int
	lastErrMsg string
}

func (f *fakeWebhookStore) InsertDeliveries(context.Context, string, string, []byte) (int64, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.inserted++
	return f.inserted, nil
}
func (f *fakeWebhookStore) ListDue(context.Context, time.Time, int) ([]DueDelivery, error) {
	f.mu.Lock()
	defer f.mu.Unlock()
	return append([]DueDelivery(nil), f.due...), nil
}
func (f *fakeWebhookStore) MarkResult(_ context.Context, id int64, ok bool, status int, errMsg string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.markCalls++
	if ok {
		f.results[id] = "ok"
	} else {
		f.results[id] = "fail"
		f.lastErrMsg = errMsg
	}
	return nil
}
func (f *fakeWebhookStore) ListDeliveries(context.Context, int64) ([]Delivery, error) {
	return nil, nil
}
func (f *fakeWebhookStore) Requeue(context.Context, int64) error { return nil }

// newReceiver 起一个验签接收端:校验 X-BOSS-Signature 的 v1 与 SignPayload 一致。
func newReceiver(t *testing.T, secret string, status int) (*httptest.Server, *sync.Map) {
	t.Helper()
	got := &sync.Map{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body := make([]byte, r.ContentLength)
		_, _ = r.Body.Read(body)
		sig := r.Header.Get("X-BOSS-Signature")
		ts := r.Header.Get("X-BOSS-Timestamp")
		want := "t=" + ts + ",v1=" + SignPayload(secret, ts, body)
		got.Store(r.Header.Get("X-BOSS-EventID"), sig == want)
		w.WriteHeader(status)
	}))
	return srv, got
}

func TestDeliverDueSignsPayloadAndMarksDone(t *testing.T) {
	const secret = "ops_test"
	srv, got := newReceiver(t, secret, 200)
	defer srv.Close()

	store := &fakeWebhookStore{results: map[int64]string{}}
	store.due = []DueDelivery{{
		Delivery:    Delivery{ID: 7, EventID: "evt-1", EventType: "openplat.test", Payload: `{"a":1}`},
		EndpointURL: srv.URL, Secret: secret,
	}}
	d := NewWebhookDispatcher(store, NewHTTPPoster())

	n, err := d.DeliverDue(context.Background())
	if err != nil || n != 1 {
		t.Fatalf("DeliverDue: n=%d err=%v", n, err)
	}
	if store.results[7] != "ok" {
		t.Fatalf("want ok, got %q (err=%s)", store.results[7], store.lastErrMsg)
	}
	v, _ := got.Load("evt-1")
	if ok, _ := v.(bool); !ok {
		t.Fatal("receiver signature verification failed")
	}
}

func TestDeliverDueRetryOnFailure(t *testing.T) {
	const secret = "ops_test"
	srv, _ := newReceiver(t, secret, 500) // 接收端恒 500
	defer srv.Close()

	store := &fakeWebhookStore{results: map[int64]string{}}
	store.due = []DueDelivery{{
		Delivery:    Delivery{ID: 8, EventID: "evt-2", Payload: `{}`},
		EndpointURL: srv.URL, Secret: secret,
	}}
	d := NewWebhookDispatcher(store, NewHTTPPoster())

	if _, err := d.DeliverDue(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.results[8] != "fail" || !strings.Contains(store.lastErrMsg, "500") {
		t.Fatalf("want fail w/ http 500, got %q err=%q", store.results[8], store.lastErrMsg)
	}
}

func TestDeliverDueNetworkErrorMarksFail(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	srv.Close() // 立即关闭 → 连接失败

	store := &fakeWebhookStore{results: map[int64]string{}}
	store.due = []DueDelivery{{Delivery: Delivery{ID: 9}, EndpointURL: srv.URL, Secret: "s"}}
	d := NewWebhookDispatcher(store, NewHTTPPoster())

	if _, err := d.DeliverDue(context.Background()); err != nil {
		t.Fatal(err)
	}
	if store.results[9] != "fail" || store.lastErrMsg == "" {
		t.Fatalf("network error not recorded: %+v", store.results)
	}
}

func TestBackoffExponentialCapped(t *testing.T) {
	cases := []struct {
		attempts int
		want     time.Duration
	}{
		{1, 30 * time.Second},
		{2, time.Minute},
		{3, 2 * time.Minute},
		{13, time.Hour}, // 封顶
	}
	for _, c := range cases {
		if got := Backoff(c.attempts); got != c.want {
			t.Fatalf("Backoff(%d)=%v want %v", c.attempts, got, c.want)
		}
	}
}

func TestEmitSerializesPayload(t *testing.T) {
	store := &fakeWebhookStore{}
	d := NewWebhookDispatcher(store, nil)
	n, err := d.Emit(context.Background(), "order.activated", "ORD-1:activated", map[string]string{"orderNo": "ORD-1"})
	if err != nil || n != 1 {
		t.Fatalf("Emit: n=%d err=%v", n, err)
	}
	if store.inserted != 1 {
		t.Fatalf("inserted=%d", store.inserted)
	}
}

func TestEmitRejectsBadPayload(t *testing.T) {
	d := NewWebhookDispatcher(&fakeWebhookStore{}, nil)
	_, err := d.Emit(context.Background(), "e", "id", func() {}) // 通道不可序列化
	if err == nil {
		t.Fatal("expected marshal error")
	}
}
