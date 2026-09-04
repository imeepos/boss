package monthly

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"

	"github.com/jackc/pgx/v5"
)

// UpsertUserRevenue 单行 upsert(PUT 与导入共用写入口)。
func (s *PGStore) UpsertUserRevenue(ctx context.Context, r UserRevenue) error {
	return s.upsertRegionFact(ctx, TableUserRevenue, userRevenueRow(r))
}

// UpsertNetworkDelivery 单行 upsert。
func (s *PGStore) UpsertNetworkDelivery(ctx context.Context, r NetworkDelivery) error {
	return s.upsertRegionFact(ctx, TableNetworkDelivery, networkDeliveryRow(r))
}

// UpsertFinanceCost 单行 upsert。
func (s *PGStore) UpsertFinanceCost(ctx context.Context, r FinanceCost) error {
	return s.upsertRegionFact(ctx, TableFinanceCost, financeCostRow(r))
}

// upsertRegionFact 校验(月份/区域/非负)+白名单复核 + 幂等 upsert。
func (s *PGStore) upsertRegionFact(ctx context.Context, kind string, r csvRow) error {
	if err := validateFact(kind, r); err != nil {
		return err
	}
	ok, err := s.regionActive(ctx, r.Region)
	if err != nil {
		return err
	}
	if !ok {
		return fmt.Errorf("monthly: %w: region %q 不在白名单", ErrInvalidInput, r.Region)
	}
	return s.upsertFact(ctx, kind, r)
}

// regionActive 区域是否在白名单且 active。
func (s *PGStore) regionActive(ctx context.Context, region string) (bool, error) {
	var ok bool
	err := s.db.QueryRow(ctx, `SELECT active FROM monthly_regions WHERE region = $1`, region).Scan(&ok)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, fmt.Errorf("monthly: region lookup: %w", err)
	}
	return ok, nil
}

// upsertFact 通用幂等写:PK(month,region) 冲突整行更新;
// INSERT 列不含派生生成列——外部派生值天然无入口(服务端计算铁律)。
func (s *PGStore) upsertFact(ctx context.Context, kind string, r csvRow) error {
	spec, err := specOf(kind)
	if err != nil {
		return err
	}
	cols := append([]string{"month", "region"}, spec.dbCols...)
	args := make([]any, 0, len(cols))
	args = append(args, r.Month, r.Region)
	for _, v := range r.Values {
		args = append(args, v)
	}
	ph := make([]string, len(cols))
	for i := range ph {
		ph[i] = "$" + strconv.Itoa(i+1)
	}
	sets := make([]string, 0, len(spec.dbCols))
	for _, c := range spec.dbCols {
		sets = append(sets, c+"=EXCLUDED."+c)
	}
	q := "INSERT INTO " + spec.table + " (" + strings.Join(cols, ", ") + ") VALUES (" + strings.Join(ph, ",") + ")" +
		" ON CONFLICT (month, region) DO UPDATE SET " + strings.Join(sets, ", ") + ", updated_at = now()"
	if _, err := s.db.Exec(ctx, q, args...); err != nil {
		return fmt.Errorf("monthly: upsert %s: %w", spec.table, err)
	}
	return nil
}
