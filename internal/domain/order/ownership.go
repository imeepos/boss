package order

import (
	"context"
	"errors"
	"fmt"
)

// ErrAddressNotCovered 安装地址未挂经营区域,或所在区域(含祖先)无运营主体覆盖。
// PG 实现已由平台总公司兜底(migrations/000077),此错误仅在内存实现/配置缺失时出现。
var ErrAddressNotCovered = errors.New("order: address not covered by any legal entity")

// ErrPlatformMissing 未配置平台总公司(is_platform=true),兜底链断裂,属配置错误。
var ErrPlatformMissing = errors.New("order: platform legal entity not configured")

// ErrOwnershipMismatch 调用方传入的 LegalEntityID 与地址推导结果冲突。
var ErrOwnershipMismatch = errors.New("order: legal entity mismatch with address")

// AddressOwnership 安装地址推导出的归属(运营主体 + 覆盖区域路径)。
type AddressOwnership struct {
	LegalEntityID int64
	RegionPath    string
}

// OwnershipResolver 地址→归属推导口(环节1 前置)。
// PG 实现内嵌于 PGStore.Submit;内存实现供 MemoryService/单测注入。
type OwnershipResolver interface {
	Resolve(ctx context.Context, addressID int64) (AddressOwnership, error)
}

// OwnershipMap 内存版 OwnershipResolver:addressID → 归属。查不到返回 ErrAddressNotCovered。
type OwnershipMap map[int64]AddressOwnership

// Resolve 按 map 直查。
func (m OwnershipMap) Resolve(_ context.Context, addressID int64) (AddressOwnership, error) {
	o, ok := m[addressID]
	if !ok {
		return AddressOwnership{}, fmt.Errorf("address %d: %w", addressID, ErrAddressNotCovered)
	}
	return o, nil
}

// checkOwnershipConflict 调用方显式传了 LegalEntityID 时做一致性校验,不一致拒单。
func checkOwnershipConflict(req SubmitReq, own AddressOwnership) error {
	if req.LegalEntityID != 0 && req.LegalEntityID != own.LegalEntityID {
		return fmt.Errorf("req=%d derived=%d: %w", req.LegalEntityID, own.LegalEntityID, ErrOwnershipMismatch)
	}
	return nil
}
