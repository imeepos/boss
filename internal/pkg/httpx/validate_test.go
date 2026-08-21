package httpx

import (
	"testing"
)

func TestRequireString(t *testing.T) {
	tests := []struct {
		name    string
		value   string
		field   string
		maxLen  int
		wantErr bool
	}{
		{"ok", "hello", "name", 64, false},
		{"empty", "", "name", 64, true},
		{"spaces only", "   ", "name", 64, true},
		{"exceeds max", "hello", "name", 3, true},
		{"no limit", "a long string", "name", 0, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := RequireString(tt.value, tt.field, tt.maxLen)
			if (err != nil) != tt.wantErr {
				t.Fatalf("RequireString(%q, %q, %d) = %v, wantErr=%v", tt.value, tt.field, tt.maxLen, err, tt.wantErr)
			}
		})
	}
}

func TestRequirePositiveID(t *testing.T) {
	if err := RequirePositiveID(1, "id"); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if err := RequirePositiveID(0, "id"); err == nil {
		t.Fatal("expected error for 0")
	}
	if err := RequirePositiveID(-1, "id"); err == nil {
		t.Fatal("expected error for -1")
	}
}

func TestRequirePositiveFloat(t *testing.T) {
	if err := RequirePositiveFloat(1.5, "fee"); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if err := RequirePositiveFloat(0, "fee"); err == nil {
		t.Fatal("expected error for 0")
	}
	if err := RequirePositiveFloat(-1, "fee"); err == nil {
		t.Fatal("expected error for -1")
	}
}

func TestRequireEnum(t *testing.T) {
	if err := RequireEnum("A", "status", "A", "B", "C"); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if err := RequireEnum("D", "status", "A", "B", "C"); err == nil {
		t.Fatal("expected error for invalid enum")
	}
	if err := RequireEnum("", "status", "A", "B"); err == nil {
		t.Fatal("expected error for empty")
	}
}

func TestRequirePhone(t *testing.T) {
	if err := RequirePhone("", "phone"); err != nil {
		t.Fatalf("expected nil for empty (optional), got %v", err)
	}
	if err := RequirePhone("+63 917-123-4567", "phone"); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
	if err := RequirePhone("ab", "phone"); err == nil {
		t.Fatal("expected error for too short")
	}
	if err := RequirePhone("abc!@#", "phone"); err == nil {
		t.Fatal("expected error for invalid chars")
	}
}

func TestCollectErrors(t *testing.T) {
	v1 := RequirePositiveID(0, "id")
	v2 := RequireString("", "name", 64)
	if err := CollectErrors(v1, v2); err == nil {
		t.Fatal("expected error")
	}
	if err := CollectErrors(nil, nil); err != nil {
		t.Fatalf("expected nil, got %v", err)
	}
}

func TestCollectAllErrors(t *testing.T) {
	v1 := RequirePositiveID(0, "id")
	v2 := RequireString("", "name", 64)
	v3 := RequirePositiveFloat(1.0, "fee")
	errs := CollectAllErrors(v1, v2, v3)
	if len(errs) != 2 {
		t.Fatalf("expected 2 errors, got %d", len(errs))
	}
}
