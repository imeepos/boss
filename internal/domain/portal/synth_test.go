package portal

import (
	"context"
	"testing"
)

// 回归(000057 负数号段裁定): 合成 ID 必须为负,与真实 customers.id 正数段隔离。
func TestSyntheticIDNegative(t *testing.T) {
	for _, seq := range []int64{1, 2, 999} {
		if got := syntheticID(seq); got >= 0 {
			t.Fatalf("syntheticID(%d)=%d, want negative", seq, got)
		}
	}
	s := NewMemory().(*memoryStore)
	for i := 0; i < 3; i++ {
		id, err := s.NextSyntheticCustomerID(context.Background())
		if err != nil {
			t.Fatal(err)
		}
		if id >= 0 {
			t.Fatalf("NextSyntheticCustomerID #%d = %d, want negative", i+1, id)
		}
	}
}
