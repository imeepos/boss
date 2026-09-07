package odn

import (
	"errors"
	"testing"
)

// facilitySeqSpec/FormatFacilityCode 纯函数回归:模式拼装、容量上限、非法 kind。
func TestFacilitySeqSpec(t *testing.T) {
	t.Run("P/MH 网格模式", func(t *testing.T) {
		pattern, digits, limit, err := facilitySeqSpec(KindPole, 12)
		if err != nil || pattern != "^P12[0-9]{3}$" || digits != 3 || limit != 999 {
			t.Fatalf("pattern=%q digits=%d limit=%d err=%v", pattern, digits, limit, err)
		}
	})
	t.Run("网格越界拒绝", func(t *testing.T) {
		if _, _, _, err := facilitySeqSpec(KindManhole, 0); !errors.Is(err, ErrInvalidCode) {
			t.Fatalf("err=%v", err)
		}
		if _, _, _, err := facilitySeqSpec(KindManhole, 100); !errors.Is(err, ErrInvalidCode) {
			t.Fatalf("err=%v", err)
		}
	})
	t.Run("TW/CLS/TBX 顺序模式", func(t *testing.T) {
		pattern, digits, limit, err := facilitySeqSpec(KindTower, 0)
		if err != nil || pattern != "^TW[0-9]{5}$" || digits != 5 || limit != 99999 {
			t.Fatalf("pattern=%q digits=%d limit=%d err=%v", pattern, digits, limit, err)
		}
	})
	t.Run("非法 kind 拒绝", func(t *testing.T) {
		if _, _, _, err := facilitySeqSpec("XX", 1); !errors.Is(err, ErrInvalidCode) {
			t.Fatalf("err=%v", err)
		}
	})
}

func TestFormatFacilityCode(t *testing.T) {
	if got := FormatFacilityCode(KindPole, 12, 1); got != "P12001" {
		t.Fatalf("got=%q", got)
	}
	if got := FormatFacilityCode(KindManhole, 3, 47); got != "MH03047" {
		t.Fatalf("got=%q", got)
	}
	if got := FormatFacilityCode(KindClosure, 0, 12); got != "CLS00012" {
		t.Fatalf("got=%q", got)
	}
}
