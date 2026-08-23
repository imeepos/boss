package push

// 设备注册表语义测试:入参校验、幂等注册、同设备改绑主体。

import (
	"context"
	"errors"
	"testing"
)

func TestDevicesRegisterAndRebind(t *testing.T) {
	s := NewDevicesMemoryStore()
	ctx := context.Background()

	if err := s.RegisterDevice(ctx, SubjectWorker, 7, "1507bfd3f9ac1e045a", "jpush"); err != nil {
		t.Fatal(err)
	}
	ids, err := s.RegistrationIDs(ctx, SubjectWorker, 7)
	if err != nil || len(ids) != 1 || ids[0] != "1507bfd3f9ac1e045a" {
		t.Fatalf("worker devices=%v err=%v", ids, err)
	}

	// 同设备换账号:改绑到 user 9,原主体不再可见。
	if err := s.RegisterDevice(ctx, SubjectUser, 9, "1507bfd3f9ac1e045a", "jpush"); err != nil {
		t.Fatal(err)
	}
	if ids, _ := s.RegistrationIDs(ctx, SubjectWorker, 7); len(ids) != 0 {
		t.Fatalf("old subject should be unbound: %v", ids)
	}
	if ids, _ := s.RegistrationIDs(ctx, SubjectUser, 9); len(ids) != 1 {
		t.Fatalf("new subject should own device: %v", ids)
	}
}

func TestDevicesValidation(t *testing.T) {
	s := NewDevicesMemoryStore()
	ctx := context.Background()
	cases := []struct {
		subjectType string
		subjectID   int64
		regID       string
	}{
		{"account", 1, "1507bfd3f9ac1e045a"},                // 主体枚举外
		{SubjectUser, 0, "1507bfd3f9ac1e045a"},              // 主体 ID 非法
		{SubjectUser, 1, "short123"},                        // RegistrationID 太短
		{SubjectUser, 1, "含中文的registrationid非常长1234567890"}, // 非字母数字
		{SubjectUser, 1, ""},                                // 空
	}
	for i, c := range cases {
		if err := s.RegisterDevice(ctx, c.subjectType, c.subjectID, c.regID, "jpush"); !errors.Is(err, ErrInvalidDevice) {
			t.Fatalf("case %d should be invalid, got %v", i, err)
		}
	}
}
