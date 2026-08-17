package user

import "context"

// Service 阶段1:账号/角色/权限/区域地址层级。
// 权限变更即时生效:RBAC 快照写 Redis,校验走快照。
type Service interface {
	Login(ctx context.Context, username, password string) (token string, err error)
	HasPermission(ctx context.Context, accountID int64, permCode string) (bool, error)
	ListAddresses(ctx context.Context, parentID int64) ([]Address, error)
	ImportAddresses(ctx context.Context, rows []AddressRow) (imported int, err error)
}

type Address struct {
	ID       int64
	ParentID int64
	Level    int8 // 1市 2区 3街道 4小区 5楼栋
	Name     string
}

type AddressRow struct {
	Path string // ltree 路径,如 bj.chaoyang.wangjing.xq1.ld2
	Name string
}
