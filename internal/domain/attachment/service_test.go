package attachment

import (
	"context"
	"errors"
	"io"
	"strings"
	"testing"
)

type fakeStore struct {
	saved *Attachment
}

// getByIDStore 让 Get 命中指定附件(Download 成功路径)。
type getByIDStore struct{ fakeStore }

func (g *getByIDStore) Get(context.Context, int64) (*Attachment, error) {
	return &Attachment{ID: 7, ObjectKey: "20260101/ab", FileName: "f.json", ContentType: "application/json"}, nil
}

func (f *fakeStore) Create(_ context.Context, at *Attachment) (*Attachment, error) {
	at.ID = 1
	f.saved = at
	return at, nil
}

func (f *fakeStore) Get(context.Context, int64) (*Attachment, error) { return nil, ErrNotFound }

func (f *fakeStore) ListByUploader(context.Context, string, int64, int) ([]Attachment, error) {
	return nil, nil
}

func (f *fakeStore) List(context.Context, ListFilter) ([]Attachment, int, error) { return nil, 0, nil }

func (f *fakeStore) Delete(context.Context, int64) error { return nil }

func (f *fakeStore) GetByIDs(context.Context, []int64) ([]Attachment, error) { return nil, nil }

type fakeObj struct {
	key string
	err error
	got MinIOConfig
}

func (f *fakeObj) Put(_ context.Context, cfg MinIOConfig, _ io.Reader, _ int64, _, _ string) (string, error) {
	f.got = cfg
	if f.err != nil {
		return "", f.err
	}
	return f.key, nil
}

func (f *fakeObj) Open(_ context.Context, cfg MinIOConfig, _ string) (io.ReadCloser, error) {
	f.got = cfg
	if f.err != nil {
		return nil, f.err
	}
	return io.NopCloser(strings.NewReader("obj-bytes")), nil
}

func TestUploadRecordsUploader(t *testing.T) {
	svc := &Service{St: &fakeStore{}, Obj: &fakeObj{key: "20260101/abc.pdf"}, Conf: MinIOConfig{Bucket: "b"}}
	at, err := svc.Upload(context.Background(), &Attachment{
		FileName: "报告.pdf", ContentType: "application/pdf",
		UploaderType: UploaderCustomer, UploaderID: 42,
	}, strings.NewReader("x"), 1)
	if err != nil {
		t.Fatalf("upload: %v", err)
	}
	if at.ObjectKey != "20260101/abc.pdf" || at.ID != 1 {
		t.Fatalf("unexpected result: %+v", at)
	}
	st := svc.St.(*fakeStore).saved
	if st.UploaderType != "customer" || st.UploaderID != 42 {
		t.Fatalf("uploader not recorded: %+v", st)
	}
}

func TestUploadRejectsInvalidUploader(t *testing.T) {
	svc := &Service{St: &fakeStore{}, Obj: &fakeObj{key: "k"}}
	if _, err := svc.Upload(context.Background(), &Attachment{UploaderType: "guest", UploaderID: 1}, strings.NewReader("x"), 1); err == nil {
		t.Fatal("invalid uploader type should fail")
	}
	if _, err := svc.Upload(context.Background(), &Attachment{UploaderType: UploaderAccount, UploaderID: 0}, strings.NewReader("x"), 1); err == nil {
		t.Fatal("zero uploader id should fail")
	}
}

// 合成客户(负数段隔离空间 ID)上传证件照:修复前 UploaderID<=0 一刀切拒绝 →
// App 实名上传 50000 内部错误;现仅拒 0,负数合法。
func TestUploadSyntheticCustomerID(t *testing.T) {
	svc := &Service{St: &fakeStore{}, Obj: &fakeObj{key: "k"}, Conf: MinIOConfig{Bucket: "b"}}
	at, err := svc.Upload(context.Background(), &Attachment{
		FileName: "id_front.png", ContentType: "image/png",
		UploaderType: UploaderCustomer, UploaderID: -9,
	}, strings.NewReader("x"), 1)
	if err != nil {
		t.Fatalf("synthetic customer upload: %v", err)
	}
	if at.UploaderID != -9 {
		t.Fatalf("uploader id not echoed: %+v", at)
	}
	if st := svc.St.(*fakeStore).saved; st.UploaderID != -9 {
		t.Fatalf("uploader id not recorded: %+v", st)
	}
}

func TestValidUploaderType(t *testing.T) {
	for _, ok := range []string{"account", "worker", "customer"} {
		if !ValidUploaderType(ok) {
			t.Fatalf("%s should be valid", ok)
		}
	}
	if ValidUploaderType("admin") {
		t.Fatal("admin should be invalid")
	}
}

func TestUploadResolveConfig(t *testing.T) {
	obj := &fakeObj{key: "k"}
	svc := &Service{St: &fakeStore{}, Obj: obj,
		Resolve: func(context.Context) (MinIOConfig, error) { return MinIOConfig{Bucket: "resolved"}, nil }}
	if _, err := svc.Upload(context.Background(), &Attachment{UploaderType: UploaderAccount, UploaderID: 1}, strings.NewReader("x"), 1); err != nil {
		t.Fatalf("upload: %v", err)
	}
	if obj.got.Bucket != "resolved" {
		t.Fatalf("resolved config not used: %+v", obj.got)
	}

	wantErr := errors.New("resolve boom")
	svc.Resolve = func(context.Context) (MinIOConfig, error) { return MinIOConfig{}, wantErr }
	if _, err := svc.Upload(context.Background(), &Attachment{UploaderType: UploaderAccount, UploaderID: 1}, strings.NewReader("x"), 1); !errors.Is(err, wantErr) {
		t.Fatalf("resolve error should propagate: %v", err)
	}
}

func TestUploadObjPutError(t *testing.T) {
	svc := &Service{St: &fakeStore{}, Obj: &fakeObj{err: errors.New("put boom")}}
	if _, err := svc.Upload(context.Background(), &Attachment{UploaderType: UploaderWorker, UploaderID: 2}, strings.NewReader("x"), 1); err == nil {
		t.Fatal("obj put error should fail")
	}
}

func TestDownloadReturnsMetaAndStream(t *testing.T) {
	obj := &fakeObj{}
	svc := &Service{St: &getByIDStore{}, Obj: obj, Conf: MinIOConfig{Bucket: "b"}, Resolve: func(context.Context) (MinIOConfig, error) {
		return MinIOConfig{Endpoint: "e", Bucket: "resolved"}, nil
	}}
	at, r, err := svc.Download(context.Background(), 7)
	if err != nil {
		t.Fatalf("download: %v", err)
	}
	defer r.Close()
	if at.FileName != "f.json" || obj.got.Bucket != "resolved" {
		t.Fatalf("unexpected: %+v cfg=%+v", at, obj.got)
	}
	body, _ := io.ReadAll(r)
	if string(body) != "obj-bytes" {
		t.Fatalf("unexpected body: %q", body)
	}
}

func TestDownloadNotFound(t *testing.T) {
	svc := &Service{St: &fakeStore{}, Obj: &fakeObj{}}
	if _, _, err := svc.Download(context.Background(), 404); !errors.Is(err, ErrNotFound) {
		t.Fatalf("want ErrNotFound, got %v", err)
	}
}
