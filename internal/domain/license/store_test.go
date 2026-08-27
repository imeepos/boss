package license

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreRoundTrip(t *testing.T) {
	p := filepath.Join(t.TempDir(), "lic", "license.json")
	s := &Store{Path: p}
	if err := s.Save(context.Background(), []byte(`{"payload":"x"}`)); err != nil {
		t.Fatal(err)
	}
	got, err := s.Load(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != `{"payload":"x"}` {
		t.Fatalf("roundtrip mismatch: %s", got)
	}
}

func TestStoreLoadMissing(t *testing.T) {
	s := &Store{Path: filepath.Join(t.TempDir(), "nope.json")}
	_, err := s.Load(context.Background())
	if !errors.Is(err, ErrNoLicense) {
		t.Fatalf("want ErrNoLicense, got %v", err)
	}
}

func TestStoreSaveEmpty(t *testing.T) {
	s := &Store{Path: filepath.Join(t.TempDir(), "l.json")}
	if err := s.Save(context.Background(), nil); !errors.Is(err, ErrInvalidToken) {
		t.Fatalf("want ErrInvalidToken, got %v", err)
	}
}

func TestStoreClear(t *testing.T) {
	p := filepath.Join(t.TempDir(), "l.json")
	s := &Store{Path: p}
	_ = s.Save(context.Background(), []byte(`{}`))
	if err := s.Clear(context.Background()); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(p); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("file should be gone, err=%v", err)
	}
	// 清除不存在的文件幂等成功。
	if err := s.Clear(context.Background()); err != nil {
		t.Fatalf("clear missing file should be ok: %v", err)
	}
}