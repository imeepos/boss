package odn

// 分光容量模型存储(W5,审查 F1 分光比建模;迁移 000221):
// odn_resource_chain 暂存的一级/二级分光比回写建模为设备级容量事实(幂等全量重建)。
// 设备解析优先级:城市域同名**唯一**设备优先(容量随城市维度进城市行)→ 城市域跨城同名
// 歧义不建模(不猜填)→ 无城市域登记再退导入域影子设备(无城市,只进全网/设备视图);
// 解析不到的编码计数并留 [odn-split-backfill] 日志(fields.md 1.5.15)。

import (
	"context"
	"fmt"
	"log"
)

// splitCapacityListSQL 容量清单(设备维度;导入域 prv/city NULL→JSON null)。
var splitCapacityListSQL = `
SELECT sc.device_id, d.code, d.kind, sc.split_level, sc.ratio, sc.chain_rows, sc.used_ports,
	d.prv_code, d.city_prefix, d.lifecycle_status, sc.has_secondary
FROM odn_device_split_capacity sc
JOIN odn_device d ON d.id = sc.device_id
ORDER BY d.prv_code NULLS LAST, d.city_prefix NULLS LAST, d.code`

// SplitCapacity 容量读视图;容量表未部署(000221 代差)→ 空报告(items 空表非错误)。
func (s *PGStore) SplitCapacity(ctx context.Context) (*SplitCapacityReport, error) {
	ok, err := s.hasTable(ctx, "odn_device_split_capacity")
	if err != nil {
		return nil, err
	}
	if !ok {
		return &SplitCapacityReport{Items: []SplitCapacityRow{}}, nil
	}
	rows, err := s.db.Query(ctx, splitCapacityListSQL)
	if err != nil {
		return nil, fmt.Errorf("odn: split capacity query: %w", err)
	}
	defer rows.Close()
	report := &SplitCapacityReport{Items: []SplitCapacityRow{}}
	for rows.Next() {
		var r SplitCapacityRow
		var prv, city any
		if err := rows.Scan(&r.DeviceID, &r.Code, &r.Kind, &r.SplitLevel, &r.Ratio,
			&r.ChainRows, &r.UsedPorts, &prv, &city, &r.LifecycleStatus, &r.HasSecondary); err != nil {
			return nil, fmt.Errorf("odn: split capacity scan: %w", err)
		}
		r.PrvCode = scanNullText(prv)
		r.CityPrefix = scanNullText(city)
		r.Expandable = r.Ratio - r.UsedPorts
		p, c := homesPotential(r.SplitLevel, r.HasSecondary, r.Ratio, r.UsedPorts)
		report.Summary.Devices++
		report.Summary.PotentialHomes += p
		report.Summary.ConnectedHomes += c
		report.Items = append(report.Items, r)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("odn: split capacity rows: %w", err)
	}
	report.Summary.ExpandableHomes = report.Summary.PotentialHomes - report.Summary.ConnectedHomes
	return report, nil
}

// scanNullText NULL 文本列 → nil 指针(导入域设备无城市)。
func scanNullText(v any) *string {
	s, ok := v.(string)
	if !ok {
		return nil
	}
	return &s
}

// chainSplitAgg 单箱体编码的链行聚合事实(SQL GROUP BY 产物,split_level 为判别列)。
type chainSplitAgg struct {
	code         string
	level        int // 1=一级(OBD) 2=二级(SBD)
	ratio        int
	chainRows    int
	usedPorts    int
	hasSecondary bool
	deviceID     int64
}

// chainSplitAggSQL 链行按箱体编码聚合(有分光比才可建模容量,无比率的行不参与);
// used_ports=端口标签去重计数(空标签不计;不猜填红线:不解析标签格式);
// 一级段 has_secondary=BOOL_OR(链行带 SBD),户级口径据此跳过下挂二级链的一级器。
var chainSplitAggSQL = `
SELECT obd_code, 1, MAX(split1_ratio), COUNT(*), COUNT(DISTINCT NULLIF(split1_port, '')),
	BOOL_OR(sbd_code IS NOT NULL)
FROM odn_resource_chain
WHERE obd_code IS NOT NULL AND split1_ratio IS NOT NULL
GROUP BY obd_code
UNION ALL
SELECT sbd_code, 2, MAX(split2_ratio), COUNT(*), COUNT(DISTINCT NULLIF(split2_port, '')), FALSE
FROM odn_resource_chain
WHERE sbd_code IS NOT NULL AND split2_ratio IS NOT NULL
GROUP BY sbd_code`

