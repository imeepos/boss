package worker

import (
	"context"
	"time"
)

// Notice 师傅公告:admin 发布/上下架,active 控制师傅端公告页可见。
type Notice struct {
	ID          int64     `json:"noticeId"`
	Title       string    `json:"title"`
	Category    string    `json:"category"`
	Active      bool      `json:"active"`
	PublishedAt time.Time `json:"publishedAt"`
}

// WorkerNoticeService 公告域服务口:列表(含已下架)/发布/上下架切换。
type WorkerNoticeService interface {
	ListNotices(ctx context.Context) ([]Notice, error)
	CreateNotice(ctx context.Context, n Notice) (int64, error)
	ToggleNotice(ctx context.Context, id int64) error
}
