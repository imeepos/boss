package order

// MemoryService 的 PrepaidAmount 实现(独立文件控制 memory.go 行数红线)。
// 内存实现无应收语义:现场收款不适用,返回 prepaid=false。

import "context"

func (s *MemoryService) PrepaidAmount(ctx context.Context, orderID int64) (float64, int, bool, error) {
	return 0, 0, false, nil
}
