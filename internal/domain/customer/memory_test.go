package customer

import (
	"context"
	"testing"
)

func TestMemoryServiceCreateGet(t *testing.T) {
	s := NewMemoryService()
	id, err := s.Create(context.Background(), Customer{Name: "王先生", Phone: "138****1234"})
	if err != nil {
		t.Fatalf("Create err = %v", err)
	}
	if id <= 0 {
		t.Fatalf("id = %d, want > 0", id)
	}
	c, err := s.Get(context.Background(), id)
	if err != nil {
		t.Fatalf("Get err = %v", err)
	}
	if c.Name != "王先生" {
		t.Fatalf("Name = %q, want 王先生", c.Name)
	}
}

func TestMemoryServiceGetNotFound(t *testing.T) {
	s := NewMemoryService()
	if _, err := s.Get(context.Background(), 99); err != ErrCustomerNotFound {
		t.Fatalf("err = %v, want ErrCustomerNotFound", err)
	}
}

func TestMemoryServiceList(t *testing.T) {
	s := NewMemoryService()
	_, _ = s.Create(context.Background(), Customer{Name: "王先生", ServiceStatus: "ACTIVE"})
	_, _ = s.Create(context.Background(), Customer{Name: "吴女士", ServiceStatus: "ARREARS"})

	tests := []struct {
		name   string
		q      CustomerQuery
		wantN  int
	}{
		{"全部", CustomerQuery{}, 2},
		{"按关键字", CustomerQuery{NameKeyword: "王"}, 1},
		{"按状态", CustomerQuery{Status: "ARREARS"}, 1},
		{"limit", CustomerQuery{Limit: 1}, 1},
		{"无匹配", CustomerQuery{NameKeyword: "不存在"}, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := s.List(context.Background(), tt.q)
			if err != nil {
				t.Fatalf("List err = %v", err)
			}
			if len(got) != tt.wantN {
				t.Fatalf("len = %d, want %d", len(got), tt.wantN)
			}
		})
	}
}
