package odn

import (
	"context"
	"errors"
	"fmt"
	"log"

	"github.com/jackc/pgx/v5"
)

// 生命周期状态(P6 设计-施工,迁移 000198;与在用性 status 列分离,RETIRED 双列同步)。
const (
	LCPlanned   = "PLANNED"    // 规划已备案
	LCInBuild   = "IN_BUILD"   // 施工中
	LCInService = "IN_SERVICE" // 在网运行
	LCRetired   = "RETIRED"    // 退役(终态)
)

// ErrInvalidLifecycle 非法生命周期转移(未知状态/终态逆转/并发改写 CAS 失败)。
var ErrInvalidLifecycle = errors.New("odn: invalid lifecycle transition")

// lifecycleTransitions 允许的转移:线性推进 + 各态可径直退役;RETIRED 终态不可逆。
var lifecycleTransitions = map[string][]string{
	LCPlanned:   {LCInBuild, LCInService, LCRetired},
	LCInBuild:   {LCInService, LCRetired},
	LCInService: {LCRetired},
	LCRetired:   {},
}

// ValidateLifecycleTransition 转移合法性(同态 no-op 放行)。
func ValidateLifecycleTransition(from, to string) error {
	if from == to {
		return nil
	}
	for _, nxt := range lifecycleTransitions[from] {
		if nxt == to {
			return nil
		}
	}
	return ErrInvalidLifecycle
}

// setLifecycle 通用 CAS 转移:WHERE 带 from 态防并发双写;RETIRED 同步 status 列。
// 表名/定位列来自固定白名单调用点,无注入面。
func (s *PGStore) setLifecycle(ctx context.Context, entity, table, where string, args []any, to string) error {
	from, err := s.currentLifecycle(ctx, entity, table, where, args)
	if err != nil {
		return err
	}
	if err := ValidateLifecycleTransition(from, to); err != nil {
		return err
	}
	n := len(args)
	q := fmt.Sprintf("UPDATE %s SET lifecycle_status=$%d, status=CASE WHEN $%d='RETIRED' THEN 'RETIRED' ELSE status END WHERE %s AND lifecycle_status=$%d",
		table, n+1, n+2, where, n+3)
	all := append(append([]any{}, args...), to, to, from)
	tag, err := s.db.Exec(ctx, q, all...)
	if err != nil {
		log.Printf("[odn-lifecycle] UPDATE FAILED entity=%s to=%s: %v", entity, to, err)
		return fmt.Errorf("odn: lifecycle update %s: %w", entity, err)
	}
	if tag.RowsAffected() == 0 {
		return ErrInvalidLifecycle // 并发改写:from 态已漂移
	}
	return nil
}

// currentLifecycle 读当前态(不存在 → ErrNotFound)。
func (s *PGStore) currentLifecycle(ctx context.Context, entity, table, where string, args []any) (string, error) {
	var cur string
	err := s.db.QueryRow(ctx, "SELECT lifecycle_status FROM "+table+" WHERE "+where, args...).Scan(&cur)
	if errors.Is(err, pgx.ErrNoRows) {
		return "", ErrNotFound
	}
	if err != nil {
		return "", fmt.Errorf("odn: lifecycle read %s: %w", entity, err)
	}
	return cur, nil
}

// SetFacilityLifecycle 设施生命周期转移。
func (s *PGStore) SetFacilityLifecycle(ctx context.Context, code, to string) error {
	return s.setLifecycle(ctx, "facility", "odn_facility", "code=$1", []any{code}, to)
}

// SetSiteLifecycle 局点生命周期转移。
func (s *PGStore) SetSiteLifecycle(ctx context.Context, prvCode, cityPrefix string, siteNo int16, to string) error {
	return s.setLifecycle(ctx, "site", "odn_site", "prv_code=$1 AND city_prefix=$2 AND site_no=$3",
		[]any{prvCode, cityPrefix, siteNo}, to)
}

// SetDeviceLifecycle 设备生命周期转移。
func (s *PGStore) SetDeviceLifecycle(ctx context.Context, id int64, to string) error {
	return s.setLifecycle(ctx, "device", "odn_device", "id=$1", []any{id}, to)
}
