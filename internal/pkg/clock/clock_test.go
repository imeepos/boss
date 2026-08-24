package clock

import (
	"testing"
	"time"
)

func TestSetAndLocation(t *testing.T) {
	if err := Set("Asia/Manila"); err != nil {
		t.Fatalf("set Asia/Manila: %v", err)
	}
	t.Cleanup(func() { _ = Set("UTC") })
	if got := Location().String(); got != "Asia/Manila" {
		t.Fatalf("location = %s, want Asia/Manila", got)
	}
	if err := Set("no/such-zone"); err == nil {
		t.Fatal("unknown zone should error")
	}
	if err := Set(""); err != nil {
		t.Fatalf("empty name should be no-op: %v", err)
	}
}

func TestSetFixed(t *testing.T) {
	Set("UTC")
	t.Cleanup(func() { SetFixed(time.Time{}) })
	fixedTime := time.Date(2026, 8, 21, 2, 0, 0, 0, time.UTC)
	SetFixed(fixedTime)
	if !Now().Equal(fixedTime) {
		t.Fatalf("Now() = %v, want %v", Now(), fixedTime)
	}
	// 固定时刻优先于业务时区:Now() 返回钉死的绝对时刻本身。
	if err := Set("Asia/Manila"); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = Set("UTC") })
	if !Now().Equal(fixedTime) {
		t.Fatalf("Now() = %v, want fixed %v after tz change", Now(), fixedTime)
	}
	SetFixed(time.Time{})
	if Now().Equal(fixedTime) {
		t.Fatal("zero value should clear fixed time")
	}
}

func TestDayBounds(t *testing.T) {
	if err := Set("Asia/Manila"); err != nil {
		t.Fatalf("set: %v", err)
	}
	t.Cleanup(func() { _ = Set("UTC") })
	// Manila=UTC+8:马尼拉 2026-08-21 01:00 == UTC 2026-08-20 17:00,
	// 业务日应属 08-21,日界为 UTC 08-20T16:00 与 08-21T16:00。
	utcTime := time.Date(2026, 8, 20, 17, 0, 0, 0, time.UTC)
	start, end := DayBounds(utcTime)
	if want := time.Date(2026, 8, 20, 16, 0, 0, 0, time.UTC); !start.Equal(want) {
		t.Fatalf("start = %v, want %v", start, want)
	}
	if want := time.Date(2026, 8, 21, 16, 0, 0, 0, time.UTC); !end.Equal(want) {
		t.Fatalf("end = %v, want %v", end, want)
	}
}
