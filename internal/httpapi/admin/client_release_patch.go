package adminapi

import (
	"github.com/ymm-001/boss/internal/domain/apprelease"
)

// releasePatchBody PATCH /client-releases/:id 请求体:指针字段不传不变,
// whitelistIds 非 nil 即覆盖(空数组=清空)。
type releasePatchBody struct {
	Version          *string `json:"version"`
	VersionCode      *int    `json:"versionCode"`
	MinSupportedCode *int    `json:"minSupportedCode"`
	Notes            *string `json:"notes"`
	Force            *bool   `json:"force"`
	Status           *string `json:"status"`
	RolloutPercent   *int    `json:"rolloutPercent"`
	WhitelistIDs     []int64 `json:"whitelistIds"`
}

// applyReleasePatch PATCH 语义:仅覆盖请求出现的字段。
// (空数组=清空白名单,不传=不变,用 body 里的 nil/非 nil 区分)。
func applyReleasePatch(r *apprelease.Release, b *releasePatchBody) {
	if b.Version != nil {
		r.Version = *b.Version
	}
	if b.VersionCode != nil {
		r.VersionCode = *b.VersionCode
	}
	if b.MinSupportedCode != nil {
		r.MinSupportedCode = *b.MinSupportedCode
	}
	if b.Notes != nil {
		r.Notes = *b.Notes
	}
	if b.Force != nil {
		r.Force = *b.Force
	}
	if b.Status != nil {
		r.Status = *b.Status
	}
	if b.RolloutPercent != nil {
		r.RolloutPercent = *b.RolloutPercent
	}
	if b.WhitelistIDs != nil {
		r.WhitelistIDs = b.WhitelistIDs
	}
}
