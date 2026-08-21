package attachment

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestNewObjectKey(t *testing.T) {
	k := newObjectKey("Report.PDF")
	if !strings.HasSuffix(k, ".pdf") {
		t.Fatalf("ext not lowercased: %s", k)
	}
	if len(strings.SplitN(k, "/", 2)) != 2 || len(strings.SplitN(k, "/", 2)[0]) != 8 {
		t.Fatalf("date dir missing: %s", k)
	}
	if newObjectKey("a.verylongextensiontoolong") == "" {
		t.Fatal("should not be empty")
	}
	if k2 := newObjectKey("noext"); strings.Contains(k2, ".") {
		t.Fatalf("no-ext name should have no dot: %s", k2)
	}
}

func TestNewClientInvalidEndpoint(t *testing.T) {
	if _, err := newClient(MinIOConfig{Endpoint: ""}); err == nil {
		t.Fatal("empty endpoint should fail")
	}
}

func TestMinIOStoragePutInvalidEndpoint(t *testing.T) {
	s := NewMinIOStorage()
	if _, err := s.Put(context.Background(), MinIOConfig{Endpoint: ""}, strings.NewReader("x"), 1, "", "a.txt"); err == nil {
		t.Fatal("invalid endpoint should fail")
	}
}

// fakeS3 用最小 S3 语义驱动 MinIOStorage.Put 全路径。
func fakeS3(t *testing.T) (endpoint string) {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("ETag", "\"d41d8cd98f00b204e9800998ecf8427e\"")
		switch r.Method {
		case http.MethodHead: // BucketExists -> 404 表示不存在
			w.WriteHeader(http.StatusNotFound)
		case http.MethodPut: // MakeBucket / PutObject
			w.WriteHeader(http.StatusOK)
		case http.MethodGet: // location 探测
			fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?><LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/"></LocationConstraint>`)
		default:
			w.WriteHeader(http.StatusOK)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.Listener.Addr().String()
}

func TestMinIOStoragePut(t *testing.T) {
	s := NewMinIOStorage()
	key, err := s.Put(context.Background(), MinIOConfig{Endpoint: fakeS3(t), AccessKey: "a", SecretKey: "b", Bucket: "bkt"},
		strings.NewReader("hello"), 5, "", "doc.txt")
	if err != nil {
		t.Fatalf("put: %v", err)
	}
	if !strings.HasSuffix(key, ".txt") {
		t.Fatalf("unexpected key: %s", key)
	}
}

// fakeS3State 按方法返回状态码,可中途改状态模拟并发建桶等场景。
type fakeS3State struct {
	head           int // HEAD /bucket
	mkPut          int // PUT /bucket (MakeBucket)
	objPut         int
	afterMkPutHead int // MakeBucket 失败后复查 BucketExists 的 HEAD 状态码
	mkPutCalls     int
}

func startFakeS3(t *testing.T, st *fakeS3State) string {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodHead:
			code := st.head
			if st.afterMkPutHead != 0 && st.mkPutCalls > 0 {
				code = st.afterMkPutHead
			}
			w.WriteHeader(code)
		case r.Method == http.MethodPut && r.URL.Path == "/bkt":
			st.mkPutCalls++
			w.WriteHeader(st.mkPut)
		case r.Method == http.MethodPut:
			w.Header().Set("ETag", "\"d41d8cd98f00b204e9800998ecf8427e\"")
			w.WriteHeader(st.objPut)
		default:
			fmt.Fprint(w, `<?xml version="1.0" encoding="UTF-8"?><LocationConstraint xmlns="http://s3.amazonaws.com/doc/2006-03-01/"></LocationConstraint>`)
		}
	}))
	t.Cleanup(srv.Close)
	return srv.Listener.Addr().String()
}

func putTo(t *testing.T, st *fakeS3State) error {
	t.Helper()
	_, err := NewMinIOStorage().Put(context.Background(),
		MinIOConfig{Endpoint: startFakeS3(t, st), AccessKey: "a", SecretKey: "b", Bucket: "bkt"},
		strings.NewReader("hello"), 5, "text/plain", "doc.txt")
	return err
}

func TestMinIOStoragePutBucketExistsError(t *testing.T) {
	st := &fakeS3State{head: http.StatusInternalServerError}
	if err := putTo(t, st); err == nil {
		t.Fatal("BucketExists 500 should fail")
	}
}

func TestMinIOStoragePutBucketAlreadyExists(t *testing.T) {
	st := &fakeS3State{head: http.StatusOK, objPut: http.StatusOK}
	if err := putTo(t, st); err != nil {
		t.Fatalf("existing bucket path should succeed: %v", err)
	}
}

func TestMinIOStoragePutMakeBucketRaceExists(t *testing.T) {
	// MakeBucket 失败但复查桶已存在(并发撞已存在),应继续上传成功。
	st := &fakeS3State{head: http.StatusNotFound, mkPut: http.StatusInternalServerError, afterMkPutHead: http.StatusOK, objPut: http.StatusOK}
	if err := putTo(t, st); err != nil {
		t.Fatalf("race-exists should succeed: %v", err)
	}
}

func TestMinIOStoragePutObjectError(t *testing.T) {
	st := &fakeS3State{head: http.StatusOK, objPut: http.StatusInternalServerError}
	if err := putTo(t, st); err == nil {
		t.Fatal("PutObject 500 should fail")
	}
}
