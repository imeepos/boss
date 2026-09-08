package odn

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5"
)

// 下一可用编码(P-INFRA-1 UX):退役行占号永久不复用(红线 2),故取现存 MAX+1。
// 设施编码主键全网唯一,P/MH 与 TW/CLS/TBX 均按编码前缀全局取号,保证可直接入库。

// facilitySeq 设施编码取号规格:编码正则、序号起始位(1-based)、序号位数、容量上限。
// 序号切片口径必须与 ValidateFacilityCode 一致:P/MH = 前缀+2 位网格之后的 3 位,
// TW/CLS/TBX = 前缀之后的 5 位。历史缺陷:按 '[0-9]+$' 贪婪取数把网格位并入序号,
// P91011 被读成 91011 > 999,网格 91 全量误报 ErrGridFull(W7 验收实锤)。
type facilitySeq struct {
	pattern string
	offset  int
	digits  int
	limit   int
}

// facilitySeqSpec 取号规格:P/MH 网格分区模式(kind+2 位网格+3 位序号);
// TW/CLS/TBX 顺序模式(kind+5 位)。
func facilitySeqSpec(kind string, gridCode int16) (facilitySeq, error) {
	switch kind {
	case KindPole, KindManhole:
		if gridCode < 1 || gridCode > 99 {
			return facilitySeq{}, ErrInvalidCode
		}
		return facilitySeq{
			pattern: fmt.Sprintf("^%s%02d[0-9]{3}$", kind, gridCode),
			offset:  len(kind) + 2 + 1,
			digits:  3,
			limit:   MaxFacilities,
		}, nil
	case KindTower, KindClosure, KindTermBox:
		return facilitySeq{
			pattern: fmt.Sprintf("^%s[0-9]{5}$", kind),
			offset:  len(kind) + 1,
			digits:  5,
			limit:   99999,
		}, nil
	}
	return facilitySeq{}, ErrInvalidCode
}

// FormatFacilityCode 按模式拼装编码(seq>=1,预留号 000/00000 由 seq 从 MAX+1 天然规避)。
func FormatFacilityCode(kind string, gridCode int16, seq int) string {
	if kind == KindPole || kind == KindManhole {
		return fmt.Sprintf("%s%02d%03d", kind, gridCode, seq)
	}
	return fmt.Sprintf("%s%05d", kind, seq)
}

// seqRowQuerier 池与事务共用的单行查询能力(*pgxpool.Pool / pgx.Tx 均满足)。
type seqRowQuerier interface {
	QueryRow(ctx context.Context, sql string, args ...any) pgx.Row
}

// nextFacilityCode 池/事务共用取号实现:按 spec.offset/digits 精确切序号段,
// 只在同 kind(+同网格)集合内取 MAX+1;超容量回 ErrGridFull。
func nextFacilityCode(ctx context.Context, q seqRowQuerier, kind string, gridCode int16) (string, error) {
	spec, err := facilitySeqSpec(kind, gridCode)
	if err != nil {
		return "", err
	}
	var maxSeq int
	if err := q.QueryRow(ctx,
		`SELECT COALESCE(MAX(SUBSTRING(code FROM $2 FOR $3)::int),0)
			FROM odn_facility WHERE code ~ $1`,
		spec.pattern, spec.offset, spec.digits).Scan(&maxSeq); err != nil {
		return "", fmt.Errorf("odn: next facility code %s: %w", kind, err)
	}
	if maxSeq+1 > spec.limit {
		return "", fmt.Errorf("%w: %s 编号已用尽", ErrGridFull, kind)
	}
	return FormatFacilityCode(kind, gridCode, maxSeq+1), nil
}

// NextFacilityCode 下一可用设施编码(现存 MAX+1;超容量回 ErrGridFull)。
func (s *PGStore) NextFacilityCode(ctx context.Context, kind string, gridCode int16) (string, error) {
	return nextFacilityCode(ctx, s.db, kind, gridCode)
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
