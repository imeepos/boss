// Package apprelease 客户端版本发布域:client_releases 单表承载双端(user/worker)
// App 发版。契约见 docs/contract/fields.md 8F;裁定见 adopted/2026-08-28-app-release-domain.md。
package apprelease

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"errors"
	"fmt"
)

// 状态机(terms.md):DRAFT→GRAY(比例+白名单分桶)→PUBLISHED(全量);
// ROLLED_BACK 撤回不可再投放。
const (
	StatusDraft      = "DRAFT"
	StatusGray       = "GRAY"
	StatusPublished  = "PUBLISHED"
	StatusRolledBack = "ROLLED_BACK"
)

// 端枚举:app 字段域。
const (
	AppUser   = "user"
	AppWorker = "worker"
)

// PlatformAndroid 当前唯一平台。
const PlatformAndroid = "android"

var (
	// ErrNotFound 发版记录不存在。
	ErrNotFound = errors.New("apprelease: release not found")
	// ErrInvalid 字段校验失败。
	ErrInvalid = errors.New("apprelease: invalid release")
	// ErrDuplicateVersion (app,platform,version_code) 已存在。
	ErrDuplicateVersion = errors.New("apprelease: version code already exists")
)

// Release client_releases 行投影;时间为展示格式字符串。
type Release struct {
	ID               int64   `json:"id"`
	App              string  `json:"app"`
	Platform         string  `json:"platform"`
	Version          string  `json:"version"`
	VersionCode      int     `json:"versionCode"`
	MinSupportedCode int     `json:"minSupportedCode"`
	Notes            string  `json:"notes"`
	Force            bool    `json:"force"`
	Status           string  `json:"status"`
	RolloutPercent   int     `json:"rolloutPercent"`
	WhitelistIDs     []int64 `json:"whitelistIds"`
	ApkObjectKey     string  `json:"apkObjectKey"`
	ApkSize          int64   `json:"apkSize"`
	Sha256           string  `json:"sha256"`
	CreatedAt        string  `json:"createdAt"`
	UpdatedAt        string  `json:"updatedAt"`
}

// Store 发版记录存取接口。
type Store interface {
	// Create 落库并回填自增 id/createdAt;version_code 撞唯一键返回 ErrDuplicateVersion。
	Create(ctx context.Context, r *Release) error
	// Get 按 id 查;不存在返回 ErrNotFound。
	Get(ctx context.Context, id int64) (*Release, error)
	// List 按 app 过滤(id 新→旧);app 空查全部双端。
	List(ctx context.Context, app string) ([]Release, error)
	// Update 全量更新(id 定位);不存在返回 ErrNotFound。
	Update(ctx context.Context, r *Release) error
	// Actives 取 app+platform 的投放候选:最高 version_code 的 GRAY 与 PUBLISHED
	// 各一行(没有则空)。客户端检查更新的判定输入。
	Actives(ctx context.Context, app, platform string) ([]Release, error)
}

// validate 落库前校验;status/force 等默认值由调用方补齐后再校验。
func (r *Release) validate() error {
	if r.App != AppUser && r.App != AppWorker {
		return ErrInvalid
	}
	if r.Platform == "" {
		r.Platform = PlatformAndroid
	}
	if r.Platform != PlatformAndroid || r.Version == "" || r.VersionCode <= 0 {
		return ErrInvalid
	}
	switch r.Status {
	case StatusDraft, StatusGray, StatusPublished, StatusRolledBack:
	default:
		return ErrInvalid
	}
	if r.RolloutPercent < 0 || r.RolloutPercent > 100 {
		return ErrInvalid
	}
	if r.MinSupportedCode < 1 {
		return ErrInvalid
	}
	return nil
}

// CheckResult GET /client/latest 响应数据(updateAvailable=false 时仅回 latestVersion)。
type CheckResult struct {
	UpdateAvailable bool   `json:"updateAvailable"`
	Force           bool   `json:"force"`
	Version         string `json:"version"`
	VersionCode     int    `json:"versionCode"`
	Notes           string `json:"notes"`
	Sha256          string `json:"sha256"`
	Size            int64  `json:"size"`
	ReleaseID       int64  `json:"releaseId"`
}

// grayHit 确定性灰度分桶:sha256(deviceId:releaseId) 前 8 字节取模 100。
// 同一设备在灰度期间命中状态稳定,不随查询次数闪变(adopted note 裁定)。
func grayHit(deviceID string, releaseID int64, percent int) bool {
	if percent <= 0 {
		return false
	}
	if percent >= 100 {
		return true
	}
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s:%d", deviceID, releaseID)))
	bucket := binary.BigEndian.Uint64(sum[:8]) % 100
	return int(bucket) < percent
}

// Decide 升级判定(fields.md 8F):candidates 为 Store.Actives 结果。
// 规则:GRAY 未命中(分桶+白名单均不中)回落最近 PUBLISHED;客户端已是最新则
// updateAvailable=false;force = 客户端 versionCode < minSupportedCode 或 force。
func Decide(clientVersionCode int, deviceID string, subjectID int64, candidates []Release) CheckResult {
	var gray, published *Release
	for i := range candidates {
		switch candidates[i].Status {
		case StatusGray:
			gray = &candidates[i]
		case StatusPublished:
			published = &candidates[i]
		}
	}
	target := pickTarget(gray, published, deviceID, subjectID)
	if target == nil || target.VersionCode <= clientVersionCode {
		return CheckResult{}
	}
	force := target.Force || clientVersionCode < target.MinSupportedCode
	return CheckResult{
		UpdateAvailable: true, Force: force,
		Version: target.Version, VersionCode: target.VersionCode,
		Notes: target.Notes, Sha256: target.Sha256, Size: target.ApkSize,
		ReleaseID: target.ID,
	}
}

// pickTarget 灰度门槛:命中(分桶或白名单)取 GRAY,否则回落 PUBLISHED。
func pickTarget(gray, published *Release, deviceID string, subjectID int64) *Release {
	if gray != nil {
		if grayHit(deviceID, gray.ID, gray.RolloutPercent) || containsID(gray.WhitelistIDs, subjectID) {
			return gray
		}
	}
	return published
}

func containsID(ids []int64, id int64) bool {
	for _, v := range ids {
		if v == id && v != 0 {
			return true
		}
	}
	return false
}
