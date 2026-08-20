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

func (f *fakeStore) Create(_ context.Context, at *Attachment) (*Attachment, error) {
	at.ID = 1
	f.saved = at
	return at, nil
}

func (f *fakeStore) Get(context.Context, int64) (*Attachment, error) { return nil, ErrNotFound }

func (f *fakeStore) ListByUploader(context.Context, string, int64, int) ([]Attachment, error) {
	return nil, nil
}

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
