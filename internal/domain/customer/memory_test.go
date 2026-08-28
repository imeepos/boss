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
		name  string
		q     CustomerQuery
		wantN int
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

func TestMemoryServiceGetInScope(t *testing.T) {
	s := NewMemoryService()
	inID, _ := s.Create(context.Background(), Customer{Name: "范围内", LegalEntityID: 7, RegionName: "马尼拉"})
	_, _ = s.Create(context.Background(), Customer{Name: "他企客户", LegalEntityID: 8, RegionName: "马尼拉"})
	_, _ = s.Create(context.Background(), Customer{Name: "区域外", LegalEntityID: 7, RegionName: "达沃"})

	tests := []struct {
		name    string
		id      int64
		le      int64
		region  string
		wantErr error
	}{
		{"范围内命中", inID, 7, "马尼拉", nil},
		{"范围不限全命中", inID, 0, "", nil},
		{"他企越界按不存在", 2, 7, "马尼拉", ErrCustomerNotFound},
		{"区域外越界按不存在", 3, 7, "马尼拉", ErrCustomerNotFound},
		{"不存在同错误", 999, 0, "", ErrCustomerNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := s.GetInScope(context.Background(), tt.id, tt.le, tt.region)
			if tt.wantErr != nil {
				if err != tt.wantErr {
					t.Fatalf("err = %v, want %v", err, tt.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatalf("GetInScope err = %v", err)
			}
			if c.ID != tt.id {
				t.Fatalf("id = %d, want %d", c.ID, tt.id)
			}
		})
	}
}