// BackfillSplitCapacity 幂等全量重建:SQL 聚合→设备解析→事务 DELETE+INSERT。
// 链表未部署(W3 代差)→ 空聚合(容量表清空,返回零值);任何库错上抛并留日志。
func (s *PGStore) BackfillSplitCapacity(ctx context.Context) (*SplitBackfillResult, error) {
	aggs, err := s.aggregateChainSplits(ctx)
	if err != nil {
		log.Printf("[odn-split-backfill] AGG FAILED err=%v", err)
		return nil, err
	}
	rows := make([]chainSplitAgg, 0, len(aggs))
	res := &SplitBackfillResult{}
	for _, a := range aggs {
		deviceID, derr := s.resolveSplitDevice(ctx, a)
		if derr != nil {
			res.UnresolvedCodes++
			log.Printf("[odn-split-backfill] UNRESOLVED code=%s level=%d reason=%v", a.code, a.level, derr)
			continue
		}
		a.deviceID = deviceID
		rows = append(rows, a)
	}
	if err := s.replaceSplitCapacity(ctx, rows); err != nil {
		log.Printf("[odn-split-backfill] REPLACE FAILED err=%v", err)
		return nil, err
	}
	for _, a := range rows {
		res.DevicesModeled++
		if a.level == 1 {
			res.Level1++
		} else {
			res.Level2++
		}
	}
	log.Printf("[odn-split-backfill] DONE devices=%d level1=%d level2=%d unresolved=%d",
		res.DevicesModeled, res.Level1, res.Level2, res.UnresolvedCodes)
	return res, nil
}

// aggregateChainSplits 链行聚合;链表未建(W3 代差)→ 空聚合非错误。
func (s *PGStore) aggregateChainSplits(ctx context.Context) ([]chainSplitAgg, error) {
	ok, err := s.hasTable(ctx, "odn_resource_chain")
	if err != nil {
		return nil, err
	}
	if !ok {
		return []chainSplitAgg{}, nil
	}
	rows, err := s.db.Query(ctx, chainSplitAggSQL)
	if err != nil {
		return nil, fmt.Errorf("odn: chain split agg query: %w", err)
	}
	defer rows.Close()
	out := []chainSplitAgg{}
	for rows.Next() {
		var a chainSplitAgg
		if err := rows.Scan(&a.code, &a.level, &a.ratio, &a.chainRows, &a.usedPorts, &a.hasSecondary); err != nil {
			return nil, fmt.Errorf("odn: chain split agg scan: %w", err)
		}
		out = append(out, a)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("odn: chain split agg rows: %w", err)
	}
	return out, nil
}

// resolveSplitDevice 箱体编码→设备 id(裁定:城市域同名**唯一**优先,导入域兜底):
// 操作员在城市域实名登记的箱体是物理本体,链行指向同一物理体,容量随城市维度进城市行;
// 城市域同名多行=跨城歧义,不定城不建模(不猜填);无城市域登记再退导入域影子设备(无城市)。
func (s *PGStore) resolveSplitDevice(ctx context.Context, a chainSplitAgg) (int64, error) {
	kind := "OBD"
	if a.level == 2 {
		kind = "SBD"
	}
	var n int
	if err := s.db.QueryRow(ctx,
		"SELECT count(*) FROM odn_device WHERE code=$1 AND kind=$2 AND prv_code IS NOT NULL AND status <> 'RETIRED'",
		a.code, kind).Scan(&n); err != nil {
		return 0, fmt.Errorf("odn: resolve city-domain count: %w", err)
	}
	var id int64
	var err error
	if n == 1 {
		err = s.db.QueryRow(ctx,
			"SELECT id FROM odn_device WHERE code=$1 AND kind=$2 AND prv_code IS NOT NULL AND status <> 'RETIRED' ORDER BY id LIMIT 1",
			a.code, kind).Scan(&id)
		if err == nil {
			return id, nil
		}
		return 0, fmt.Errorf("odn: resolve city-domain device: %w", err)
	}
	if n > 1 {
		return 0, fmt.Errorf("odn: code %s ambiguous in %d cities", a.code, n)
	}
	err = s.db.QueryRow(ctx,
		"SELECT id FROM odn_device WHERE code=$1 AND kind=$2 AND prv_code IS NULL AND status <> 'RETIRED' ORDER BY id LIMIT 1",
		a.code, kind).Scan(&id)
	if err != nil {
		return 0, fmt.Errorf("odn: no %s device for code %s", kind, a.code)
	}
	return id, nil
}

// replaceSplitCapacity 事务内全量重建(幂等);defer 兜底回滚,提交后 no-op。
func (s *PGStore) replaceSplitCapacity(ctx context.Context, rows []chainSplitAgg) error {
	tx, err := s.db.Begin(ctx)
	if err != nil {
		return fmt.Errorf("odn: split backfill begin: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if _, err := tx.Exec(ctx, "DELETE FROM odn_device_split_capacity"); err != nil {
		return fmt.Errorf("odn: split backfill delete: %w", err)
	}
	for _, a := range rows {
		if _, err := tx.Exec(ctx,
			"INSERT INTO odn_device_split_capacity (device_id, split_level, ratio, chain_rows, used_ports, has_secondary) VALUES ($1,$2,$3,$4,$5,$6)",
			a.deviceID, a.level, a.ratio, a.chainRows, a.usedPorts, a.hasSecondary); err != nil {
			return fmt.Errorf("odn: split backfill insert %s: %w", a.code, err)
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return fmt.Errorf("odn: split backfill commit: %w", err)
	}
	return nil
}
