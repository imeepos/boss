package order

import "context"

// Channel 下单渠道目录(REQ-ORD-006 必填不可改)。
type Channel struct {
	ID     int64
	Code   string // HALL营业厅/ONLINE线上/AGENT代理商
	Name   string
	Status string // ACTIVE启用/DISABLED停用
}

// ChannelService 下单渠道目录域服务口(阶段5)。
type ChannelService interface {
	ListChannels(ctx context.Context) ([]Channel, error)
	GetChannel(ctx context.Context, id int64) (*Channel, error)
	CreateChannel(ctx context.Context, c Channel) (int64, error)
}
