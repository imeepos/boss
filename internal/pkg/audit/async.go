package audit

import (
	"context"
	"log"
	"sync"
)

// AsyncWriter 异步审计写入器:Write 非阻塞入队,后台 worker 批量落库;Close 优雅排空。
// 用于不阻塞业务主路径的审计;丢失策略为"尽力而为"(落库失败仅记日志,不反压业务)。
type AsyncWriter struct {
	next Writer
	q    chan Event
	wg   sync.WaitGroup
	once sync.Once
}

// NewAsyncWriter 构造异步写入器;buf 为队列深度。
func NewAsyncWriter(next Writer, buf int) *AsyncWriter {
	a := &AsyncWriter{next: next, q: make(chan Event, buf)}
	a.wg.Add(1)
	go a.worker()
	return a
}

func (a *AsyncWriter) worker() {
	defer a.wg.Done()
	for e := range a.q {
		if err := a.next.Write(context.Background(), e); err != nil {
			log.Printf("audit: async write failed: %v", err)
		}
	}
}

// Write 非阻塞入队;队列满时丢弃并记日志(审计是尽力而为,不阻塞主路径)。
func (a *AsyncWriter) Write(ctx context.Context, e Event) error {
	select {
	case a.q <- e:
		return nil
	default:
		log.Printf("audit: queue full, drop event target=%s/%s", e.TargetType, e.TargetID)
		return nil
	}
}

// List 透传同步查询(审计查询走同步路径,见 writer)。
func (a *AsyncWriter) List(ctx context.Context, q Query) ([]Entry, error) {
	return a.next.List(ctx, q)
}

// Close 关闭入队并等待 worker 排空队列后退出;幂等。
func (a *AsyncWriter) Close() {
	a.once.Do(func() {
		close(a.q)
	})
	a.wg.Wait()
}
