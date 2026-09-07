package odn

import (
	"context"
	"fmt"
)

// 下一可用编码(P-INFRA-1 UX):退役行占号永久不复用(红线 2),故取现存 MAX+1。
// 设施编码主键全网唯一,P/MH 与 TW/CLS/TBX 均按编码前缀全局取号,保证可直接入库。

// facilitySeqSpec 设施编码取号规格:前缀正则、序号位数、容量上限。
// P/MH 网格分区模式(kind+2 位网格+3 位序号);TW/CLS/TBX 顺序模式(kind+5 位)。
func facilitySeqSpec(kind string, gridCode int16) (pattern string, digits, limit int, err error) {
	switch kind {
	case KindPole, KindManhole:
		if gridCode < 1 || gridCode > 99 {
			return "", 0, 0, ErrInvalidCode
		}
		return fmt.Sprintf("^%s%02d[0-9]{3}$", kind, gridCode), 3, MaxFacilities, nil
	case KindTower, KindClosure, KindTermBox:
		return fmt.Sprintf("^%s[0-9]{5}$", kind), 5, 99999, nil
	}
	return "", 0, 0, ErrInvalidCode
}

// FormatFacilityCode 按模式拼装编码(seq>=1,预留号 000/00000 由 seq 从 MAX+1 天然规避)。
func FormatFacilityCode(kind string, gridCode int16, seq int) string {
	if kind == KindPole || kind == KindManhole {
		return fmt.Sprintf("%s%02d%03d", kind, gridCode, seq)
	}
	return fmt.Sprintf("%s%05d", kind, seq)
}

// NextFacilityCode 下一可用设施编码(现存 MAX+1;超容量回 ErrGridFull)。
func (s *PGStore) NextFacilityCode(ctx context.Context, kind string, gridCode int16) (string, error) {
	pattern, _, limit, err := facilitySeqSpec(kind, gridCode)
	if err != nil {
		return "", err
	}
	var maxSeq int
	if err := s.db.QueryRow(ctx,
		`SELECT COALESCE(MAX(SUBSTRING(code FROM '[0-9]+$')::int),0) FROM odn_facility WHERE code ~ $1`,
		pattern).Scan(&maxSeq); err != nil {
		return "", fmt.Errorf("odn: next facility code %s: %w", kind, err)
	}
	if maxSeq+1 > limit {
		return "", fmt.Errorf("%w: %s 编号已用尽", ErrGridFull, kind)
	}
	return FormatFacilityCode(kind, gridCode, maxSeq+1), nil
}

// NextSiteNo 下一可用局点序号(MAX 含 RETIRED 行:局点号退役占号不复用;满 999 回 ErrSiteFull)。
func (s *PGStore) NextSiteNo(ctx context.Context, prvCode, cityPrefix string) (int16, error) {
	var maxNo int16
	if err := s.db.QueryRow(ctx,
		`SELECT COALESCE(MAX(site_no),0) FROM odn_site WHERE prv_code=$1 AND city_prefix=$2`,
		prvCode, cityPrefix).Scan(&maxNo); err != nil {
		return 0, fmt.Errorf("odn: next site no %s/%s: %w", prvCode, cityPrefix, err)
	}
	if maxNo >= 999 {
		return 0, fmt.Errorf("%w: %s%s 局点序号已用尽", ErrSiteFull, prvCode, cityPrefix)
	}
	return maxNo + 1, nil
}
